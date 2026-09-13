package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// configEnvVars lists every environment variable LoadConfig reads.
var configEnvVars = []string{
	"PORT", "STATIC_DIR", "CORS_ORIGINS", "JWT_SECRET",
	"BLUEPRINT_DB_HOST", "BLUEPRINT_DB_PORT", "BLUEPRINT_DB_DATABASE",
	"BLUEPRINT_DB_USERNAME", "BLUEPRINT_DB_PASSWORD",
}

// clearConfigEnv resets every env var LoadConfig reads to unset (via
// t.Setenv, so it's restored automatically at the end of the (sub)test) —
// each test then only sees the vars it explicitly sets itself.
func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, k := range configEnvVars {
		t.Setenv(k, "")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	clearConfigEnv(t)
	// A cleared JWT_SECRET exercises the "empty" substitution branch, tested
	// separately below — set a strong one here so this test only asserts
	// on the plain fallback defaults.
	t.Setenv("JWT_SECRET", "a-genuinely-strong-random-secret")

	cfg := LoadConfig()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "./frontend/dist", cfg.StaticDir)
	assert.Equal(t, []string{"http://localhost:5173"}, cfg.CORSOrigins)
	assert.Equal(t, DBConfig{Host: "localhost", Port: "3306", Database: "", Username: "", Password: ""}, cfg.DB)
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("STATIC_DIR", "/var/www")
	t.Setenv("BLUEPRINT_DB_HOST", "db.internal")
	t.Setenv("BLUEPRINT_DB_PORT", "3307")
	t.Setenv("BLUEPRINT_DB_DATABASE", "findb")
	t.Setenv("BLUEPRINT_DB_USERNAME", "admin")
	t.Setenv("BLUEPRINT_DB_PASSWORD", "secretpw")
	t.Setenv("JWT_SECRET", "a-genuinely-strong-random-secret")

	cfg := LoadConfig()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "/var/www", cfg.StaticDir)
	assert.Equal(t, DBConfig{
		Host: "db.internal", Port: "3307", Database: "findb",
		Username: "admin", Password: "secretpw",
	}, cfg.DB)
}

func TestLoadConfig_CORSOriginsCommaSplit(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("CORS_ORIGINS", "https://a.com,https://b.com,https://c.com")
	t.Setenv("JWT_SECRET", "a-genuinely-strong-random-secret")

	cfg := LoadConfig()

	assert.Equal(t, []string{"https://a.com", "https://b.com", "https://c.com"}, cfg.CORSOrigins)
}

// TestValidateJWTSecret covers the fail-fast rule LoadConfig applies via
// log.Fatalf (not exercised directly here — os.Exit would kill the test
// binary — so this pins the pure decision function instead).
func TestValidateJWTSecret(t *testing.T) {
	t.Run("empty secret is rejected", func(t *testing.T) {
		assert.Error(t, validateJWTSecret(""))
	})

	t.Run("dev-secret variant is rejected", func(t *testing.T) {
		assert.Error(t, validateJWTSecret("my-dev-secret-value"))
	})

	t.Run("change-me variant is rejected", func(t *testing.T) {
		assert.Error(t, validateJWTSecret("please-change-me-in-prod"))
	})

	t.Run("strong secret is accepted", func(t *testing.T) {
		assert.NoError(t, validateJWTSecret("a-genuinely-strong-random-secret"))
	})
}
