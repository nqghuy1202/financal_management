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
