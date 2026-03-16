package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/jmoiron/sqlx"
)

type ClientRepository interface {
	List(ctx context.Context, search string, page, limit int) ([]models.ClientWithTariff, int, error)
	GetByID(ctx context.Context, id int) (*models.Client, error)
	Create(ctx context.Context, input models.CreateClientInput) (*models.Client, error)
	Update(ctx context.Context, id int, input models.UpdateClientInput) (*models.Client, error)
	UpdatePhoto(ctx context.Context, id int, photoPath string) error
	SetActive(ctx context.Context, id int, active bool) error
	Delete(ctx context.Context, id int) error
}

type clientRepo struct{ db *sqlx.DB }

func NewClientRepository(db *sqlx.DB) ClientRepository {
	return &clientRepo{db}
}

func (r *clientRepo) List(ctx context.Context, search string, page, limit int) ([]models.ClientWithTariff, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	cond := ""
	args := []interface{}{}
	argIdx := 1

	if strings.TrimSpace(search) != "" {
		cond = fmt.Sprintf(" WHERE c.full_name ILIKE $%d OR c.phone ILIKE $%d", argIdx, argIdx+1)
		like := "%" + search + "%"
		args = append(args, like, like)
		argIdx += 2
	}

	countQuery := `SELECT COUNT(*) FROM clients c` + cond
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count clients: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT c.*,
		       t.name AS active_tariff_name,
		       ct.end_date AS active_tariff_end
		FROM clients c
		LEFT JOIN client_tariffs ct ON ct.client_id = c.id
		    AND ct.start_date <= CURRENT_DATE AND ct.end_date >= CURRENT_DATE
		LEFT JOIN tariffs t ON t.id = ct.tariff_id
		%s
		ORDER BY c.id DESC
		LIMIT $%d OFFSET $%d
	`, cond, argIdx, argIdx+1)
	args = append(args, limit, offset)

	var clients []models.ClientWithTariff
	if err := r.db.SelectContext(ctx, &clients, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list clients: %w", err)
	}
	return clients, total, nil
}

func (r *clientRepo) GetByID(ctx context.Context, id int) (*models.Client, error) {
	var c models.Client
	if err := r.db.GetContext(ctx, &c, "SELECT * FROM clients WHERE id=$1", id); err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	return &c, nil
}

func (r *clientRepo) Create(ctx context.Context, input models.CreateClientInput) (*models.Client, error) {
	var c models.Client
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO clients(full_name, phone, card_number) VALUES($1,$2,$3) RETURNING *`,
		input.FullName, input.Phone, input.CardNumber,
	).StructScan(&c)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	return &c, nil
}

func (r *clientRepo) Update(ctx context.Context, id int, input models.UpdateClientInput) (*models.Client, error) {
	var c models.Client
	err := r.db.QueryRowxContext(ctx,
		`UPDATE clients SET full_name=$1, phone=$2, card_number=$3 WHERE id=$4 RETURNING *`,
		input.FullName, input.Phone, input.CardNumber, id,
	).StructScan(&c)
	if err != nil {
		return nil, fmt.Errorf("update client: %w", err)
	}
	return &c, nil
}

func (r *clientRepo) UpdatePhoto(ctx context.Context, id int, photoPath string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE clients SET photo_path=$1 WHERE id=$2", photoPath, id)
	return err
}

func (r *clientRepo) SetActive(ctx context.Context, id int, active bool) error {
	_, err := r.db.ExecContext(ctx, "UPDATE clients SET is_active=$1 WHERE id=$2", active, id)
	return err
}

func (r *clientRepo) Delete(ctx context.Context, id int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM transactions WHERE client_id=$1", id); err != nil {
		return fmt.Errorf("delete transactions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM client_tariffs WHERE client_id=$1", id); err != nil {
		return fmt.Errorf("delete client_tariffs: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE access_events SET client_id=NULL WHERE client_id=$1", id); err != nil {
		return fmt.Errorf("nullify access_events: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM clients WHERE id=$1", id); err != nil {
		return fmt.Errorf("delete client: %w", err)
	}

	return tx.Commit()
}
