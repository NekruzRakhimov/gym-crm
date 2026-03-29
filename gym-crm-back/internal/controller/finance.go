package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/repository"
)

type FinanceController struct {
	transactionRepo repository.TransactionRepository
}

func NewFinanceController(transactionRepo repository.TransactionRepository) *FinanceController {
	return &FinanceController{transactionRepo}
}

func (h *FinanceController) GetStats(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	stats, err := h.transactionRepo.GetFinanceStats(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
