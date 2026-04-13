package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
	"github.com/gym-crm/gym-crm-back/internal/service"
)

// terminalStatus holds the cached online/offline state for a single terminal.
// OFFLINE is only reported after two consecutive Ping failures (hysteresis),
// so a single slow response or a momentary busy state does not flip the UI.
type terminalStatus struct {
	online    bool
	failCount int
	checkedAt time.Time
}

// statusCacheTTL is how long a cached result is served without re-pinging.
// The frontend polls every 30 s, so 20 s keeps responses fresh while
// preventing every frontend request from hitting the terminal directly.
const statusCacheTTL = 20 * time.Second

type TerminalController struct {
	terminalRepo repository.TerminalRepository
	syncSvc      *service.SyncService
	serverIP     string
	serverPort   int

	statusMu    sync.Mutex
	statusCache map[int]*terminalStatus
}

func NewTerminalController(
	terminalRepo repository.TerminalRepository,
	syncSvc *service.SyncService,
	serverIP string,
	serverPort int,
) *TerminalController {
	return &TerminalController{
		terminalRepo: terminalRepo,
		syncSvc:      syncSvc,
		serverIP:     serverIP,
		serverPort:   serverPort,
		statusCache:  make(map[int]*terminalStatus),
	}
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
	// Sync all existing clients to the new terminal.
	// Use context.Background() — the gin request will finish before the sync does.
	go h.syncSvc.SyncAllClientsToTerminal(context.Background(), t.ID)
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
	// Credentials or IP may have changed — drop the cached client and status.
	h.syncSvc.InvalidateTerminalClient(id)
	h.invalidateStatus(id)
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
	h.syncSvc.InvalidateTerminalClient(id)
	h.invalidateStatus(id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetStatus returns the cached online/offline state of a terminal.
//
// Hysteresis rule: a terminal is only flipped to OFFLINE after two consecutive
// Ping failures. A single failed request (e.g. terminal busy processing a sync)
// keeps the previous state. This prevents the UI from flapping.
//
// The result is cached for statusCacheTTL so that the frontend's 30-second
// poll does not hit the terminal on every request.
//
// The shared HikvisionClient from SyncService is used so that Ping traffic
// shares the same http.Transport (and its connection pool) as sync operations,
// avoiding redundant TCP handshakes.
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

	// Serve from cache if still fresh.
	h.statusMu.Lock()
	st := h.statusCache[id]
	if st == nil {
		st = &terminalStatus{}
		h.statusCache[id] = st
	}
	if time.Since(st.checkedAt) < statusCacheTTL {
		online := st.online
		h.statusMu.Unlock()
		c.JSON(http.StatusOK, gin.H{"online": online})
		return
	}
	h.statusMu.Unlock()

	// Ping outside the lock to avoid blocking other status requests.
	// Use the shared HikvisionClient so Ping and sync operations share the
	// same http.Transport and do not open redundant TCP connections.
	hik := h.syncSvc.ClientForTerminal(*t)
	pingErr := hik.Ping(c.Request.Context())

	h.statusMu.Lock()
	defer h.statusMu.Unlock()

	// Re-fetch in case another goroutine already refreshed.
	st = h.statusCache[id]
	if st == nil {
		st = &terminalStatus{}
		h.statusCache[id] = st
	}

	st.checkedAt = time.Now()
	if pingErr == nil {
		st.online = true
		st.failCount = 0
	} else {
		st.failCount++
		// Require two consecutive failures before reporting OFFLINE.
		if st.failCount >= 2 {
			st.online = false
		}
		// failCount == 1: keep the previous online state (hysteresis).
	}

	c.JSON(http.StatusOK, gin.H{"online": st.online})
}

// invalidateStatus clears the cached status entry for a terminal.
func (h *TerminalController) invalidateStatus(terminalID int) {
	h.statusMu.Lock()
	defer h.statusMu.Unlock()
	delete(h.statusCache, terminalID)
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
	hik := h.syncSvc.ClientForTerminal(*t)
	if err := hik.OpenDoor(c.Request.Context(), 1); err != nil {
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
	hik := h.syncSvc.ClientForTerminal(*t)
	if err := hik.SetupWebhook(c.Request.Context(), h.serverIP, h.serverPort, t.ID); err != nil {
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
	// Use context.Background() — the gin request will finish well before
	// the full client sync completes.
	go h.syncSvc.SyncAllClientsToTerminal(context.Background(), id)
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

