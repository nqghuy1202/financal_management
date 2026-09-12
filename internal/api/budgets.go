package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

var monthRe = regexp.MustCompile(`^\d{4}-\d{2}$`)

type budgetInput struct {
	CategoryID string `json:"categoryId"`
	Limit      int64  `json:"limit"`
	Month      string `json:"month"`
}

func (h *Handler) ListBudgets(c *gin.Context) {
	list, err := h.budgets.List(c.Request.Context(), userIDFrom(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 50030, "Không thể tải ngân sách")
		return
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
	ctx := c.Request.Context()
	uid := userIDFrom(c)

	owns, err := h.categories.Owns(ctx, uid, in.CategoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50034, "Lỗi máy chủ")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40032, "Danh mục không tồn tại")
		return
	}

	b, err := h.budgets.Upsert(ctx, uid, Budget{CategoryID: in.CategoryID, Limit: in.Limit, Month: in.Month})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50032, "Không thể lưu ngân sách")
		return
	}
	ok(c, b)
}

func (h *Handler) DeleteBudget(c *gin.Context) {
	if err := h.budgets.Delete(c.Request.Context(), userIDFrom(c), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, 50033, "Không thể xóa ngân sách")
		return
	}
	ok(c, gin.H{"deleted": true})
}
