package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetCycleSummary computes Safe-to-spend for the budget cycle containing
// "now", always recomputed from the DB — no caching, per the spec's "tính
// lại mỗi lần tải, không cache cũ" requirement. Missing income/fixed
// costs/savings goal default to 0 in the formula; only a not-declared income
// is exposed as null in the response (income/previousIncome), so the
// frontend can decide how to gate on it.
func (h *Handler) GetCycleSummary(c *gin.Context) {
	ctx := c.Request.Context()
	userID := userIDFrom(c)
	now := time.Now()

	settings, err := h.settings.Get(ctx, userID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50080, "Không thể tải tổng quan chu kỳ")
		return
	}

	cycleStart, cycleEnd := CycleWindow(settings.CycleStartDay, now)

	incomePtr, err := h.lookupIncome(ctx, userID, cycleStart)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50081, "Không thể tải tổng quan chu kỳ")
		return
	}

	var previousIncomePtr *int64
	if prevCycles := PreviousCycles(settings.CycleStartDay, now, 1); len(prevCycles) > 0 {
		previousIncomePtr, err = h.lookupIncome(ctx, userID, prevCycles[0].Start)
		if err != nil {
			fail(c, http.StatusInternalServerError, 50082, "Không thể tải tổng quan chu kỳ")
			return
		}
	}

	fixedCosts, err := h.fixedCosts.List(ctx, userID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50083, "Không thể tải tổng quan chu kỳ")
		return
	}
	var fixedTotal int64
	for _, fc := range fixedCosts {
		fixedTotal += fc.Amount
	}

	spent, err := h.transactions.SumExpensesInRange(ctx, userID, cycleStart, cycleEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50084, "Không thể tải tổng quan chu kỳ")
		return
	}

	daysRemaining := DaysRemaining(cycleEnd, now)

	var incomeForCalc int64
	if incomePtr != nil {
		incomeForCalc = *incomePtr
	}
	safeToSpend := SafeToSpend(incomeForCalc, fixedTotal, settings.SavingsGoal, spent, daysRemaining)

	ok(c, CycleSummary{
		SafeToSpend:    safeToSpend,
		DaysRemaining:  daysRemaining,
		Income:         incomePtr,
		PreviousIncome: previousIncomePtr,
		Budgets:        []any{},
		ActiveAlerts:   []any{},
	})
}

// lookupIncome fetches the declared income for (userID, cycleStart), and
// returns a nil pointer (not an error) when nothing has been declared for
// that cycle — any other error propagates so the caller can surface it as a
// 500.
func (h *Handler) lookupIncome(ctx context.Context, userID string, cycleStart time.Time) (*int64, error) {
	income, err := h.incomes.Get(ctx, userID, cycleStart)
	if err == nil {
		amt := income.Amount
		return &amt, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}
