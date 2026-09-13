package api

import (
	"math"
	"time"
)

// CycleWindow computes the [start, end) boundary for the budget cycle that
// contains asOf, given the day-of-month a cycle starts on (1-31).
//
// A cycle starting on cycleStartDay in a given month runs from that day
// (inclusive) up to the same day next month (exclusive). When cycleStartDay
// does not exist in a given month (e.g. 31 in February), the effective start
// day for that month clamps to the month's last day rather than overflowing
// into the next month — and asOf landing exactly on that clamped day counts
// as the start of the new cycle, same as an unclamped day would.
//
// Pure function, no I/O — this is the single source of truth every later
// story/epic imports for cycle boundaries.
func CycleWindow(cycleStartDay int, asOf time.Time) (start, end time.Time) {
	loc := asOf.Location()
	year, month, day := asOf.Date()

	thisMonthStartDay := clampDay(cycleStartDay, year, month)

	startYear, startMonth := year, month
	if day < thisMonthStartDay {
		startMonth--
		if startMonth < time.January {
			startMonth = time.December
			startYear--
		}
	}
	startDay := clampDay(cycleStartDay, startYear, startMonth)
	start = time.Date(startYear, startMonth, startDay, 0, 0, 0, 0, loc)

	endYear, endMonth := startYear, startMonth+1
	if endMonth > time.December {
		endMonth = time.January
		endYear++
	}
	endDay := clampDay(cycleStartDay, endYear, endMonth)
	end = time.Date(endYear, endMonth, endDay, 0, 0, 0, 0, loc)

	return start, end
}

// clampDay clamps day (1-31) to the valid range of days for year/month,
// so a nominal day that doesn't exist in a short month (e.g. 31 in
// February) resolves to that month's last day.
func clampDay(day int, year int, month time.Month) int {
	last := lastDayOfMonth(year, month)
	if day > last {
		day = last
	}
	if day < 1 {
		day = 1
	}
	return day
}

// lastDayOfMonth returns the number of days in the given month/year.
func lastDayOfMonth(year int, month time.Month) int {
	// Day 0 of the following month is the last day of this one.
	firstOfNext := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	lastOfThis := firstOfNext.AddDate(0, 0, -1)
	return lastOfThis.Day()
}

// SafeToSpend computes the daily spending envelope for the rest of the
// cycle: (income - fixedCosts - savingsGoal - spent) / daysRemaining. Pure,
// no I/O. Never clamped to 0 (per AD-8) — an overspent cycle must surface as
// a real negative number, not a floor of 0, so the UI can show it honestly.
// daysRemaining is floored at 1, matching DaysRemaining's own convention, so
// a caller passing 0 never divides by zero.
func SafeToSpend(income, fixedCosts, savingsGoal, spent int64, daysRemaining int) int64 {
	if daysRemaining <= 0 {
		daysRemaining = 1
	}
	return floorDiv(income-fixedCosts-savingsGoal-spent, int64(daysRemaining))
}

// floorDiv divides a by b, rounding toward negative infinity (unlike Go's
// built-in / which truncates toward zero) so a negative numerator smaller in
// magnitude than b still yields a real negative result instead of 0.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// DaysRemaining returns the number of days left in a cycle ending at end
// (exclusive), counted from asOf. Both end and asOf are truncated to
// midnight in asOf's location before differencing, so time-of-day never
// affects the count. Floored at 1 so the last day of a cycle (and any
// same-day edge case) never divides by zero.
func DaysRemaining(end, asOf time.Time) int {
	loc := asOf.Location()

	y, m, d := asOf.Date()
	todayMidnight := time.Date(y, m, d, 0, 0, 0, 0, loc)

	ey, em, ed := end.Date()
	endMidnight := time.Date(ey, em, ed, 0, 0, 0, 0, loc)

	days := int(math.Round(endMidnight.Sub(todayMidnight).Hours() / 24))
	if days < 1 {
		days = 1
	}
	return days
}

// CrossedThreshold reports the single highest budget-usage threshold (100,
// 90, or 70, checked in that order) that newSpent newly crosses relative to
// prevSpent, given limit. Pure, no I/O.
//
// A non-positive limit means "no budget to check against" and never crosses.
// Checking highest-first means a single save that jumps usage across several
// thresholds at once (e.g. 40% -> 105%) reports only the highest one newly
// crossed, never all of them.
func CrossedThreshold(prevSpent, newSpent, limit int64) (threshold int, crossed bool) {
	if limit <= 0 {
		return 0, false
	}
	for _, t := range []int{100, 90, 70} {
		if prevSpent*100/limit < int64(t) && newSpent*100/limit >= int64(t) {
			return t, true
		}
	}
	return 0, false
}

// AlertStatus maps a crossed threshold (70, 90, 100) to its display status:
// "over" for 100, "near" for anything below. Pure, no I/O — lives next to
// CrossedThreshold so Story 2.3 can import/call this exact mapping for
// GET /budgets's per-category status instead of re-deriving it.
func AlertStatus(threshold int) string {
	if threshold >= 100 {
		return "over"
	}
	return "near"
}

// BudgetStatus computes a budget's usage percent and 3-level status for
// spent against limit: "within" (<70%), "near" (70-99%), "over" (>=100%).
// Pure, no I/O — the single source of truth GET /budgets and the Dashboard
// both call, so the two surfaces always agree on a category's status
// (Story 2.3). A non-positive limit has no meaningful percentage; callers
// should not invoke this for a budget without a limit (UpsertBudget already
// rejects limit<=0 on write).
func BudgetStatus(spent, limit int64) (percent int, status string) {
	if limit <= 0 {
		return 0, "within"
	}
	percent = int(spent * 100 / limit)
	switch {
	case percent >= 100:
		status = "over"
	case percent >= 70:
		status = "near"
	default:
		status = "within"
	}
	return percent, status
}

// CycleWindowForMonth returns the cycle window "labeled" by month (a
// "2006-01" calendar-month string, e.g. a budgets.month value), interpreted
// per AD-4: the cycle that starts within that month at the CURRENT
// cycleStartDay setting — not whatever cycleStartDay was in effect when the
// row was created. This is what makes "budgets.month" mean the same thing
// everywhere it's read: changing cycleStartDay reinterprets old rows
// immediately, with nothing stored per-cycle to migrate.
func CycleWindowForMonth(cycleStartDay int, month string) (start, end time.Time, err error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	year, mon, _ := t.Date()
	// The clamped start day within that month always lands inside the cycle
	// that starts in that month — CycleWindow treats a day exactly on the
	// clamped boundary as the start of the new cycle (see its doc comment).
	asOf := time.Date(year, mon, clampDay(cycleStartDay, year, mon), 0, 0, 0, 0, t.Location())
	start, end = CycleWindow(cycleStartDay, asOf)
	return start, end, nil
}

// CycleInfo is one cycle's [Start, End) window.
type CycleInfo struct {
	Start time.Time
	End   time.Time
}

// PreviousCycles returns the n cycles strictly before the cycle containing
// asOf, most recent first (index 0 is the cycle immediately preceding the
// current one).
func PreviousCycles(cycleStartDay int, asOf time.Time, n int) []CycleInfo {
	if n <= 0 {
		return nil
	}
	out := make([]CycleInfo, 0, n)
	cursor := asOf
	for i := 0; i < n; i++ {
		start, _ := CycleWindow(cycleStartDay, cursor)
		// Step one day before this cycle's start to land in the previous cycle.
		cursor = start.AddDate(0, 0, -1)
		prevStart, prevEnd := CycleWindow(cycleStartDay, cursor)
		out = append(out, CycleInfo{Start: prevStart, End: prevEnd})
	}
	return out
}
