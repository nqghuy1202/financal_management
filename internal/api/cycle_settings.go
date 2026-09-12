package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// cycleSettingsInput is the single "Save" action's full request body: PUT
// full-replace semantics, no partial updates — the client always sends all
// three fields together, matching the cycle-update-sheet's one Save button.
type cycleSettingsInput struct {
	Income        int64  `json:"income"`
	SavingsGoal   *int64 `json:"savingsGoal"`
	CycleStartDay int    `json:"cycleStartDay"`
}

// UpdateCycleSettings saves income + savings goal + cycle start day together
// in one transaction: the single endpoint behind the cycle-update-sheet's one
// "Save" action. This replaces the retired POST /incomes and
// PUT /settings/savings-goal endpoints.
//
// The request's own cycleStartDay is what keys the income write in this same
// save (via CycleWindow), so changing the cycle start day takes effect
// immediately — the income lands under the NEW cycle, not the old one.
func (h *Handler) UpdateCycleSettings(c *gin.Context) {
	var in cycleSettingsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40070, "Dữ liệu không hợp lệ")
		return
	}
	if in.Income <= 0 {
		fail(c, http.StatusBadRequest, 40071, "Số tiền phải lớn hơn 0")
		return
	}
	if in.SavingsGoal == nil || *in.SavingsGoal < 0 {
		fail(c, http.StatusBadRequest, 40072, "Mục tiêu tiết kiệm không được âm")
		return
	}
	if in.CycleStartDay < 1 || in.CycleStartDay > 31 {
		fail(c, http.StatusBadRequest, 40073, "Ngày bắt đầu chu kỳ không hợp lệ")
		return
	}

	ctx := c.Request.Context()
	userID := userIDFrom(c)

	var result CycleSettingsResult
	err := h.withTx(ctx, func(tx *sql.Tx) error {
		settings, err := NewSettingsRepo(tx).Upsert(ctx, userID, *in.SavingsGoal, in.CycleStartDay)
		if err != nil {
			return err
		}

		// Use the request's own cycle_start_day — not any previously stored
		// value — so a same-save change takes immediate effect. Note this
		// leaves prior incomes rows keyed to the old cycle_start_day as
		// unreferenced history: there is no effective-dating and nothing is
		// deleted, by design.
		cycleStart, _ := CycleWindow(in.CycleStartDay, time.Now())
		income, err := NewIncomeRepo(tx).Upsert(ctx, userID, cycleStart, in.Income)
		if err != nil {
			return err
		}

		result = CycleSettingsResult{Income: income, Settings: settings}
		return nil
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50070, "Không thể lưu cài đặt chu kỳ")
		return
	}
	ok(c, result)
}
