package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Register mounts all API routes onto the given group (e.g. r.Group("/api")).
// Public: /auth/register, /auth/login. Everything else requires a valid JWT.
func (h *Handler) Register(rg *gin.RouterGroup) {
	// Throttle auth endpoints per IP to slow brute-force attempts.
	authLimit := newRateLimiter(10, time.Minute).middleware()
	rg.POST("/auth/register", authLimit, h.RegisterUser)
	rg.POST("/auth/login", authLimit, h.Login)
	rg.POST("/auth/demo", authLimit, h.Demo)

	auth := rg.Group("")
	auth.Use(h.AuthMiddleware())
	{
		auth.GET("/auth/me", h.Me)

		auth.GET("/categories", h.ListCategories)
		auth.POST("/categories", h.CreateCategory)
		auth.DELETE("/categories/:id", h.DeleteCategory)

		auth.GET("/transactions", h.ListTransactions)
		auth.POST("/transactions", h.CreateTransaction)
		auth.PUT("/transactions/:id", h.UpdateTransaction)
		auth.DELETE("/transactions/:id", h.DeleteTransaction)

		auth.GET("/budgets", h.ListBudgets)
		auth.POST("/budgets", h.UpsertBudget)
		auth.DELETE("/budgets/:id", h.DeleteBudget)
	}
}
