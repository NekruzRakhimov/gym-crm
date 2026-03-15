package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/service"
)

type WebSocketController struct {
	hub     *service.Hub
	authSvc *service.AuthService
}

func NewWebSocketController(hub *service.Hub, authSvc *service.AuthService) *WebSocketController {
	return &WebSocketController{hub, authSvc}
}

func (h *WebSocketController) Handle(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	if _, err := h.authSvc.ValidateAccessToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	h.hub.ServeWS(c.Writer, c.Request)
}
