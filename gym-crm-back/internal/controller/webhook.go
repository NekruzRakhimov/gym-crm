package controller

import (
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/service"
)

type WebhookController struct {
	accessSvc *service.AccessService
}

func NewWebhookController(accessSvc *service.AccessService) *WebhookController {
	return &WebhookController{accessSvc}
}

// Actual flat format sent by Hikvision DS-K1T344MBFWX-E1
type hikvisionFlatEvent struct {
	DateTime  string `json:"dateTime"`
	EventType string `json:"eventType"`
	AccessControllerEvent *struct {
		EmployeeNoString  string `json:"employeeNoString"`
		MajorEventType    int    `json:"majorEventType"`
		SubEventType      int    `json:"subEventType"`
		CurrentVerifyMode string `json:"currentVerifyMode"`
		Name              string `json:"name"`
		SerialNo          int    `json:"serialNo"`
		CardReaderNo      int    `json:"cardReaderNo"`
		// RemoteCheck=true means terminal is waiting for our allow/deny response.
		RemoteCheck bool `json:"remoteCheck"`
	} `json:"AccessControllerEvent"`
}

func authMethodFromVerifyMode(mode string) string {
	mode = strings.ToLower(mode)
	switch {
	case strings.Contains(mode, "face"):
		return "face"
	case strings.Contains(mode, "fp"):
		return "fingerprint"
	case strings.Contains(mode, "card"):
		return "card"
	case strings.Contains(mode, "pw"):
		return "pin"
	default:
		return "unknown"
	}
}

func (h *WebhookController) Handle(c *gin.Context) {
	terminalID, err := strconv.Atoi(c.Param("terminal_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid terminal_id"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
		return
	}

	ct := c.GetHeader("Content-Type")
	var jsonData []byte // only the JSON part, stored in DB

	if strings.Contains(ct, "multipart/form-data") {
		boundary := ""
		for _, part := range strings.Split(ct, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "boundary=") {
				boundary = strings.TrimPrefix(part, "boundary=")
			}
		}
		if boundary != "" {
			mr := multipart.NewReader(strings.NewReader(string(rawBody)), boundary)
			for {
				p, err := mr.NextPart()
				if err != nil {
					break
				}
				partData, _ := io.ReadAll(p)
				// Use first part that is valid JSON
				if json.Valid(partData) {
					jsonData = partData
					break
				}
			}
		}
	} else {
		jsonData = rawBody
	}

	if len(jsonData) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	// Log every incoming payload so we can debug unexpected formats.
	log.Printf("webhook raw [terminal=%d ct=%s]: %s", terminalID, ct, string(jsonData))

	var evt hikvisionFlatEvent
	_ = json.Unmarshal(jsonData, &evt)

	// Skip heartbeats and non-access events
	if evt.EventType != "AccessControllerEvent" {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	// Skip events without a person (e.g. door open events)
	if evt.AccessControllerEvent == nil || evt.AccessControllerEvent.EmployeeNoString == "" {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	// Skip remote verification events — they are already processed by the /verify endpoint
	if evt.AccessControllerEvent.RemoteCheck {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	ace := evt.AccessControllerEvent
	authMethod := authMethodFromVerifyMode(ace.CurrentVerifyMode)
	employeeNo := ace.EmployeeNoString

	// Regular post-facto event notification (no response verdict needed).
	eventTime, err := time.Parse(time.RFC3339, evt.DateTime)
	if err != nil {
		eventTime = time.Now()
	}

	log.Printf("webhook: terminal=%d employee=%s method=%s name=%s", terminalID, employeeNo, authMethod, ace.Name)

	_, err = h.accessSvc.ProcessEvent(
		c.Request.Context(),
		terminalID,
		employeeNo,
		eventTime,
		authMethod,
		jsonData,
	)
	if err != nil {
		log.Printf("process event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// AcsRemoteVerifyReqInfo is the payload Hikvision sends for Remote Verification.
type acsRemoteVerifyReq struct {
	AcsRemoteVerifyReqInfo *struct {
		EmployeeNo  string `json:"employeeNo"`
		FPID        string `json:"FPID"`        // face person ID (sometimes used instead)
		VerifyMode  string `json:"verifyMode"`
		SerialNo    int    `json:"serialNo"`
		ReqType     string `json:"reqType"`
	} `json:"AcsRemoteVerifyReqInfo"`
}

type acsRemoteVerifyResp struct {
	AcsRemoteVerify struct {
		SerialNo       int    `json:"serialNo"`
		DoorIndex      int    `json:"doorIndex"`
		Status         string `json:"status"` // "normal" = open, "notOpen" = deny
		Msg            string `json:"msg"`
		CardNo         string `json:"cardNo"`
		Password       string `json:"password"`
		UserVerifyMode string `json:"userVerifyMode"`
	} `json:"AcsRemoteVerify"`
}

// Verify handles Remote Verification requests from Hikvision terminals.
// The terminal sends auth info here and waits for our allow/deny response.
func (h *WebhookController) Verify(c *gin.Context) {
	terminalID, err := strconv.Atoi(c.Param("terminal_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid terminal_id"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
		return
	}

	log.Printf("verify raw [terminal=%d]: %s", terminalID, string(rawBody))

	var req acsRemoteVerifyReq
	_ = json.Unmarshal(rawBody, &req)

	var employeeNo, verifyMode string
	if req.AcsRemoteVerifyReqInfo != nil {
		employeeNo = req.AcsRemoteVerifyReqInfo.EmployeeNo
		if employeeNo == "" {
			employeeNo = req.AcsRemoteVerifyReqInfo.FPID
		}
		verifyMode = req.AcsRemoteVerifyReqInfo.VerifyMode
	}

	authMethod := authMethodFromVerifyMode(verifyMode)
	log.Printf("remote verify: terminal=%d employee=%s method=%s", terminalID, employeeNo, authMethod)

	granted, reason, err := h.accessSvc.Verify(
		c.Request.Context(),
		terminalID,
		employeeNo,
		authMethod,
		rawBody,
	)
	if err != nil {
		log.Printf("remote verify error: %v", err)
	}

	resp := acsRemoteVerifyResp{}
	resp.AcsRemoteVerify.DoorIndex = 1
	if granted {
		resp.AcsRemoteVerify.Status = "normal"
	} else {
		resp.AcsRemoteVerify.Status = "notOpen"
		resp.AcsRemoteVerify.Msg = reason
	}

	log.Printf("remote verify: terminal=%d employee=%s granted=%v reason=%s", terminalID, employeeNo, granted, reason)
	c.JSON(http.StatusOK, resp)
}
