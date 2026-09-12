package api

import (
	"bytes"
	"log"
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

// captureLog redirects the standard logger's output for the duration of fn
// and returns what was written.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)
	fn()
	return buf.String()
}

func TestLoadConfig_JWTSecret(t *testing.T) {
	t.Run("empty secret is substituted with a default and warns", func(t *testing.T) {
		clearConfigEnv(t)

		var cfg Config
		out := captureLog(t, func() { cfg = LoadConfig() })

		assert.Equal(t, []byte("dev-secret-change-me"), cfg.JWTSecret)
		assert.Contains(t, out, "WARNING")
	})

	t.Run("weak secret warns but keeps the literal value", func(t *testing.T) {
		clearConfigEnv(t)
		t.Setenv("JWT_SECRET", "my-dev-secret-value")

		var cfg Config
		out := captureLog(t, func() { cfg = LoadConfig() })

		assert.Equal(t, []byte("my-dev-secret-value"), cfg.JWTSecret)
		assert.Contains(t, out, "WARNING")
	})

	t.Run("another weak variant (change-me) also warns", func(t *testing.T) {
		clearConfigEnv(t)
		t.Setenv("JWT_SECRET", "please-change-me-in-prod")

		var cfg Config
		out := captureLog(t, func() { cfg = LoadConfig() })

		assert.Equal(t, []byte("please-change-me-in-prod"), cfg.JWTSecret)
		assert.Contains(t, out, "WARNING")
	})

	t.Run("strong secret produces no warning and is used as-is", func(t *testing.T) {
		clearConfigEnv(t)
		t.Setenv("JWT_SECRET", "a-genuinely-strong-random-secret")

		var cfg Config
		out := captureLog(t, func() { cfg = LoadConfig() })

		assert.Equal(t, []byte("a-genuinely-strong-random-secret"), cfg.JWTSecret)
		assert.Empty(t, out)
	})
}
