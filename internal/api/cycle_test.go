package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustDate(s string) time.Time {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

// TestCycleWindow_Day1PlainCase pins the simplest case: cycleStartDay=1
// behaves like a plain calendar month.
func TestCycleWindow_Day1PlainCase(t *testing.T) {
	start, end := CycleWindow(1, mustDate("2026-09-15"))
	assert.True(t, start.Equal(mustDate("2026-09-01")))
	assert.True(t, end.Equal(mustDate("2026-10-01")))
}

// TestCycleWindow_MidMonth exercises a cycleStartDay that isn't 1, both
// before and after that day within the same calendar month.
func TestCycleWindow_MidMonth(t *testing.T) {
	// asOf after the start day this month: cycle began this month.
	start, end := CycleWindow(10, mustDate("2026-09-15"))
	assert.True(t, start.Equal(mustDate("2026-09-10")))
	assert.True(t, end.Equal(mustDate("2026-10-10")))

	// asOf before the start day this month: cycle began last month.
	start, end = CycleWindow(10, mustDate("2026-09-05"))
	assert.True(t, start.Equal(mustDate("2026-08-10")))
	assert.True(t, end.Equal(mustDate("2026-09-10")))
}

// TestCycleWindow_ClampShortMonth pins the I/O matrix's clamp scenario:
// cycleStartDay=31 with asOf in February must clamp start/end to the last
// day of the short month instead of overflowing into the next month.
func TestCycleWindow_ClampShortMonth(t *testing.T) {
	start, end := CycleWindow(31, mustDate("2026-02-15"))
	assert.True(t, start.Equal(mustDate("2026-01-31")), "start should clamp to Jan 31 (Jan has 31 days)")
	assert.True(t, end.Equal(mustDate("2026-02-28")), "end should clamp to Feb 28 (2026 is not a leap year), not overflow into March")
}

// TestCycleWindow_BoundaryInclusive pins the I/O matrix's boundary
// scenario: asOf exactly on cycleStartDay counts as the start of the new
// cycle (start inclusive, end exclusive).
func TestCycleWindow_BoundaryInclusive(t *testing.T) {
	start, end := CycleWindow(10, mustDate("2026-09-10"))
	assert.True(t, start.Equal(mustDate("2026-09-10")), "asOf on the start day itself must count as the new cycle's start")
	assert.True(t, end.Equal(mustDate("2026-10-10")))
}

// TestCycleWindow_BoundaryInclusive_ClampedDay pins the boundary rule when
// the start day itself is a clamped one: asOf landing exactly on Feb's
// clamped last day (standing in for day 31) must still count as the new
// cycle's start, and the following cycle must resolve against the
// unclamped day 31 once March (31 days) makes it valid again.
func TestCycleWindow_BoundaryInclusive_ClampedDay(t *testing.T) {
	start, end := CycleWindow(31, mustDate("2026-02-28"))
	assert.True(t, start.Equal(mustDate("2026-02-28")))
	assert.True(t, end.Equal(mustDate("2026-03-31")))
}

// TestPreviousCycles_ClampShortMonthBoundary walks PreviousCycles backward
// from a date in March with cycleStartDay=31, so the walk must cross the
// Feb28-clamped cycle (Jan 31 - Feb 28) before reaching the unclamped Jan
// cycle (Dec 31 - Jan 31).
func TestPreviousCycles_ClampShortMonthBoundary(t *testing.T) {
	cycles := PreviousCycles(31, mustDate("2026-03-15"), 3)
	require.Len(t, cycles, 3)

	// Current cycle (not returned) is Feb 28 - Mar 31; index 0 is the one
	// immediately before it.
	assert.True(t, cycles[0].Start.Equal(mustDate("2026-01-31")), "cycle 0 should start Jan 31")
	assert.True(t, cycles[0].End.Equal(mustDate("2026-02-28")), "cycle 0 should end clamped to Feb 28")

	assert.True(t, cycles[1].Start.Equal(mustDate("2025-12-31")), "cycle 1 should start Dec 31")
	assert.True(t, cycles[1].End.Equal(mustDate("2026-01-31")), "cycle 1 should end Jan 31")

	assert.True(t, cycles[2].Start.Equal(mustDate("2025-11-30")), "cycle 2 should start clamped to Nov 30")
	assert.True(t, cycles[2].End.Equal(mustDate("2025-12-31")), "cycle 2 should end Dec 31")
}

// TestSafeToSpend_EnvelopeFormula pins the core formula: income minus fixed
// costs, savings goal and what's already spent this cycle, split evenly over
// the days remaining.
func TestSafeToSpend_EnvelopeFormula(t *testing.T) {
	got := SafeToSpend(15_000_000, 5_000_000, 2_000_000, 1_000_000, 10)
	assert.Equal(t, int64(700_000), got)
}

// TestSafeToSpend_MissingInputsDefaultToZero pins the I/O matrix's "fixed
// costs / savings goal not set" scenario at the pure-function level: callers
// pass 0 for whichever inputs are undeclared and the formula treats them as
// such, without special-casing.
func TestSafeToSpend_MissingInputsDefaultToZero(t *testing.T) {
	got := SafeToSpend(10_000_000, 0, 0, 0, 10)
	assert.Equal(t, int64(1_000_000), got)
}

// TestSafeToSpend_Overspent_NeverClampedToZero pins AD-8: when
// spentThisCycleSoFar pushes the envelope negative, SafeToSpend returns the
// real negative number, never clamped to 0.
func TestSafeToSpend_Overspent_NeverClampedToZero(t *testing.T) {
	got := SafeToSpend(5_000_000, 3_000_000, 1_000_000, 4_000_000, 5)
	assert.Equal(t, int64(-600_000), got)
	assert.Negative(t, got)
}

// TestSafeToSpend_Overspent_FloorsTowardNegativeInfinity pins the floor-div
// fix: a numerator that's negative but smaller in magnitude than
// daysRemaining must still floor to -1, not truncate to 0 the way Go's
// built-in / would (-1/2 == 0 under truncation).
func TestSafeToSpend_Overspent_FloorsTowardNegativeInfinity(t *testing.T) {
	got := SafeToSpend(0, 0, 0, 1, 2)
	assert.Equal(t, int64(-1), got)
}

// TestSafeToSpend_ZeroDaysRemaining_NoPanic pins the guard: daysRemaining=0
// must not divide by zero, and instead behaves as if it were 1 (matching
// DaysRemaining's own floor-at-1 convention).
func TestSafeToSpend_ZeroDaysRemaining_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		got := SafeToSpend(10_000_000, 0, 0, 0, 0)
		assert.Equal(t, int64(10_000_000), got)
	})
}

// TestDaysRemaining_MidCycle pins the ordinary case: several days left,
// truncated to whole days regardless of time-of-day.
func TestDaysRemaining_MidCycle(t *testing.T) {
	asOf := mustDate("2026-09-15")
	end := mustDate("2026-10-01")
	assert.Equal(t, 16, DaysRemaining(end, asOf))
}

// TestDaysRemaining_LastDayOfCycle pins the I/O matrix's boundary scenario:
// asOf is the final calendar day before end, so exactly 1 day remains, never
// 0 (avoids div-by-zero downstream in SafeToSpend).
func TestDaysRemaining_LastDayOfCycle(t *testing.T) {
	asOf := mustDate("2026-09-30")
	end := mustDate("2026-10-01")
	assert.Equal(t, 1, DaysRemaining(end, asOf))
}

// TestDaysRemaining_TimeOfDayIgnored pins the midnight-truncation rule: an
// asOf later in the day than end's own time-of-day must not reduce the count
// below the whole-day difference.
func TestDaysRemaining_TimeOfDayIgnored(t *testing.T) {
	asOf := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 1, DaysRemaining(end, asOf))
}

// TestDaysRemaining_FlooredAtOne pins the floor itself: even an asOf that
// lands on or after end (which CycleWindow should never produce, but the
// pure function must still defend against) never returns less than 1.
func TestDaysRemaining_FlooredAtOne(t *testing.T) {
	asOf := mustDate("2026-10-01")
	end := mustDate("2026-10-01")
	assert.Equal(t, 1, DaysRemaining(end, asOf))

	asOf = mustDate("2026-10-02")
	end = mustDate("2026-10-01")
	assert.Equal(t, 1, DaysRemaining(end, asOf))
}

func TestPreviousCycles(t *testing.T) {
	cycles := PreviousCycles(1, mustDate("2026-09-15"), 3)
	assert.Len(t, cycles, 3)
	assert.True(t, cycles[0].Start.Equal(mustDate("2026-08-01")))
	assert.True(t, cycles[0].End.Equal(mustDate("2026-09-01")))
	assert.True(t, cycles[1].Start.Equal(mustDate("2026-07-01")))
	assert.True(t, cycles[1].End.Equal(mustDate("2026-08-01")))
	assert.True(t, cycles[2].Start.Equal(mustDate("2026-06-01")))
	assert.True(t, cycles[2].End.Equal(mustDate("2026-07-01")))
}

// TestCrossedThreshold_FirstCrossing pins the I/O matrix's "First crossing"
// scenario: spend moves from below 70% to above it.
func TestCrossedThreshold_FirstCrossing(t *testing.T) {
	threshold, crossed := CrossedThreshold(600_000, 750_000, 1_000_000)
	assert.True(t, crossed)
	assert.Equal(t, 70, threshold)
}

// TestCrossedThreshold_AlreadyOverThreshold_NoNewCrossing pins the "Already
// alerted at this threshold" scenario at the pure-function level: spend stays
// above 90% but below 100%, so no new crossing is reported for either
// threshold it's already past.
func TestCrossedThreshold_AlreadyOverThreshold_NoNewCrossing(t *testing.T) {
	threshold, crossed := CrossedThreshold(950_000, 980_000, 1_000_000)
	assert.False(t, crossed)
	assert.Equal(t, 0, threshold)
}

// TestCrossedThreshold_MultiThresholdJump pins the "Multi-threshold jump"
// scenario: one jump from 40% to 105% must report only the highest threshold
// (100), never 70 or 90.
func TestCrossedThreshold_MultiThresholdJump(t *testing.T) {
	threshold, crossed := CrossedThreshold(400_000, 1_050_000, 1_000_000)
	assert.True(t, crossed)
	assert.Equal(t, 100, threshold)
}

// TestCrossedThreshold_NoBudget_LimitZeroOrNegative pins the "no budget"
// guard: a non-positive limit never reports a crossing, however large spend
// is, avoiding a division by zero.
func TestCrossedThreshold_NoBudget_LimitZeroOrNegative(t *testing.T) {
	threshold, crossed := CrossedThreshold(0, 1_000_000, 0)
	assert.False(t, crossed)
	assert.Equal(t, 0, threshold)

	threshold, crossed = CrossedThreshold(0, 1_000_000, -1)
	assert.False(t, crossed)
	assert.Equal(t, 0, threshold)
}

// TestCrossedThreshold_ExactlyAtThreshold pins the boundary: landing exactly
// on a threshold (not just past it) counts as crossing it.
func TestCrossedThreshold_ExactlyAtThreshold(t *testing.T) {
	threshold, crossed := CrossedThreshold(690_000, 700_000, 1_000_000)
	assert.True(t, crossed)
	assert.Equal(t, 70, threshold)
}

// TestCrossedThreshold_DecreasingSpend_NeverCrosses pins the monotonic
// property checkBudgetThreshold's callers rely on: spend moving down can
// never report a crossing, only moving up can.
func TestCrossedThreshold_DecreasingSpend_NeverCrosses(t *testing.T) {
	threshold, crossed := CrossedThreshold(900_000, 500_000, 1_000_000)
	assert.False(t, crossed)
	assert.Equal(t, 0, threshold)
}

// TestCrossedThreshold_StaysBelowAllThresholds pins the plain no-crossing
// case: spend increases but stays under 70%.
func TestCrossedThreshold_StaysBelowAllThresholds(t *testing.T) {
	threshold, crossed := CrossedThreshold(100_000, 500_000, 1_000_000)
	assert.False(t, crossed)
	assert.Equal(t, 0, threshold)
}
