package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL            string
	JWTAccessSecret  string
	JWTRefreshSecret string
	ServerIP         string
	ServerPort       int
	AdminUsername    string
	AdminPassword    string
	UploadsDir       string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		port = 8080
	}

	return &Config{
		DBURL:            getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/gym_crm?sslmode=disable"),
		JWTAccessSecret:  getEnv("JWT_ACCESS_SECRET", "access-secret"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "refresh-secret"),
		ServerIP:         getEnv("SERVER_IP", "127.0.0.1"),
		ServerPort:       port,
		AdminUsername:    getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", "admin"),
		UploadsDir:       getEnv("UPLOADS_DIR", "./uploads"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
