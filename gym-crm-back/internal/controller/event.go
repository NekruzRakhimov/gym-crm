package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
)

type EventController struct {
	eventRepo repository.AccessEventRepository
}

func NewEventController(eventRepo repository.AccessEventRepository) *EventController {
	return &EventController{eventRepo}
}

func (h *EventController) List(c *gin.Context) {
	var filter models.EventsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events, total, err := h.eventRepo.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": events, "total": total})
}
