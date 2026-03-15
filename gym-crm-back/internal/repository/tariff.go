package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/jmoiron/sqlx"
)

type TariffRepository interface {
	List(ctx context.Context) ([]models.Tariff, error)
	GetByID(ctx context.Context, id int) (*models.Tariff, error)
	Create(ctx context.Context, input models.CreateTariffInput) (*models.Tariff, error)
	Update(ctx context.Context, id int, input models.CreateTariffInput) (*models.Tariff, error)
	Delete(ctx context.Context, id int) error
	ToggleActive(ctx context.Context, id int) (*models.Tariff, error)
}

type tariffRepo struct{ db *sqlx.DB }

func NewTariffRepository(db *sqlx.DB) TariffRepository {
	return &tariffRepo{db}
}

func (r *tariffRepo) List(ctx context.Context) ([]models.Tariff, error) {
	var ts []models.Tariff
	if err := r.db.SelectContext(ctx, &ts, "SELECT * FROM tariffs ORDER BY id"); err != nil {
		return nil, fmt.Errorf("list tariffs: %w", err)
	}
	return ts, nil
}

func (r *tariffRepo) GetByID(ctx context.Context, id int) (*models.Tariff, error) {
	var t models.Tariff
	if err := r.db.GetContext(ctx, &t, "SELECT * FROM tariffs WHERE id=$1", id); err != nil {
		return nil, fmt.Errorf("get tariff: %w", err)
	}
	return &t, nil
}

var validDayAbbr = map[string]bool{
	"mon": true, "tue": true, "wed": true, "thu": true,
	"fri": true, "sat": true, "sun": true,
}

func scheduleDays(s string) string {
	switch s {
	case "all", "weekdays", "weekends", "even", "odd":
		return s
	default:
		// Accept comma-separated weekday abbreviations: "mon,wed,fri"
		parts := strings.Split(s, ",")
		for _, p := range parts {
			if !validDayAbbr[strings.TrimSpace(p)] {
				return "all"
			}
		}
		return s
	}
}

func (r *tariffRepo) Create(ctx context.Context, input models.CreateTariffInput) (*models.Tariff, error) {
	var t models.Tariff
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO tariffs(name, duration_days, max_visits_per_day, price, schedule_days, time_from, time_to)
		 VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING *`,
		input.Name, input.DurationDays, input.MaxVisitsPerDay, input.Price,
		scheduleDays(input.ScheduleDays), input.TimeFrom, input.TimeTo,
	).StructScan(&t)
	if err != nil {
		return nil, fmt.Errorf("create tariff: %w", err)
	}
	return &t, nil
}

func (r *tariffRepo) Update(ctx context.Context, id int, input models.CreateTariffInput) (*models.Tariff, error) {
	var t models.Tariff
	err := r.db.QueryRowxContext(ctx,
		`UPDATE tariffs SET name=$1, duration_days=$2, max_visits_per_day=$3, price=$4,
		  schedule_days=$5, time_from=$6, time_to=$7
		 WHERE id=$8 RETURNING *`,
		input.Name, input.DurationDays, input.MaxVisitsPerDay, input.Price,
		scheduleDays(input.ScheduleDays), input.TimeFrom, input.TimeTo, id,
	).StructScan(&t)
	if err != nil {
		return nil, fmt.Errorf("update tariff: %w", err)
	}
	return &t, nil
}

func (r *tariffRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tariffs WHERE id=$1", id)
	return err
}

func (r *tariffRepo) ToggleActive(ctx context.Context, id int) (*models.Tariff, error) {
	var t models.Tariff
	err := r.db.QueryRowxContext(ctx,
		"UPDATE tariffs SET active=NOT active WHERE id=$1 RETURNING *", id,
	).StructScan(&t)
	if err != nil {
		return nil, fmt.Errorf("toggle tariff: %w", err)
	}
	return &t, nil
}
