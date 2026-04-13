package clients

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/icholy/digest"
)

type HikvisionClient struct {
	BaseURL  string
	Username string
	Password string
	HTTP     *http.Client
}

func NewHikvisionClient(ip string, port int, username, password string) *HikvisionClient {
	// DS-K1T344 terminals only expose ISAPI on HTTPS (port 443).
	// The stored port is ignored for management calls; we always use 443.
	_ = port
	transport := &digest.Transport{
		Username: username,
		Password: password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
	return &HikvisionClient{
		BaseURL:  fmt.Sprintf("https://%s", ip),
		Username: username,
		Password: password,
		HTTP: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
	}
}

// doRead performs a GET and returns the raw response body.
// The provided context controls the lifetime of the request: if ctx is
// cancelled or its deadline elapses the request is aborted immediately.
func (c *HikvisionClient) doRead(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hikvision GET %s: status %d: %s", path, resp.StatusCode, string(b))
	}
	return b, nil
}

// do performs an arbitrary HTTP request against the terminal.
// The body is buffered upfront so icholy/digest can reliably replay it on the
// 401-challenge → authenticated-retry cycle without consuming the reader.
// The provided context controls cancellation for both the challenge request
// and the authenticated retry.
func (c *HikvisionClient) do(ctx context.Context, method, path string, body io.Reader, contentType string) error {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}
	}

	var req *http.Request
	var err error
	if len(bodyBytes) > 0 {
		req, err = http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(bodyBytes))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.BaseURL+path, nil)
	}
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hikvision %s %s: status %d: %s", method, path, resp.StatusCode, string(b))
	}
	return nil
}

// UpsertPerson creates or updates a person entry on the terminal.
// It tries Modify first (works for existing users), then SetUp PUT
// (DS-K1T344 firmware), then SetUp POST (fallback for other models).
func (c *HikvisionClient) UpsertPerson(ctx context.Context, clientID int, fullName string, validEndTime time.Time) error {
	body := fmt.Sprintf(`{
		"UserInfo": {
			"employeeNo": "%d",
			"name": %q,
			"userType": "normal",
			"Valid": {
				"enable": true,
				"beginTime": "2000-01-01T00:00:00",
				"endTime": "%s"
			},
			"doorRight": "1",
			"RightPlan": [{"doorNo": 1, "planTemplateNo": "1"}]
		}
	}`, clientID, fullName, validEndTime.Format("2006-01-02T15:04:05"))

	// 1. Try Modify (update existing user).
	if err := c.do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/Modify?format=json",
		strings.NewReader(body), "application/json"); err == nil {
		return nil
	}
	// 2. Try PUT /SetUp — DS-K1T344 firmware rejects POST but accepts PUT.
	if err := c.do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/SetUp?format=json",
		strings.NewReader(body), "application/json"); err == nil {
		return nil
	}
	// 3. Fallback: standard POST /SetUp for other models.
	return c.do(ctx, "POST", "/ISAPI/AccessControl/UserInfo/SetUp?format=json",
		strings.NewReader(body), "application/json")
}

// UploadFace sends the face photo to the terminal as a multipart upload.
// jpegData must be a valid JPEG image ≤200KB with a detectable face.
func (c *HikvisionClient) UploadFace(ctx context.Context, clientID int, jpegData []byte) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Part 1: JSON metadata
	metaPart, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="FaceDataRecord"`},
		"Content-Type":        {"application/json"},
	})
	if err != nil {
		return fmt.Errorf("create meta part: %w", err)
	}
	meta := fmt.Sprintf(`{"employeeNo":"%d","faceLibType":"blackFD","FDID":"1","FPID":"%d"}`, clientID, clientID)
	if _, err := metaPart.Write([]byte(meta)); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}

	// Part 2: JPEG binary
	imgPart, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="img"; filename="face.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	if err != nil {
		return fmt.Errorf("create img part: %w", err)
	}
	if _, err := imgPart.Write(jpegData); err != nil {
		return fmt.Errorf("write img: %w", err)
	}
	w.Close()

	return c.do(ctx, "POST", "/ISAPI/Intelligent/FDLib/FaceDataRecord?format=json",
		&buf, w.FormDataContentType())
}

func (c *HikvisionClient) DeletePerson(ctx context.Context, clientID int) error {
	body := fmt.Sprintf(`{"UserInfoDelCond":{"EmployeeNoList":[{"employeeNo":"%d"}]}}`, clientID)
	return c.do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/Delete?format=json",
		strings.NewReader(body), "application/json")
}

func (c *HikvisionClient) SetupWebhook(ctx context.Context, ourIP string, ourPort int, terminalID int) error {
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<HttpHostNotification>
<id>1</id>
<url>/api/webhooks/hikvision/%d</url>
<protocolType>HTTP</protocolType>
<parameterFormatType>JSON</parameterFormatType>
<addressingFormatType>ipaddress</addressingFormatType>
<ipAddress>%s</ipAddress>
<portNo>%d</portNo>
<httpAuthenticationMethod>none</httpAuthenticationMethod>
</HttpHostNotification>`, terminalID, ourIP, ourPort)
	return c.do(ctx, "PUT", "/ISAPI/Event/notification/httpHosts/1",
		strings.NewReader(body), "application/xml")
}

func (c *HikvisionClient) OpenDoor(ctx context.Context, doorNo int) error {
	body := `<?xml version='1.0' encoding='utf-8'?><RemoteControlDoor xmlns="http://www.isapi.org/ver20/XMLSchema" version="2.0"><cmd>open</cmd></RemoteControlDoor>`
	return c.do(ctx, "PUT", fmt.Sprintf("/ISAPI/AccessControl/RemoteControl/door/%d", doorNo),
		strings.NewReader(body), "application/xml")
}

// Ping checks device reachability by fetching basic device info.
func (c *HikvisionClient) Ping(ctx context.Context) error {
	return c.do(ctx, "GET", "/ISAPI/System/deviceInfo", nil, "")
}

// EnableRemoteVerification configures the terminal to use sync remote
// verification mode. The terminal will POST each auth attempt to our
// webhook URL and wait for allow/deny before opening the door.
// Call after SetupWebhook so the HTTP host is already configured.
func (c *HikvisionClient) EnableRemoteVerification(ctx context.Context) error {
	// Read current config to preserve all existing fields.
	current, err := c.doRead(ctx, "/ISAPI/AccessControl/AcsCfgNormal?format=json")
	if err != nil {
		return fmt.Errorf("read AcsCfgNormal: %w", err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(current, &cfg); err != nil {
		return fmt.Errorf("parse AcsCfgNormal: %w", err)
	}

	inner, ok := cfg["AcsCfgNormal"].(map[string]any)
	if !ok {
		inner = make(map[string]any)
		cfg["AcsCfgNormal"] = inner
	}

	inner["remoteCheckNonResident"] = true
	inner["remoteCheckNonResidentEnabled"] = true
	inner["remoteCheckMode"] = "sync"
	inner["remoteCheckTypeList"] = []string{"normal", "visitor", "unregistered"}

	updated, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal AcsCfgNormal: %w", err)
	}

	return c.do(ctx, "PUT", "/ISAPI/AccessControl/AcsCfgNormal?format=json",
		bytes.NewReader(updated), "application/json")
}
