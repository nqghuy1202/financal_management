package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ListCategories(c *gin.Context) {
	rows, err := h.db.Query(
		`SELECT id, name, type, color, icon FROM categories WHERE user_id = ? ORDER BY created_at`,
		userIDFrom(c),
	)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50010, "Không thể tải danh mục")
		return
	}
	defer rows.Close()

	list := make([]Category, 0)
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Color, &cat.Icon); err != nil {
			fail(c, http.StatusInternalServerError, 50011, "Lỗi đọc danh mục")
			return
		}
		list = append(list, cat)
	}
	ok(c, list)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var in Category
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40010, "Dữ liệu không hợp lệ")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || (in.Type != "income" && in.Type != "expense") {
		fail(c, http.StatusBadRequest, 40011, "Tên và loại danh mục là bắt buộc")
		return
	}
	if in.Color == "" {
		in.Color = "#64748b"
	}
	if in.Icon == "" {
		in.Icon = "Tag"
	}
	in.ID = uuid.NewString()

	if _, err := h.db.Exec(
		`INSERT INTO categories (id, user_id, name, type, color, icon) VALUES (?, ?, ?, ?, ?, ?)`,
		in.ID, userIDFrom(c), in.Name, in.Type, in.Color, in.Icon,
	); err != nil {
		fail(c, http.StatusInternalServerError, 50012, "Không thể tạo danh mục")
		return
	}
	ok(c, in)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	if _, err := h.db.Exec(
		`DELETE FROM categories WHERE id = ? AND user_id = ?`,
		c.Param("id"), userIDFrom(c),
	); err != nil {
		fail(c, http.StatusInternalServerError, 50013, "Không thể xóa danh mục")
		return
	}
	ok(c, gin.H{"deleted": true})
}
