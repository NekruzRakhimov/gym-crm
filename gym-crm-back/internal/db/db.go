package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func Connect(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func RunMigrations(db *sqlx.DB, migrationsDir string) error {
	// ensure schema_migrations table exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(50) PRIMARY KEY,
		applied_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		version := filepath.Base(f)
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version=$1", version).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if count > 0 {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations(version) VALUES($1)", version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		log.Printf("applied migration: %s", version)
	}
	return nil
}

func SeedAdmin(db *sqlx.DB, username, password string) error {
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM admins").Scan(&count); err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	username = strings.TrimSpace(username)
	if _, err := db.ExecContext(context.Background(),
		"INSERT INTO admins(username, password_hash, role) VALUES($1, $2, 'admin')",
		username, string(hash),
	); err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}
	log.Printf("created admin user: %s", username)
	return nil
}
