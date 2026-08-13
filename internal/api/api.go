// Package api implements the real MySQL-backed JSON API (auth + CRUD) used in
// production. It is self-contained (own db/models/handlers) and intentionally
// separate from the experimental controller/service/repo scaffold.
//
// All responses use the shared envelope {code, message, data}.
package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db     *sql.DB
	secret []byte
}

func NewHandler(db *sql.DB, secret []byte) *Handler {
	return &Handler{db: db, secret: secret}
}

// ---- domain models (JSON matches the frontend types) ----

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // income | expense
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

type Transaction struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // income | expense
	Amount     int64  `json:"amount"`
	CategoryID string `json:"categoryId"`
	Note       string `json:"note"`
	Date       string `json:"date"` // yyyy-mm-dd
}

type Budget struct {
	ID         string `json:"id"`
	CategoryID string `json:"categoryId"`
	Limit      int64  `json:"limit"`
	Month      string `json:"month"` // yyyy-mm
}

// ---- response helpers ----

func ok(c *gin.Context, data any) {
	c.JSON(200, gin.H{"code": 20000, "message": "OK", "data": data})
}

func fail(c *gin.Context, status, code int, message string) {
	c.JSON(status, gin.H{"code": code, "message": message, "data": nil})
	c.Abort()
}
