package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type settingsInput struct {
	SavingsGoal *int64 `json:"savingsGoal"`
}

// UpdateSavingsGoal sets the caller's savings goal. cycle_start_day is left
// untouched here — it keeps its default (1) until Story 1.4 absorbs this
// into the shared PUT /cycle-settings endpoint.
func (h *Handler) UpdateSavingsGoal(c *gin.Context) {
	var in settingsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40060, "Dữ liệu không hợp lệ")
		return
	}
	if in.SavingsGoal == nil {
		fail(c, http.StatusBadRequest, 40060, "Dữ liệu không hợp lệ")
		return
	}
	if *in.SavingsGoal < 0 {
		fail(c, http.StatusBadRequest, 40061, "Mục tiêu tiết kiệm không được âm")
		return
	}

	settings, err := h.settings.UpsertSavingsGoal(c.Request.Context(), userIDFrom(c), *in.SavingsGoal)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50060, "Không thể lưu mục tiêu tiết kiệm")
		return
	}
	ok(c, settings)
}
