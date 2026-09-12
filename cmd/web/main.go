// Command web is the production entrypoint: it serves the JSON API under
// /api/* and the built React SPA (frontend/dist) with history-mode fallback.
//
// It is intentionally separate from the experimental entrypoints
// (cmd/api, cmd/server) so packaging/deploy does not depend on the
// half-finished learning scaffold. Configure via env: PORT (default 8080),
// STATIC_DIR (default ./frontend/dist), CORS_ORIGINS (comma-separated).
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"financal_management/internal/api"
	"financal_management/internal/pkg/response"
)

func main() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg := api.LoadConfig()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ---- API ----
	apiGroup := r.Group("/api")
	apiGroup.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.ResponseData{
			Code:    20000,
			Message: "OK",
			Data:    gin.H{"status": "up"},
		})
	})

	// Real MySQL-backed API (auth + CRUD). If the DB is unreachable the server
	// still boots and serves the SPA + /health; API routes will then 500.
	db, err := api.Connect(cfg.DB)
	if err != nil {
		log.Printf("WARNING: database unavailable, API disabled: %v", err)
	} else {
		if err := api.Migrate(db); err != nil {
			log.Printf("WARNING: migration failed: %v", err)
		}
		api.NewHandler(db, cfg.JWTSecret).Register(apiGroup)
		log.Println("API mounted at /api (MySQL connected)")
	}

	// ---- Static SPA ----
	// Vite emits: index.html, /assets/*, favicon.svg
	r.Static("/assets", filepath.Join(cfg.StaticDir, "assets"))
	r.StaticFile("/favicon.svg", filepath.Join(cfg.StaticDir, "favicon.svg"))
	indexFile := filepath.Join(cfg.StaticDir, "index.html")
	r.StaticFile("/", indexFile)

	// History-mode fallback: unknown non-API routes return index.html so the
	// client router can handle them; unknown API routes return a JSON 404.
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, response.ResponseData{
				Code:    40400,
				Message: "Not Found",
				Data:    nil,
			})
			return
		}
		c.File(indexFile)
	})

	log.Printf("web server listening on :%s (static: %s)", cfg.Port, cfg.StaticDir)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
