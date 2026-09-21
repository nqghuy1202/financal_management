package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func d(s string) time.Time {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

// TestAdvanceRecurrence_Weekly_SingleStep pins the I/O matrix's "Due item,
// app opened" scenario: a weekly template exactly due today advances by
// exactly one 7-day step.
func TestAdvanceRecurrence_Weekly_SingleStep(t *testing.T) {
	got := AdvanceRecurrence(d("2026-09-21"), "weekly", d("2026-09-21"))
	assert.Equal(t, d("2026-09-28"), got)
}

// TestAdvanceRecurrence_Weekly_MultiMiss pins the "Multiple missed
// occurrences" scenario for a weekly template: three missed weeks collapse
// into a single advance to the next future occurrence, not three separate
// steps left for the caller to loop over.
func TestAdvanceRecurrence_Weekly_MultiMiss(t *testing.T) {
	// next_due_date was 2026-08-31 (3 weekly steps ago); today is 2026-09-21.
	got := AdvanceRecurrence(d("2026-08-31"), "weekly", d("2026-09-21"))
	assert.Equal(t, d("2026-09-28"), got)
}

// TestAdvanceRecurrence_Monthly_SingleStep pins the ordinary monthly case:
// due today, one calendar-month step forward, same day-of-month.
func TestAdvanceRecurrence_Monthly_SingleStep(t *testing.T) {
	got := AdvanceRecurrence(d("2026-09-21"), "monthly", d("2026-09-21"))
	assert.Equal(t, d("2026-10-21"), got)
}

// TestAdvanceRecurrence_Monthly_MultiMiss pins the acceptance criterion: "a
// monthly template is 3 months overdue ... confirming it advances
// next_due_date to the next future monthly occurrence" (a single collapse,
// not three separate rows/steps left dangling).
func TestAdvanceRecurrence_Monthly_MultiMiss(t *testing.T) {
	// next_due_date was 2026-06-21 (3 months before "today" 2026-09-21).
	got := AdvanceRecurrence(d("2026-06-21"), "monthly", d("2026-09-21"))
	assert.Equal(t, d("2026-10-21"), got)
}

// TestAdvanceRecurrence_Monthly_EndOfMonthClamp pins the Code Map's explicit
// edge case: a template due the 31st stepping into a 30-day (or shorter)
// month clamps to that month's last day, per iteration.
func TestAdvanceRecurrence_Monthly_EndOfMonthClamp(t *testing.T) {
	// Jan 31 -> Feb 28 (2026 is not a leap year) -> Mar 28 (clamped day
	// carries forward, per the Design Notes' iterative stepping) -> ... ->
	// eventually strictly after "today" 2026-03-15, landing on Mar 28.
	got := AdvanceRecurrence(d("2026-01-31"), "monthly", d("2026-03-15"))
	assert.Equal(t, d("2026-03-28"), got)
}

// TestAdvanceRecurrence_NotYetDue pins that a template whose next_due_date
// is already strictly after asOf is returned unchanged — the loop condition
// stops immediately, so a not-yet-due call never advances the schedule.
// (ConfirmRecurringTransaction is never expected to be called for a
// not-yet-due template in normal operation, but the pure function's
// behavior is still pinned here.)
func TestAdvanceRecurrence_NotYetDue(t *testing.T) {
	got := AdvanceRecurrence(d("2026-09-25"), "weekly", d("2026-09-21"))
	assert.Equal(t, d("2026-09-25"), got)
}

// TestMostRecentOccurrence_SingleMiss pins the "Due item, app opened"
// scenario: exactly due today, the most recent occurrence is today itself.
func TestMostRecentOccurrence_SingleMiss(t *testing.T) {
	got := MostRecentOccurrence(d("2026-09-21"), "weekly", d("2026-09-21"))
	assert.Equal(t, d("2026-09-21"), got)
}

// TestMostRecentOccurrence_MultiMiss pins the "Multiple missed occurrences"
// scenario end-to-end: the suggested draft must be dated at the *last*
// missed occurrence, not the original (stale) next_due_date and not any of
// the earlier skipped ones.
func TestMostRecentOccurrence_MultiMiss(t *testing.T) {
	// Monthly, next_due_date 2026-06-21, "today" 2026-09-21 — occurrences at
	// Jun21, Jul21, Aug21, Sep21 are all <= today; the most recent is Sep21.
	got := MostRecentOccurrence(d("2026-06-21"), "monthly", d("2026-09-21"))
	assert.Equal(t, d("2026-09-21"), got)
}

func TestAttachDueDraft_PausedNeverDue(t *testing.T) {
	rt := RecurringTransaction{NextDueDate: "2026-01-01", Frequency: "monthly", Active: false}
	attachDueDraft(&rt, d("2026-09-21"))
	assert.Empty(t, rt.DueDraftDate)
}

func TestAttachDueDraft_NotYetDue(t *testing.T) {
	rt := RecurringTransaction{NextDueDate: "2026-10-01", Frequency: "monthly", Active: true}
	attachDueDraft(&rt, d("2026-09-21"))
	assert.Empty(t, rt.DueDraftDate)
}

func TestAttachDueDraft_DueToday(t *testing.T) {
	rt := RecurringTransaction{NextDueDate: "2026-09-21", Frequency: "weekly", Active: true}
	attachDueDraft(&rt, d("2026-09-21"))
	assert.Equal(t, "2026-09-21", rt.DueDraftDate)
}

func TestAttachDueDraft_CatchUpCollapsesToSingleDraft(t *testing.T) {
	rt := RecurringTransaction{NextDueDate: "2026-06-21", Frequency: "monthly", Active: true}
	attachDueDraft(&rt, d("2026-09-21"))
	assert.Equal(t, "2026-09-21", rt.DueDraftDate)
}
