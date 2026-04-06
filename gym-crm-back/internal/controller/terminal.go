package controller

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/clients"
	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
	"github.com/gym-crm/gym-crm-back/internal/service"
)

type TerminalController struct {
	terminalRepo repository.TerminalRepository
	syncSvc      *service.SyncService
	serverIP     string
	serverPort   int
}

func NewTerminalController(
	terminalRepo repository.TerminalRepository,
	syncSvc *service.SyncService,
	serverIP string,
	serverPort int,
) *TerminalController {
	return &TerminalController{terminalRepo, syncSvc, serverIP, serverPort}
}

func (h *TerminalController) List(c *gin.Context) {
	ts, err := h.terminalRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ts)
}

func (h *TerminalController) Create(c *gin.Context) {
	var input models.CreateTerminalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.terminalRepo.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Sync all clients to new terminal asynchronously
	go h.syncSvc.SyncAllClientsToTerminal(c.Request.Context(), t.ID)
	c.JSON(http.StatusCreated, t)
}

func (h *TerminalController) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var input models.UpdateTerminalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.terminalRepo.Update(c.Request.Context(), id, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *TerminalController) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.terminalRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *TerminalController) GetStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	t, err := h.terminalRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	hik := clients.NewHikvisionClient(t.IP, t.Port, t.Username, t.Password)
	online := hik.Ping() == nil
	c.JSON(http.StatusOK, gin.H{"online": online})
}

func (h *TerminalController) OpenDoor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	t, err := h.terminalRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	hik := clients.NewHikvisionClient(t.IP, t.Port, t.Username, t.Password)
	if err := hik.OpenDoor(1); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "door opened"})
}

func (h *TerminalController) SetupWebhook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	t, err := h.terminalRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	hik := clients.NewHikvisionClient(t.IP, t.Port, t.Username, t.Password)
	if err := hik.SetupWebhook(h.serverIP, h.serverPort, t.ID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "webhook configured"})
}

func (h *TerminalController) Sync(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	go h.syncSvc.SyncAllClientsToTerminal(c.Request.Context(), id)
	c.JSON(http.StatusOK, gin.H{"message": "sync started"})
}

func (h *TerminalController) EnableRemoteVerification(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	t, err := h.terminalRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	webhookURL := fmt.Sprintf("http://%s:%d/api/webhooks/hikvision/%d", h.serverIP, h.serverPort, t.ID)
	// AcsCfgNormal is only accessible via web session auth, not Digest ISAPI.
	// Return manual instructions instead.
	c.JSON(http.StatusOK, gin.H{
		"manual": true,
		"steps": []string{
			fmt.Sprintf("Откройте https://%s в браузере", t.IP),
			"Перейдите: Access Control → Terminal Parameters → Remote Verification",
			"Включите Remote Verification (toggle ON)",
			"Verifying Person Type: Normal User ✓",
			"Result Return Mode: Sync",
			"Нажмите Save",
		},
		"webhook_url": webhookURL,
		"note":        "Сервер уже готов принимать запросы верификации",
	})
	log.Printf("remote-verify info for terminal %d (%s): webhook=%s", t.ID, t.IP, webhookURL)
}
