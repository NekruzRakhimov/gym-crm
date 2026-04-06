package repository

import (
	"context"
	"fmt"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/jmoiron/sqlx"
)

type TerminalRepository interface {
	List(ctx context.Context) ([]models.Terminal, error)
	ListActive(ctx context.Context) ([]models.Terminal, error)
	GetByID(ctx context.Context, id int) (*models.Terminal, error)
	Create(ctx context.Context, input models.CreateTerminalInput) (*models.Terminal, error)
	Update(ctx context.Context, id int, input models.UpdateTerminalInput) (*models.Terminal, error)
	Delete(ctx context.Context, id int) error
}

type terminalRepo struct{ db *sqlx.DB }

func NewTerminalRepository(db *sqlx.DB) TerminalRepository {
	return &terminalRepo{db}
}

func (r *terminalRepo) List(ctx context.Context) ([]models.Terminal, error) {
	var ts []models.Terminal
	if err := r.db.SelectContext(ctx, &ts, "SELECT * FROM terminals ORDER BY id"); err != nil {
		return nil, fmt.Errorf("list terminals: %w", err)
	}
	return ts, nil
}

func (r *terminalRepo) ListActive(ctx context.Context) ([]models.Terminal, error) {
	var ts []models.Terminal
	if err := r.db.SelectContext(ctx, &ts, "SELECT * FROM terminals WHERE active=true ORDER BY id"); err != nil {
		return nil, fmt.Errorf("list active terminals: %w", err)
	}
	return ts, nil
}

func (r *terminalRepo) GetByID(ctx context.Context, id int) (*models.Terminal, error) {
	var t models.Terminal
	if err := r.db.GetContext(ctx, &t, "SELECT * FROM terminals WHERE id=$1", id); err != nil {
		return nil, fmt.Errorf("get terminal: %w", err)
	}
	return &t, nil
}

func (r *terminalRepo) Create(ctx context.Context, input models.CreateTerminalInput) (*models.Terminal, error) {
	port := input.Port
	if port == 0 {
		port = 80
	}
	var t models.Terminal
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO terminals(name, ip, port, username, password, direction)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING *`,
		input.Name, input.IP, port, input.Username, input.Password, input.Direction,
	).StructScan(&t)
	if err != nil {
		return nil, fmt.Errorf("create terminal: %w", err)
	}
	return &t, nil
}

func (r *terminalRepo) Update(ctx context.Context, id int, input models.UpdateTerminalInput) (*models.Terminal, error) {
	port := input.Port
	if port == 0 {
		port = 80
	}
	var t models.Terminal
	var err error
	if input.Password != "" {
		err = r.db.QueryRowxContext(ctx,
			`UPDATE terminals SET name=$1, ip=$2, port=$3, username=$4, password=$5, direction=$6
			 WHERE id=$7 RETURNING *`,
			input.Name, input.IP, port, input.Username, input.Password, input.Direction, id,
		).StructScan(&t)
	} else {
		err = r.db.QueryRowxContext(ctx,
			`UPDATE terminals SET name=$1, ip=$2, port=$3, username=$4, direction=$5
			 WHERE id=$6 RETURNING *`,
			input.Name, input.IP, port, input.Username, input.Direction, id,
		).StructScan(&t)
	}
	if err != nil {
		return nil, fmt.Errorf("update terminal: %w", err)
	}
	return &t, nil
}

func (r *terminalRepo) Delete(ctx context.Context, id int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"UPDATE access_events SET terminal_id=NULL WHERE terminal_id=$1", id,
	); err != nil {
		return fmt.Errorf("nullify access_events: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM terminals WHERE id=$1", id,
	); err != nil {
		return fmt.Errorf("delete terminal: %w", err)
	}

	return tx.Commit()
}
