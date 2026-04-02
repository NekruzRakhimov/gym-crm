package router

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/controller"
	"github.com/gym-crm/gym-crm-back/internal/middleware"
	"github.com/gym-crm/gym-crm-back/internal/service"
)

type Controllers struct {
	Auth      *controller.AuthController
	Client    *controller.ClientController
	Tariff    *controller.TariffController
	Event     *controller.EventController
	Dashboard *controller.DashboardController
	Terminal  *controller.TerminalController
	Webhook   *controller.WebhookController
	WebSocket *controller.WebSocketController
	Finance   *controller.FinanceController
	AdminUser *controller.AdminUserController
}

func Setup(authSvc *service.AuthService, ctrls Controllers, frontendDir string) *gin.Engine {
	r := gin.Default()

	// CORS (needed only when frontend is served separately, e.g. during development)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Serve uploaded photos
	r.Static("/uploads", "./uploads")

	// Serve frontend static files
	r.Static("/assets", filepath.Join(frontendDir, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(frontendDir, "favicon.ico"))

	// WebSocket (JWT via query param)
	r.GET("/ws", ctrls.WebSocket.Handle)

	// Webhooks (no JWT - from Hikvision devices)
	r.POST("/api/webhooks/hikvision/:terminal_id", ctrls.Webhook.Handle)
	r.POST("/api/webhooks/hikvision/:terminal_id/verify", ctrls.Webhook.Verify)

	// Auth routes (no JWT)
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", ctrls.Auth.Login)
		auth.POST("/refresh", ctrls.Auth.Refresh)
		auth.POST("/logout", ctrls.Auth.Logout)
	}

	// Protected routes
	api := r.Group("/api", middleware.Auth(authSvc))
	{
		// Clients
		clients := api.Group("/clients")
		{
			clients.GET("", ctrls.Client.List)
			clients.POST("", ctrls.Client.Create)
			clients.GET("/:id", ctrls.Client.GetByID)
			clients.PUT("/:id", ctrls.Client.Update)
			clients.DELETE("/:id", middleware.RequireRole("admin"), ctrls.Client.Delete)
			clients.POST("/:id/photo", ctrls.Client.UploadPhoto)
			clients.POST("/:id/block", ctrls.Client.Block)
			clients.POST("/:id/unblock", ctrls.Client.Unblock)
			clients.GET("/:id/events", ctrls.Client.GetEvents)
			clients.GET("/:id/payments", ctrls.Client.GetPayments)
			clients.GET("/:id/active-tariff", ctrls.Client.GetActiveTariff)
			clients.POST("/:id/assign-tariff", ctrls.Client.AssignTariff)
			clients.DELETE("/:id/tariffs/:tariff_record_id", ctrls.Client.RevokeTariff)
			clients.POST("/:id/deposit", ctrls.Client.Deposit)
			clients.GET("/:id/transactions", ctrls.Client.GetTransactions)
		}

		// Tariffs
		api.GET("/tariffs", ctrls.Tariff.List) // all roles — needed for tariff assignment
		tariffs := api.Group("/tariffs", middleware.RequireRole("admin"))
		{
			tariffs.POST("", ctrls.Tariff.Create)
			tariffs.PUT("/:id", ctrls.Tariff.Update)
			tariffs.DELETE("/:id", ctrls.Tariff.Delete)
			tariffs.PATCH("/:id/toggle", ctrls.Tariff.ToggleActive)
		}

		// Events (admin only)
		api.GET("/events", middleware.RequireRole("admin"), ctrls.Event.List)

		// Dashboard (all authenticated roles)
		api.GET("/dashboard/stats", ctrls.Dashboard.GetStats)

		// Finance (admin only)
		api.GET("/finance/stats", middleware.RequireRole("admin"), ctrls.Finance.GetStats)

		// User management (admin only)
		users := api.Group("/users", middleware.RequireRole("admin"))
		{
			users.GET("", ctrls.AdminUser.List)
			users.POST("", ctrls.AdminUser.Create)
			users.DELETE("/:id", ctrls.AdminUser.Delete)
		}

		// Terminals (admin only)
		terminals := api.Group("/terminals", middleware.RequireRole("admin"))
		{
			terminals.GET("", ctrls.Terminal.List)
			terminals.POST("", ctrls.Terminal.Create)
			terminals.PUT("/:id", ctrls.Terminal.Update)
			terminals.DELETE("/:id", ctrls.Terminal.Delete)
			terminals.GET("/:id/status", ctrls.Terminal.GetStatus)
			terminals.POST("/:id/open-door", ctrls.Terminal.OpenDoor)
			terminals.POST("/:id/setup-webhook", ctrls.Terminal.SetupWebhook)
			terminals.POST("/:id/sync", ctrls.Terminal.Sync)
			terminals.POST("/:id/enable-remote-verify", ctrls.Terminal.EnableRemoteVerification)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// SPA catch-all: serve index.html for all non-API, non-static routes
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/ws") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.File(filepath.Join(frontendDir, "index.html"))
	})

	return r
}
