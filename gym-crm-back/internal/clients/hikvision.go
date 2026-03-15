package clients

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
	transport := &digest.Transport{
		Username: username,
		Password: password,
	}
	return &HikvisionClient{
		BaseURL:  fmt.Sprintf("http://%s:%d", ip, port),
		Username: username,
		Password: password,
		HTTP: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
	}
}

// doRead performs a GET and returns the raw response body.
func (c *HikvisionClient) doRead(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
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

func (c *HikvisionClient) do(method, path string, body io.Reader, contentType string) error {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
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

func (c *HikvisionClient) UpsertPerson(clientID int, fullName string, validEndTime time.Time) error {
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

	// Try Modify (update existing user) first.
	// If user doesn't exist on the terminal, fall back to SetUp (create).
	if err := c.do("PUT", "/ISAPI/AccessControl/UserInfo/Modify?format=json",
		strings.NewReader(body), "application/json"); err == nil {
		return nil
	}
	return c.do("POST", "/ISAPI/AccessControl/UserInfo/SetUp?format=json",
		strings.NewReader(body), "application/json")
}

func (c *HikvisionClient) UploadFace(clientID int, jpegData []byte) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// JSON part
	dataPart, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="FaceDataRecord"`},
		"Content-Type":        []string{"application/json"},
	})
	if err != nil {
		return fmt.Errorf("create data part: %w", err)
	}
	fmt.Fprintf(dataPart, `{"faceLibType":"blackFD","FDID":"1","FPID":"%d"}`, clientID)

	// Image part
	imgPart, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="img"; filename="face.jpg"`},
		"Content-Type":        []string{"image/jpeg"},
	})
	if err != nil {
		return fmt.Errorf("create img part: %w", err)
	}
	if _, err := imgPart.Write(jpegData); err != nil {
		return fmt.Errorf("write img: %w", err)
	}
	mw.Close()

	return c.do("PUT", "/ISAPI/Intelligent/FDLib/FDSetUp?format=json",
		&buf, mw.FormDataContentType())
}

func (c *HikvisionClient) DeletePerson(clientID int) error {
	body := fmt.Sprintf(`{"UserInfoDelCond":{"EmployeeNoList":[{"employeeNo":"%d"}]}}`, clientID)
	return c.do("PUT", "/ISAPI/AccessControl/UserInfo/Delete?format=json",
		strings.NewReader(body), "application/json")
}

func (c *HikvisionClient) SetupWebhook(ourIP string, ourPort int, terminalID int) error {
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
	return c.do("PUT", "/ISAPI/Event/notification/httpHosts/1",
		strings.NewReader(body), "application/xml")
}

func (c *HikvisionClient) OpenDoor(doorNo int) error {
	body := `<?xml version='1.0' encoding='utf-8'?><RemoteControlDoor xmlns="http://www.isapi.org/ver20/XMLSchema" version="2.0"><cmd>open</cmd></RemoteControlDoor>`
	return c.do("PUT", fmt.Sprintf("/ISAPI/AccessControl/RemoteControl/door/%d", doorNo),
		strings.NewReader(body), "application/xml")
}

func (c *HikvisionClient) Ping() error {
	return c.do("GET", "/ISAPI/System/deviceInfo", nil, "")
}

// httpsClient builds a Digest-auth HTTP client that uses HTTPS with
// InsecureSkipVerify — needed for Hikvision terminals which use self-signed certs.
func (c *HikvisionClient) httpsClient() *http.Client {
	return &http.Client{
		Transport: &digest.Transport{
			Username: c.Username,
			Password: c.Password,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
		Timeout: 10 * time.Second,
	}
}

// EnableRemoteVerification configures the terminal to use sync remote
// verification mode. The terminal will POST each auth attempt to our
// webhook URL and wait for allow/deny before opening the door.
// Call after SetupWebhook so the HTTP host is already configured.
// Uses HTTPS port 443 because AcsCfgNormal is only exposed on HTTPS.
func (c *HikvisionClient) EnableRemoteVerification() error {
	// Derive HTTPS base URL from the terminal's IP (always port 443).
	ip := strings.TrimPrefix(strings.TrimPrefix(c.BaseURL, "http://"), "https://")
	if idx := strings.Index(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	httpsBase := fmt.Sprintf("https://%s", ip)
	httpsHTTP := c.httpsClient()

	doHTTPS := func(method, path string, body io.Reader, ct string) error {
		req, err := http.NewRequest(method, httpsBase+path, body)
		if err != nil {
			return fmt.Errorf("new request: %w", err)
		}
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		resp, err := httpsHTTP.Do(req)
		if err != nil {
			return fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return fmt.Errorf("hikvision HTTPS %s %s: status %d: %s", method, path, resp.StatusCode, string(b))
		}
		return nil
	}

	getHTTPS := func(path string) ([]byte, error) {
		req, err := http.NewRequest("GET", httpsBase+path, nil)
		if err != nil {
			return nil, fmt.Errorf("new request: %w", err)
		}
		resp, err := httpsHTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("hikvision HTTPS GET %s: status %d: %s", path, resp.StatusCode, string(b))
		}
		return b, nil
	}

	// Read current config to preserve all existing fields.
	current, err := getHTTPS("/ISAPI/AccessControl/AcsCfgNormal?format=json")
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

	return doHTTPS("PUT", "/ISAPI/AccessControl/AcsCfgNormal?format=json",
		bytes.NewReader(updated), "application/json")
}
