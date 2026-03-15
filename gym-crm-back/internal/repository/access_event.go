package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/jmoiron/sqlx"
)

type AccessEventRepository interface {
	Create(ctx context.Context, event models.AccessEvent) (*models.AccessEvent, error)
	List(ctx context.Context, filter models.EventsFilter) ([]models.AccessEventDetail, int, error)
	ListByClient(ctx context.Context, clientID, page, limit int) ([]models.AccessEventDetail, int, error)
	GetDashboardStats(ctx context.Context) (models.DashboardStats, error)
	CountGrantedEntriesToday(ctx context.Context, clientID int) (int, error)
}

type accessEventRepo struct{ db *sqlx.DB }

func NewAccessEventRepository(db *sqlx.DB) AccessEventRepository {
	return &accessEventRepo{db}
}

func (r *accessEventRepo) Create(ctx context.Context, event models.AccessEvent) (*models.AccessEvent, error) {
	var e models.AccessEvent
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO access_events(client_id, terminal_id, direction, auth_method, access_granted, deny_reason, raw_event, event_time)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING *`,
		event.ClientID, event.TerminalID, event.Direction, event.AuthMethod,
		event.AccessGranted, event.DenyReason, event.RawEvent, event.EventTime,
	).StructScan(&e)
	if err != nil {
		return nil, fmt.Errorf("create access event: %w", err)
	}
	return &e, nil
}

func (r *accessEventRepo) List(ctx context.Context, filter models.EventsFilter) ([]models.AccessEventDetail, int, error) {
	page := filter.Page
	limit := filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	conditions := []string{}
	args := []interface{}{}
	argIdx := 1

	addArg := func(cond string, val interface{}) {
		conditions = append(conditions, fmt.Sprintf(cond, argIdx))
		args = append(args, val)
		argIdx++
	}

	if filter.From != "" {
		addArg("ae.event_time >= $%d", filter.From)
	}
	if filter.To != "" {
		addArg("ae.event_time <= $%d", filter.To)
	}
	if filter.ClientID != nil {
		addArg("ae.client_id = $%d", *filter.ClientID)
	}
	if filter.TerminalID != nil {
		addArg("ae.terminal_id = $%d", *filter.TerminalID)
	}
	if filter.Direction != "" {
		addArg("ae.direction = $%d", filter.Direction)
	}
	if filter.Granted != nil {
		addArg("ae.access_granted = $%d", *filter.Granted)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery := `FROM access_events ae
		LEFT JOIN clients c ON c.id = ae.client_id
		LEFT JOIN terminals t ON t.id = ae.terminal_id` + where

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	selectArgs := append(args, limit, offset)
	rows, err := r.db.QueryxContext(ctx, `
		SELECT ae.*, c.full_name AS client_name, c.photo_path AS client_photo, t.name AS terminal_name
		`+baseQuery+fmt.Sprintf(" ORDER BY ae.event_time DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1),
		selectArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []models.AccessEventDetail
	for rows.Next() {
		var e models.AccessEventDetail
		if err := rows.StructScan(&e); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, nil
}

func (r *accessEventRepo) ListByClient(ctx context.Context, clientID, page, limit int) ([]models.AccessEventDetail, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM access_events WHERE client_id=$1", clientID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count client events: %w", err)
	}

	rows, err := r.db.QueryxContext(ctx, `
		SELECT ae.*, c.full_name AS client_name, c.photo_path AS client_photo, t.name AS terminal_name
		FROM access_events ae
		LEFT JOIN clients c ON c.id = ae.client_id
		LEFT JOIN terminals t ON t.id = ae.terminal_id
		WHERE ae.client_id=$1
		ORDER BY ae.event_time DESC
		LIMIT $2 OFFSET $3
	`, clientID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list client events: %w", err)
	}
	defer rows.Close()

	var events []models.AccessEventDetail
	for rows.Next() {
		var e models.AccessEventDetail
		if err := rows.StructScan(&e); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, nil
}

func (r *accessEventRepo) GetDashboardStats(ctx context.Context) (models.DashboardStats, error) {
	var stats models.DashboardStats
	err := r.db.QueryRowContext(ctx, `
		WITH today AS (
			SELECT client_id, direction, access_granted, event_time
			FROM access_events
			WHERE event_time::date = CURRENT_DATE
		),
		last_per_client AS (
			SELECT DISTINCT ON (client_id)
				client_id, direction, access_granted
			FROM today
			WHERE client_id IS NOT NULL
			ORDER BY client_id, event_time DESC
		)
		SELECT
			(SELECT COUNT(*) FROM last_per_client
			 WHERE direction='entry' AND access_granted=true)        AS inside_now,
			(SELECT COALESCE(COUNT(*),0) FROM today WHERE direction='entry')        AS today_entries,
			(SELECT COALESCE(COUNT(*),0) FROM today WHERE direction='exit')         AS today_exits,
			(SELECT COALESCE(COUNT(*),0) FROM today WHERE access_granted=false)     AS today_denied
	`).Scan(&stats.InsideNow, &stats.TodayEntries, &stats.TodayExits, &stats.TodayDenied)
	if err != nil {
		return stats, fmt.Errorf("dashboard stats: %w", err)
	}
	return stats, nil
}

func (r *accessEventRepo) CountGrantedEntriesToday(ctx context.Context, clientID int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM access_events
		WHERE client_id=$1
		  AND direction='entry'
		  AND access_granted=true
		  AND event_time::date = CURRENT_DATE
	`, clientID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count entries today: %w", err)
	}
	return count, nil
}
