package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type incomeInput struct {
	Amount int64 `json:"amount"`
}

// UpsertIncome declares the caller's income for the current cycle: inserts
// or updates the existing row (unique per user+cycle_start_date).
func (h *Handler) UpsertIncome(c *gin.Context) {
	var in incomeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40040, "Dữ liệu không hợp lệ")
		return
	}
	if in.Amount <= 0 {
		fail(c, http.StatusBadRequest, 40041, "Số tiền phải lớn hơn 0")
		return
	}

	// cycleStartDay is hardcoded to 1 for now; Story 1.4 wires the real user
	// setting into this same CycleWindow call.
	cycleStart, _ := CycleWindow(1, time.Now())

	income, err := h.incomes.Upsert(c.Request.Context(), userIDFrom(c), cycleStart, in.Amount)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50040, "Không thể lưu thu nhập")
		return
	}
	ok(c, income)
}
