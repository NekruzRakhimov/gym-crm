package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidToken = errors.New("invalid token")

type AuthService struct {
	adminRepo        repository.AdminRepository
	refreshTokenRepo repository.RefreshTokenRepository
	accessSecret     string
	refreshSecret    string
}

func NewAuthService(
	adminRepo repository.AdminRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	accessSecret, refreshSecret string,
) *AuthService {
	return &AuthService{adminRepo, refreshTokenRepo, accessSecret, refreshSecret}
}

func (s *AuthService) Login(ctx context.Context, input models.LoginInput) (accessToken, rawRefreshToken string, expiresAt time.Time, err error) {
	admin, err := s.adminRepo.GetByUsername(ctx, input.Username)
	if err != nil {
		return "", "", time.Time{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(input.Password)); err != nil {
		return "", "", time.Time{}, ErrInvalidCredentials
	}

	accessToken, err = s.generateAccessToken(admin.ID, admin.Username, admin.Role)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate access token: %w", err)
	}

	rawRefreshToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}

	expiresAt = time.Now().Add(30 * 24 * time.Hour)
	if err := s.refreshTokenRepo.Create(ctx, admin.ID, tokenHash, expiresAt); err != nil {
		return "", "", time.Time{}, fmt.Errorf("store refresh token: %w", err)
	}

	return accessToken, rawRefreshToken, expiresAt, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (string, error) {
	hash := hashToken(rawToken)
	rt, err := s.refreshTokenRepo.GetByHash(ctx, hash)
	if err != nil {
		return "", ErrInvalidToken
	}
	if time.Now().After(rt.ExpiresAt) {
		_ = s.refreshTokenRepo.DeleteByHash(ctx, hash)
		return "", ErrInvalidToken
	}

	admin, err := s.adminRepo.GetByID(ctx, rt.AdminID)
	if err != nil {
		return "", ErrInvalidToken
	}

	return s.generateAccessToken(admin.ID, admin.Username, admin.Role)
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	hash := hashToken(rawToken)
	return s.refreshTokenRepo.DeleteByHash(ctx, hash)
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &models.Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.accessSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*models.Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) generateAccessToken(adminID int, username, role string) (string, error) {
	claims := models.Claims{
		AdminID:  adminID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.accessSecret))
}

func generateRefreshToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	hash = hashToken(raw)
	return raw, hash, nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
