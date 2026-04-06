package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/service"
)

const refreshCookieName = "refresh_token"

type AuthController struct {
	authSvc *service.AuthService
}

func NewAuthController(authSvc *service.AuthService) *AuthController {
	return &AuthController{authSvc}
}

func (h *AuthController) Login(c *gin.Context) {
	var input models.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, rawRefresh, expiresAt, err := h.authSvc.Login(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    rawRefresh,
		Path:     "/api/auth",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false, // set true in production with HTTPS
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})

	c.JSON(http.StatusOK, models.LoginResponse{AccessToken: accessToken})
}

func (h *AuthController) Refresh(c *gin.Context) {
	rawToken, err := c.Cookie(refreshCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	accessToken, err := h.authSvc.Refresh(c.Request.Context(), rawToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{AccessToken: accessToken})
}

func (h *AuthController) Logout(c *gin.Context) {
	rawToken, err := c.Cookie(refreshCookieName)
	if err == nil {
		_ = h.authSvc.Logout(c.Request.Context(), rawToken)
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/auth",
		MaxAge:   -1,
		HttpOnly: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
