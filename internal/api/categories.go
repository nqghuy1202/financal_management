package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ListCategories(c *gin.Context) {
	list, err := h.categories.List(c.Request.Context(), userIDFrom(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 50010, "Không thể tải danh mục")
		return
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

	if err := h.categories.Create(c.Request.Context(), userIDFrom(c), in); err != nil {
		fail(c, http.StatusInternalServerError, 50012, "Không thể tạo danh mục")
		return
	}
	ok(c, in)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	if err := h.categories.Delete(c.Request.Context(), userIDFrom(c), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, 50013, "Không thể xóa danh mục")
		return
	}
	ok(c, gin.H{"deleted": true})
}
