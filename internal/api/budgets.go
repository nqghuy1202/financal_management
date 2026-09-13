package api

import (
	"context"
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
	ctx := c.Request.Context()
	uid := userIDFrom(c)

	list, err := h.budgets.List(ctx, uid)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50030, "Không thể tải ngân sách")
		return
	}

	settings, err := h.settings.Get(ctx, uid)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50035, "Không thể tải ngân sách")
		return
	}
	for i := range list {
		if err := h.attachBudgetStatus(ctx, uid, settings.CycleStartDay, &list[i]); err != nil {
			fail(c, http.StatusInternalServerError, 50036, "Không thể tải ngân sách")
			return
		}
	}
	ok(c, list)
}

// attachBudgetStatus fills in b.Spent/Percent/Status (Story 2.3), computed
// from b.Month reinterpreted per the current cycleStartDay
// (CycleWindowForMonth) — the single path ListBudgets and UpsertBudget both
// call, so a budget's status is never computed two different ways.
func (h *Handler) attachBudgetStatus(ctx context.Context, userID string, cycleStartDay int, b *Budget) error {
	start, end, err := CycleWindowForMonth(cycleStartDay, b.Month)
	if err != nil {
		// Malformed month (shouldn't happen; monthRe validates on write) —
		// leave spent/percent/status at their zero values rather than fail
		// the whole list over one bad row.
		return nil
	}
	spent, err := h.transactions.SumExpensesInCategory(ctx, userID, b.CategoryID, start, end)
	if err != nil {
		return err
	}
	b.Spent = spent
	b.Percent, b.Status = BudgetStatus(spent, b.Limit)
	return nil
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

	settings, err := h.settings.Get(ctx, uid)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50035, "Không thể tải ngân sách")
		return
	}
	if err := h.attachBudgetStatus(ctx, uid, settings.CycleStartDay, &b); err != nil {
		fail(c, http.StatusInternalServerError, 50036, "Không thể tải ngân sách")
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
