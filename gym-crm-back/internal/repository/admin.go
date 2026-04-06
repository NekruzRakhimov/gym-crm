package repository

import (
	"context"
	"fmt"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/jmoiron/sqlx"
)

type AdminRepository interface {
	GetByUsername(ctx context.Context, username string) (*models.Admin, error)
	GetByID(ctx context.Context, id int) (*models.Admin, error)
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, username, passwordHash string) error
	List(ctx context.Context) ([]models.Admin, error)
	Delete(ctx context.Context, id int) error
	CreateWithRole(ctx context.Context, username, passwordHash, role string) error
}

type adminRepo struct{ db *sqlx.DB }

func NewAdminRepository(db *sqlx.DB) AdminRepository {
	return &adminRepo{db}
}

func (r *adminRepo) GetByUsername(ctx context.Context, username string) (*models.Admin, error) {
	var a models.Admin
	if err := r.db.GetContext(ctx, &a, "SELECT * FROM admins WHERE username=$1", username); err != nil {
		return nil, fmt.Errorf("get admin by username: %w", err)
	}
	return &a, nil
}

func (r *adminRepo) GetByID(ctx context.Context, id int) (*models.Admin, error) {
	var a models.Admin
	if err := r.db.GetContext(ctx, &a, "SELECT * FROM admins WHERE id=$1", id); err != nil {
		return nil, fmt.Errorf("get admin by id: %w", err)
	}
	return &a, nil
}

func (r *adminRepo) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admins").Scan(&count); err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return count, nil
}

func (r *adminRepo) Create(ctx context.Context, username, passwordHash string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO admins(username, password_hash) VALUES($1, $2)",
		username, passwordHash,
	)
	return err
}

func (r *adminRepo) List(ctx context.Context) ([]models.Admin, error) {
	var admins []models.Admin
	if err := r.db.SelectContext(ctx, &admins, "SELECT * FROM admins ORDER BY id"); err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	return admins, nil
}

func (r *adminRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM admins WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete admin: %w", err)
	}
	return nil
}

func (r *adminRepo) CreateWithRole(ctx context.Context, username, passwordHash, role string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO admins(username, password_hash, role) VALUES($1, $2, $3)",
		username, passwordHash, role,
	)
	if err != nil {
		return fmt.Errorf("create admin with role: %w", err)
	}
	return nil
}
