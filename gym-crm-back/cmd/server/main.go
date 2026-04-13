package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gym-crm/gym-crm-back/internal/config"
	"github.com/gym-crm/gym-crm-back/internal/controller"
	"github.com/gym-crm/gym-crm-back/internal/db"
	"github.com/gym-crm/gym-crm-back/internal/repository"
	"github.com/gym-crm/gym-crm-back/internal/router"
	"github.com/gym-crm/gym-crm-back/internal/service"
)

func main() {
	cfg := config.Load()

	// Connect DB
	database, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := db.RunMigrations(database, "./migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// Seed admin
	if err := db.SeedAdmin(database, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	// Application-level context: cancelled on SIGINT/SIGTERM so background
	// goroutines (hub, scheduler) exit cleanly instead of leaking.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Repositories
	adminRepo := repository.NewAdminRepository(database)
	refreshTokenRepo := repository.NewRefreshTokenRepository(database)
	terminalRepo := repository.NewTerminalRepository(database)
	tariffRepo := repository.NewTariffRepository(database)
	clientRepo := repository.NewClientRepository(database)
	clientTariffRepo := repository.NewClientTariffRepository(database)
	eventRepo := repository.NewAccessEventRepository(database)

	// Services
	authSvc := service.NewAuthService(adminRepo, refreshTokenRepo, cfg.JWTAccessSecret, cfg.JWTRefreshSecret)
	hub := service.NewHub()
	go hub.Run(ctx)

	transactionRepo := repository.NewTransactionRepository(database)

	syncSvc := service.NewSyncService(terminalRepo, clientRepo, clientTariffRepo, cfg.UploadsDir)
	schedulerSvc := service.NewSchedulerService(clientRepo, clientTariffRepo, syncSvc)
	go schedulerSvc.Run(ctx)
	tariffSvc := service.NewTariffService(tariffRepo)
	clientSvc := service.NewClientService(clientRepo, clientTariffRepo, tariffRepo, transactionRepo, syncSvc, cfg.UploadsDir)
	accessSvc := service.NewAccessService(terminalRepo, clientRepo, clientTariffRepo, eventRepo, hub)

	// Controllers
	ctrls := router.Controllers{
		Auth:      controller.NewAuthController(authSvc),
		Client:    controller.NewClientController(clientSvc, eventRepo),
		Tariff:    controller.NewTariffController(tariffSvc),
		Event:     controller.NewEventController(eventRepo),
		Dashboard: controller.NewDashboardController(eventRepo),
		Terminal:  controller.NewTerminalController(terminalRepo, syncSvc, cfg.ServerIP, cfg.ServerPort),
		Webhook:   controller.NewWebhookController(accessSvc),
		WebSocket: controller.NewWebSocketController(hub, authSvc),
		Finance:   controller.NewFinanceController(transactionRepo),
		AdminUser: controller.NewAdminUserController(adminRepo),
	}

	r := router.Setup(authSvc, ctrls, cfg.FrontendDir)

	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		log.Printf("starting HTTPS server on %s", addr)
		if err := r.RunTLS(addr, cfg.TLSCert, cfg.TLSKey); err != nil {
			log.Fatalf("server: %v", err)
		}
	} else {
		log.Printf("starting HTTP server on %s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("server: %v", err)
		}
	}
}
