package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/jmoiron/sqlx"
)

type ClientTariffRepository interface {
	Assign(ctx context.Context, clientID int, input models.AssignTariffInput, endDate time.Time, paidAmount float64) (*models.ClientTariff, error)
	GetActive(ctx context.Context, clientID int) (*models.ClientTariffDetail, error)
	ListByClient(ctx context.Context, clientID int) ([]models.ClientTariffDetail, error)
	HasExpired(ctx context.Context, clientID int) (bool, error)
	Delete(ctx context.Context, clientID, tariffRecordID int) error
}

type clientTariffRepo struct{ db *sqlx.DB }

func NewClientTariffRepository(db *sqlx.DB) ClientTariffRepository {
	return &clientTariffRepo{db}
}

func (r *clientTariffRepo) Assign(ctx context.Context, clientID int, input models.AssignTariffInput, endDate time.Time, paidAmount float64) (*models.ClientTariff, error) {
	startDate, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return nil, fmt.Errorf("parse start_date: %w", err)
	}

	var ct models.ClientTariff
	err = r.db.QueryRowxContext(ctx,
		`INSERT INTO client_tariffs(client_id, tariff_id, start_date, end_date, paid_amount)
		 VALUES($1,$2,$3,$4,$5) RETURNING *`,
		clientID, input.TariffID, startDate, endDate, paidAmount,
	).StructScan(&ct)
	if err != nil {
		return nil, fmt.Errorf("assign tariff: %w", err)
	}
	return &ct, nil
}

func (r *clientTariffRepo) GetActive(ctx context.Context, clientID int) (*models.ClientTariffDetail, error) {
	var d models.ClientTariffDetail
	err := r.db.QueryRowxContext(ctx, `
		SELECT ct.*, t.name AS tariff_name, t.duration_days, t.max_visit_days,
		       t.schedule_days, t.time_from, t.time_to
		FROM client_tariffs ct
		JOIN tariffs t ON t.id = ct.tariff_id
		WHERE ct.client_id=$1
		  AND ct.start_date <= CURRENT_DATE
		  AND ct.end_date >= CURRENT_DATE
		ORDER BY ct.id DESC
		LIMIT 1
	`, clientID).StructScan(&d)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active tariff: %w", err)
	}
	return &d, nil
}

func (r *clientTariffRepo) ListByClient(ctx context.Context, clientID int) ([]models.ClientTariffDetail, error) {
	var ds []models.ClientTariffDetail
	err := r.db.SelectContext(ctx, &ds, `
		SELECT ct.*, t.name AS tariff_name, t.duration_days, t.max_visit_days,
		       t.schedule_days, t.time_from, t.time_to
		FROM client_tariffs ct
		JOIN tariffs t ON t.id = ct.tariff_id
		WHERE ct.client_id=$1
		ORDER BY ct.id DESC
	`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list client tariffs: %w", err)
	}
	return ds, nil
}

func (r *clientTariffRepo) Delete(ctx context.Context, clientID, tariffRecordID int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Nullify FK reference in transactions so the delete doesn't fail
	if _, err := tx.ExecContext(ctx,
		"UPDATE transactions SET client_tariff_id=NULL WHERE client_tariff_id=$1",
		tariffRecordID,
	); err != nil {
		return fmt.Errorf("nullify transactions: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		"DELETE FROM client_tariffs WHERE id=$1 AND client_id=$2",
		tariffRecordID, clientID,
	)
	if err != nil {
		return fmt.Errorf("delete tariff: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tariff record not found")
	}
	return tx.Commit()
}

func (r *clientTariffRepo) HasExpired(ctx context.Context, clientID int) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM client_tariffs WHERE client_id=$1 AND end_date < CURRENT_DATE",
		clientID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check expired: %w", err)
	}
	return count > 0, nil
}
