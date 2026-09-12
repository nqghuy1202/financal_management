package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type transactionInput struct {
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	CategoryID string `json:"categoryId"`
	Note       string `json:"note"`
	Date       string `json:"date"`
}

func (in transactionInput) validate() bool {
	if in.Type != "income" && in.Type != "expense" {
		return false
	}
	if in.Amount <= 0 || in.CategoryID == "" {
		return false
	}
	if _, err := time.Parse(dateLayout, in.Date); err != nil {
		return false
	}
	return true
}

func (h *Handler) ListTransactions(c *gin.Context) {
	list, err := h.transactions.List(c.Request.Context(), userIDFrom(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 50020, "Không thể tải giao dịch")
		return
	}
	ok(c, list)
}

func (h *Handler) CreateTransaction(c *gin.Context) {
	var in transactionInput
	if err := c.ShouldBindJSON(&in); err != nil || !in.validate() {
		fail(c, http.StatusBadRequest, 40020, "Dữ liệu giao dịch không hợp lệ")
		return
	}
	ctx := c.Request.Context()
	uid := userIDFrom(c)

	owns, err := h.categories.Owns(ctx, uid, in.CategoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50025, "Lỗi máy chủ")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40021, "Danh mục không tồn tại")
		return
	}

	t := Transaction{
		ID:         uuid.NewString(),
		Type:       in.Type,
		Amount:     in.Amount,
		CategoryID: in.CategoryID,
		Note:       strings.TrimSpace(in.Note),
		Date:       in.Date,
	}
	if err := h.transactions.Create(ctx, uid, t); err != nil {
		fail(c, http.StatusInternalServerError, 50022, "Không thể tạo giao dịch")
		return
	}
	ok(c, t)
}

func (h *Handler) UpdateTransaction(c *gin.Context) {
	var in transactionInput
	if err := c.ShouldBindJSON(&in); err != nil || !in.validate() {
		fail(c, http.StatusBadRequest, 40022, "Dữ liệu giao dịch không hợp lệ")
		return
	}
	ctx := c.Request.Context()
	uid := userIDFrom(c)
	id := c.Param("id")

	owns, err := h.categories.Owns(ctx, uid, in.CategoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50026, "Lỗi máy chủ")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40023, "Danh mục không tồn tại")
		return
	}

	t := Transaction{ID: id, Type: in.Type, Amount: in.Amount, CategoryID: in.CategoryID, Note: strings.TrimSpace(in.Note), Date: in.Date}
	found, err := h.transactions.Update(ctx, uid, id, t)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50023, "Không thể cập nhật giao dịch")
		return
	}
	if !found {
		fail(c, http.StatusNotFound, 40420, "Không tìm thấy giao dịch")
		return
	}
	ok(c, t)
}

func (h *Handler) DeleteTransaction(c *gin.Context) {
	if err := h.transactions.Delete(c.Request.Context(), userIDFrom(c), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, 50024, "Không thể xóa giao dịch")
		return
	}
	ok(c, gin.H{"deleted": true})
}
