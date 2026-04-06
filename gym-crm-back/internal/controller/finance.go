package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/repository"
	"github.com/xuri/excelize/v2"
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

func (h *FinanceController) ExportExcel(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	ctx := c.Request.Context()

	stats, err := h.transactionRepo.GetFinanceStats(ctx, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	txRows, err := h.transactionRepo.ListTransactions(ctx, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	// ── Лист 1: Итоги ──────────────────────────────────────────────────────
	sheet1 := "Итоги"
	f.SetSheetName("Sheet1", sheet1)

	period := "За всё время"
	if from != "" && to != "" {
		period = from + " — " + to
	} else if from != "" {
		period = "С " + from
	} else if to != "" {
		period = "По " + to
	}

	summaryRows := [][]interface{}{
		{"Показатель", "Значение"},
		{"Период", period},
		{"Общая выручка (сомони)", stats.TotalRevenue},
		{"Всего клиентов", stats.TotalClients},
		{"Активных клиентов", stats.ActiveClients},
	}
	for i, row := range summaryRows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		f.SetSheetRow(sheet1, cell, &row)
	}

	// ── Лист 2: По дням / По месяцам ──────────────────────────────────────
	sheet2 := "По дням"
	if from == "" && to == "" {
		sheet2 = "По месяцам"
	}
	f.NewSheet(sheet2)

	colLabel := "Месяц"
	if from != "" || to != "" {
		colLabel = "День"
	}
	header2 := []interface{}{colLabel, "Выручка (сомони)"}
	f.SetSheetRow(sheet2, "A1", &header2)
	for i, m := range stats.MonthlyRevenue {
		row := []interface{}{m.Month, m.Revenue}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow(sheet2, cell, &row)
	}

	// ── Лист 3: Топ тарифы ────────────────────────────────────────────────
	sheet3 := "Топ тарифы"
	f.NewSheet(sheet3)

	header3 := []interface{}{"Тариф", "Продаж", "Выручка (сомони)"}
	f.SetSheetRow(sheet3, "A1", &header3)
	for i, t := range stats.TopTariffs {
		row := []interface{}{t.TariffName, t.Count, t.Revenue}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow(sheet3, cell, &row)
	}

	// ── Лист 4: Транзакции ────────────────────────────────────────────────
	sheet4 := "Транзакции"
	f.NewSheet(sheet4)

	header4 := []interface{}{"Дата", "Клиент", "Тип", "Сумма (сомони)", "Тариф", "Описание"}
	f.SetSheetRow(sheet4, "A1", &header4)
	for i, tx := range txRows {
		txType := "Пополнение"
		if tx.Type == "payment" {
			txType = "Оплата"
		}
		tariffName := ""
		if tx.TariffName != nil {
			tariffName = *tx.TariffName
		}
		desc := ""
		if tx.Description != nil {
			desc = *tx.Description
		}
		row := []interface{}{
			tx.CreatedAt.Format("2006-01-02 15:04"),
			tx.ClientName,
			txType,
			tx.Amount,
			tariffName,
			desc,
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow(sheet4, cell, &row)
	}

	filename := fmt.Sprintf("finance_%s.xlsx", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write excel"})
	}
}
