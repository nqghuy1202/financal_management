package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

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
	rows, err := h.db.Query(
		`SELECT id, type, amount, COALESCE(category_id, ''), note, date
		 FROM transactions WHERE user_id = ? ORDER BY date DESC, created_at DESC`,
		userIDFrom(c),
	)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50020, "Không thể tải giao dịch")
		return
	}
	defer rows.Close()

	list := make([]Transaction, 0)
	for rows.Next() {
		var t Transaction
		var d time.Time
		if err := rows.Scan(&t.ID, &t.Type, &t.Amount, &t.CategoryID, &t.Note, &d); err != nil {
			fail(c, http.StatusInternalServerError, 50021, "Lỗi đọc giao dịch")
			return
		}
		t.Date = d.Format(dateLayout)
		list = append(list, t)
	}
	ok(c, list)
}

func (h *Handler) CreateTransaction(c *gin.Context) {
	var in transactionInput
	if err := c.ShouldBindJSON(&in); err != nil || !in.validate() {
		fail(c, http.StatusBadRequest, 40020, "Dữ liệu giao dịch không hợp lệ")
		return
	}
	uid := userIDFrom(c)
	if !h.ownsCategory(uid, in.CategoryID) {
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
	if _, err := h.db.Exec(
		`INSERT INTO transactions (id, user_id, type, amount, category_id, note, date)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, uid, t.Type, t.Amount, t.CategoryID, t.Note, t.Date,
	); err != nil {
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
	uid := userIDFrom(c)
	id := c.Param("id")
	if !h.ownsCategory(uid, in.CategoryID) {
		fail(c, http.StatusBadRequest, 40023, "Danh mục không tồn tại")
		return
	}

	res, err := h.db.Exec(
		`UPDATE transactions SET type = ?, amount = ?, category_id = ?, note = ?, date = ?
		 WHERE id = ? AND user_id = ?`,
		in.Type, in.Amount, in.CategoryID, strings.TrimSpace(in.Note), in.Date, id, uid,
	)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50023, "Không thể cập nhật giao dịch")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(c, http.StatusNotFound, 40420, "Không tìm thấy giao dịch")
		return
	}
	ok(c, Transaction{ID: id, Type: in.Type, Amount: in.Amount, CategoryID: in.CategoryID, Note: strings.TrimSpace(in.Note), Date: in.Date})
}

func (h *Handler) DeleteTransaction(c *gin.Context) {
	if _, err := h.db.Exec(
		`DELETE FROM transactions WHERE id = ? AND user_id = ?`,
		c.Param("id"), userIDFrom(c),
	); err != nil {
		fail(c, http.StatusInternalServerError, 50024, "Không thể xóa giao dịch")
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *Handler) ownsCategory(userID, categoryID string) bool {
	var one int
	err := h.db.QueryRow(
		`SELECT 1 FROM categories WHERE id = ? AND user_id = ?`, categoryID, userID,
	).Scan(&one)
	return err != sql.ErrNoRows && one == 1
}
