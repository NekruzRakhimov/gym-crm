package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/jmoiron/sqlx"
)

type TransactionRepository interface {
	Deposit(ctx context.Context, clientID int, amount float64, description string) (*models.Transaction, error)
	Payment(ctx context.Context, clientID int, amount float64, description string, tariffRecordID int) (*models.Transaction, error)
	ListByClient(ctx context.Context, clientID int) ([]models.Transaction, error)
	GetFinanceStats(ctx context.Context, from, to string) (*models.FinanceStats, error)
	ListTransactions(ctx context.Context, from, to string) ([]models.TransactionRow, error)
}

type transactionRepo struct{ db *sqlx.DB }

func NewTransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepo{db}
}

func (r *transactionRepo) Deposit(ctx context.Context, clientID int, amount float64, description string) (*models.Transaction, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"UPDATE clients SET balance = balance + $1 WHERE id = $2",
		amount, clientID,
	); err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	var t models.Transaction
	desc := &description
	if description == "" {
		desc = nil
	}
	if err := tx.QueryRowxContext(ctx,
		`INSERT INTO transactions(client_id, type, amount, description)
		 VALUES($1,'deposit',$2,$3) RETURNING *`,
		clientID, amount, desc,
	).StructScan(&t); err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	return &t, tx.Commit()
}

func (r *transactionRepo) Payment(ctx context.Context, clientID int, amount float64, description string, tariffRecordID int) (*models.Transaction, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Check sufficient balance
	var balance float64
	if err := tx.QueryRowContext(ctx,
		"SELECT balance FROM clients WHERE id=$1 FOR UPDATE",
		clientID,
	).Scan(&balance); err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}
	if balance < amount {
		return nil, fmt.Errorf("insufficient balance: %.2f < %.2f", balance, amount)
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE clients SET balance = balance - $1 WHERE id = $2",
		amount, clientID,
	); err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	var t models.Transaction
	desc := &description
	if description == "" {
		desc = nil
	}
	if err := tx.QueryRowxContext(ctx,
		`INSERT INTO transactions(client_id, type, amount, description, client_tariff_id)
		 VALUES($1,'payment',$2,$3,$4) RETURNING *`,
		clientID, amount, desc, tariffRecordID,
	).StructScan(&t); err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	return &t, tx.Commit()
}

func (r *transactionRepo) ListByClient(ctx context.Context, clientID int) ([]models.Transaction, error) {
	var ts []models.Transaction
	err := r.db.SelectContext(ctx, &ts,
		`SELECT * FROM transactions WHERE client_id=$1 ORDER BY created_at DESC`,
		clientID,
	)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return ts, nil
}

func (r *transactionRepo) GetFinanceStats(ctx context.Context, from, to string) (*models.FinanceStats, error) {
	var stats models.FinanceStats
	stats.MonthlyRevenue = []models.MonthlyRevenue{}
	stats.TopTariffs = []models.TariffRevenue{}

	// Build optional date filter clause
	// from/to are expected as "YYYY-MM-DD" or empty string
	dateFilter := ""
	args := []interface{}{}
	if from != "" && to != "" {
		dateFilter = " AND created_at >= $1 AND created_at < ($2::date + INTERVAL '1 day')"
		args = []interface{}{from, to}
	} else if from != "" {
		dateFilter = " AND created_at >= $1"
		args = []interface{}{from}
	} else if to != "" {
		dateFilter = " AND created_at < ($1::date + INTERVAL '1 day')"
		args = []interface{}{to}
	}

	if err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE type='deposit'`+dateFilter,
		args...,
	).Scan(&stats.TotalRevenue); err != nil {
		return nil, fmt.Errorf("total revenue: %w", err)
	}

	// For monthly breakdown: when a period is selected show daily breakdown, otherwise monthly
	if from != "" || to != "" {
		daily := []models.MonthlyRevenue{}
		if err := r.db.SelectContext(ctx, &daily,
			`SELECT TO_CHAR(created_at,'YYYY-MM-DD') AS month, COALESCE(SUM(amount),0) AS revenue
			 FROM transactions
			 WHERE type='deposit'`+dateFilter+`
			 GROUP BY month ORDER BY month DESC`,
			args...,
		); err != nil {
			return nil, fmt.Errorf("daily revenue: %w", err)
		}
		stats.MonthlyRevenue = daily
	} else {
		if err := r.db.SelectContext(ctx, &stats.MonthlyRevenue,
			`SELECT TO_CHAR(created_at,'YYYY-MM') AS month, COALESCE(SUM(amount),0) AS revenue
			 FROM transactions
			 WHERE type='deposit'
			 GROUP BY month ORDER BY month DESC LIMIT 12`,
		); err != nil {
			return nil, fmt.Errorf("monthly revenue: %w", err)
		}
	}

	// Top tariffs with date filter
	tariffDateFilter := strings.ReplaceAll(dateFilter, "created_at", "tr.created_at")
	if err := r.db.SelectContext(ctx, &stats.TopTariffs,
		`SELECT t.name AS tariff_name, COUNT(*) AS count, COALESCE(SUM(tr.amount),0) AS revenue
		 FROM transactions tr
		 JOIN client_tariffs ct ON ct.id = tr.client_tariff_id
		 JOIN tariffs t ON t.id = ct.tariff_id
		 WHERE tr.type='payment'`+tariffDateFilter+`
		 GROUP BY t.name ORDER BY revenue DESC LIMIT 10`,
		args...,
	); err != nil {
		return nil, fmt.Errorf("top tariffs: %w", err)
	}

	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active) FROM clients`,
	).Scan(&stats.TotalClients, &stats.ActiveClients); err != nil {
		return nil, fmt.Errorf("client counts: %w", err)
	}

	return &stats, nil
}

func (r *transactionRepo) ListTransactions(ctx context.Context, from, to string) ([]models.TransactionRow, error) {
	query := `
		SELECT
			tr.created_at,
			c.full_name AS client_name,
			tr.type,
			tr.amount,
			tr.description,
			t.name AS tariff_name
		FROM transactions tr
		JOIN clients c ON c.id = tr.client_id
		LEFT JOIN client_tariffs ct ON ct.id = tr.client_tariff_id
		LEFT JOIN tariffs t ON t.id = ct.tariff_id
		WHERE 1=1`

	args := []interface{}{}
	i := 1
	if from != "" {
		query += fmt.Sprintf(" AND tr.created_at >= $%d", i)
		args = append(args, from)
		i++
	}
	if to != "" {
		query += fmt.Sprintf(" AND tr.created_at < ($%d::date + INTERVAL '1 day')", i)
		args = append(args, to)
		i++
	}
	query += " ORDER BY tr.created_at DESC"

	var rows []models.TransactionRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return rows, nil
}
