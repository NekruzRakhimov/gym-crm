package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/jmoiron/sqlx"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, adminID int, tokenHash string, expiresAt time.Time) error
	GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	DeleteByHash(ctx context.Context, tokenHash string) error
	DeleteByAdminID(ctx context.Context, adminID int) error
}

type refreshTokenRepo struct{ db *sqlx.DB }

func NewRefreshTokenRepository(db *sqlx.DB) RefreshTokenRepository {
	return &refreshTokenRepo{db}
}

func (r *refreshTokenRepo) Create(ctx context.Context, adminID int, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens(admin_id, token_hash, expires_at) VALUES($1, $2, $3)",
		adminID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var t models.RefreshToken
	if err := r.db.GetContext(ctx, &t, "SELECT * FROM refresh_tokens WHERE token_hash=$1", tokenHash); err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &t, nil
}

func (r *refreshTokenRepo) DeleteByHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE token_hash=$1", tokenHash)
	return err
}

func (r *refreshTokenRepo) DeleteByAdminID(ctx context.Context, adminID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE admin_id=$1", adminID)
	return err
}
