package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/repository"
)

type DashboardController struct {
	eventRepo repository.AccessEventRepository
}

func NewDashboardController(eventRepo repository.AccessEventRepository) *DashboardController {
	return &DashboardController{eventRepo}
}

func (h *DashboardController) GetStats(c *gin.Context) {
	stats, err := h.eventRepo.GetDashboardStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
