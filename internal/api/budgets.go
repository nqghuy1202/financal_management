package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var monthRe = regexp.MustCompile(`^\d{4}-\d{2}$`)

type budgetInput struct {
	CategoryID string `json:"categoryId"`
	Limit      int64  `json:"limit"`
	Month      string `json:"month"`
}

func (h *Handler) ListBudgets(c *gin.Context) {
	rows, err := h.db.Query(
		`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = ?`,
		userIDFrom(c),
	)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50030, "Không thể tải ngân sách")
		return
	}
	defer rows.Close()

	list := make([]Budget, 0)
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.Limit, &b.Month); err != nil {
			fail(c, http.StatusInternalServerError, 50031, "Lỗi đọc ngân sách")
			return
		}
		list = append(list, b)
	}
	ok(c, list)
}

// UpsertBudget sets the limit for a (category, month); it inserts or updates the
// existing row (unique per user+category+month).
func (h *Handler) UpsertBudget(c *gin.Context) {
	var in budgetInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40030, "Dữ liệu không hợp lệ")
		return
	}
	if in.CategoryID == "" || in.Limit <= 0 || !monthRe.MatchString(in.Month) {
		fail(c, http.StatusBadRequest, 40031, "Danh mục, hạn mức và tháng là bắt buộc")
		return
	}
	uid := userIDFrom(c)
	if !h.ownsCategory(uid, in.CategoryID) {
		fail(c, http.StatusBadRequest, 40032, "Danh mục không tồn tại")
		return
	}

	id := uuid.NewString()
	if _, err := h.db.Exec(
		`INSERT INTO budgets (id, user_id, category_id, limit_amount, month)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE limit_amount = VALUES(limit_amount)`,
		id, uid, in.CategoryID, in.Limit, in.Month,
	); err != nil {
		fail(c, http.StatusInternalServerError, 50032, "Không thể lưu ngân sách")
		return
	}

	// Return the canonical row (id may be the existing one on update).
	var b Budget
	if err := h.db.QueryRow(
		`SELECT id, category_id, limit_amount, month FROM budgets
		 WHERE user_id = ? AND category_id = ? AND month = ?`,
		uid, in.CategoryID, in.Month,
	).Scan(&b.ID, &b.CategoryID, &b.Limit, &b.Month); err != nil {
		// Fallback to what we inserted
		b = Budget{ID: id, CategoryID: in.CategoryID, Limit: in.Limit, Month: in.Month}
	}
	ok(c, b)
}

func (h *Handler) DeleteBudget(c *gin.Context) {
	if _, err := h.db.Exec(
		`DELETE FROM budgets WHERE id = ? AND user_id = ?`,
		c.Param("id"), userIDFrom(c),
	); err != nil {
		fail(c, http.StatusInternalServerError, 50033, "Không thể xóa ngân sách")
		return
	}
	ok(c, gin.H{"deleted": true})
}
