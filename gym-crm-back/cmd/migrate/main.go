package main

import (
	"log"

	"github.com/gym-crm/gym-crm-back/internal/config"
	"github.com/gym-crm/gym-crm-back/internal/db"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database, "./migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	log.Println("migrations applied successfully")
}
