package api

import (
	"log"
	"os"
	"strings"
)

// Config is the single source of runtime configuration for the web server
// and API. Load it once at process startup with LoadConfig and pass the
// result down explicitly, instead of reaching for os.Getenv from scattered
// packages.
type Config struct {
	Port        string
	StaticDir   string
	CORSOrigins []string
	JWTSecret   []byte
	DB          DBConfig
}

// DBConfig holds the MySQL connection parameters (BLUEPRINT_DB_* env vars).
type DBConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// LoadConfig reads all runtime configuration from the environment. Call it
// once at startup (see cmd/web/main.go).
func LoadConfig() Config {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" || strings.Contains(secret, "dev-secret") || strings.Contains(secret, "change-me") {
		log.Println("WARNING: JWT_SECRET yếu/mặc định - ĐẶT một chuỗi bí mật mạnh cho production!")
		if secret == "" {
			secret = "dev-secret-change-me"
		}
	}

	return Config{
		Port:        envOr("PORT", "8080"),
		StaticDir:   envOr("STATIC_DIR", "./frontend/dist"),
		CORSOrigins: strings.Split(envOr("CORS_ORIGINS", "http://localhost:5173"), ","),
		JWTSecret:   []byte(secret),
		DB: DBConfig{
			Host:     envOr("BLUEPRINT_DB_HOST", "localhost"),
			Port:     envOr("BLUEPRINT_DB_PORT", "3306"),
			Database: os.Getenv("BLUEPRINT_DB_DATABASE"),
			Username: os.Getenv("BLUEPRINT_DB_USERNAME"),
			Password: os.Getenv("BLUEPRINT_DB_PASSWORD"),
		},
	}
}
