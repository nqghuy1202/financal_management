package api

import (
	"fmt"
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

// validateJWTSecret rejects an empty secret or one still carrying a known
// placeholder marker ("dev-secret"/"change-me", e.g. .env.example's own
// default). This used to be a warn-and-substitute-a-fallback situation, but
// the source is public: anyone can read this exact fallback value and the
// token claim shape, so a deployment that never set a real secret would let
// anyone forge a valid JWT for any user_id. Refusing to start is the only
// safe response — a loud boot failure beats a silent hole.
func validateJWTSecret(secret string) error {
	if secret == "" {
		return fmt.Errorf("JWT_SECRET chưa được đặt")
	}
	if strings.Contains(secret, "dev-secret") || strings.Contains(secret, "change-me") {
		return fmt.Errorf("JWT_SECRET còn dùng giá trị mẫu (%q) — phải đổi sang một chuỗi bí mật thật", secret)
	}
	return nil
}

// LoadConfig reads all runtime configuration from the environment. Call it
// once at startup (see cmd/web/main.go). Exits the process via log.Fatalf if
// JWT_SECRET is missing or still a placeholder — see validateJWTSecret.
func LoadConfig() Config {
	secret := os.Getenv("JWT_SECRET")
	if err := validateJWTSecret(secret); err != nil {
		log.Fatalf("%v. Đặt một chuỗi ngẫu nhiên mạnh, ví dụ: openssl rand -hex 32", err)
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
