package api

import "time"

// stepFrequency advances d by one occurrence of frequency: 7 days for
// "weekly"; for "monthly", the same day-of-month next calendar month,
// clamped to that month's last day when the day doesn't exist there (e.g.
// the 31st in a 30-day month) — reuses clampDay, the same clamping
// CycleWindow uses for cycleStartDay. Any frequency value other than
// "weekly" is treated as monthly (CreateRecurringTransaction/
// UpdateRecurringTransaction already reject anything but "weekly"/"monthly"
// before a template is ever persisted).
func stepFrequency(d time.Time, frequency string) time.Time {
	if frequency == "weekly" {
		return d.AddDate(0, 0, 7)
	}
	year, month, day := d.Date()
	nextMonth := month + 1
	nextYear := year
	if nextMonth > time.December {
		nextMonth = time.January
		nextYear++
	}
	clampedDay := clampDay(day, nextYear, nextMonth)
	return time.Date(nextYear, nextMonth, clampedDay, 0, 0, 0, 0, d.Location())
}

// AdvanceRecurrence starts from a template's current next-due date and
// repeatedly steps it forward by one frequency interval until the result is
// strictly after asOf. Pure, no I/O — the single source of truth
// ConfirmRecurringTransaction calls to compute the persisted next_due_date.
//
// One call handles both the ordinary single-occurrence case (next-due date
// today or in the past, one step lands in the future) and the "catch up"
// collapse of any number of missed occurrences into a single next future
// date (spec Design Notes) — each iteration strictly advances the date, so
// no loop bound is needed.
func AdvanceRecurrence(nextDue time.Time, frequency string, asOf time.Time) time.Time {
	for !nextDue.After(asOf) {
		nextDue = stepFrequency(nextDue, frequency)
	}
	return nextDue
}

// MostRecentOccurrence returns the last occurrence date in the schedule
// anchored at nextDue (stepping forward by frequency) that falls on or
// before asOf — the date a "catch up" suggested draft is drafted for (spec
// Design Notes: "a single catch up suggested draft ... for the most recent
// occurrence"). Callers must only call this when nextDue is already on/before
// asOf (i.e. the template is due) — attachDueDraft checks that first.
func MostRecentOccurrence(nextDue time.Time, frequency string, asOf time.Time) time.Time {
	occurrence := nextDue
	for {
		next := stepFrequency(occurrence, frequency)
		if next.After(asOf) {
			return occurrence
		}
		occurrence = next
	}
}

// attachDueDraft sets rt.DueDraftDate (see RecurringTransaction's doc
// comment) when rt is active and its NextDueDate is on/before asOf. A
// malformed NextDueDate (shouldn't happen — always written by this
// package) is treated as not-due rather than failing the whole list.
func attachDueDraft(rt *RecurringTransaction, asOf time.Time) {
	if !rt.Active {
		return
	}
	nextDue, err := time.ParseInLocation(dateLayout, rt.NextDueDate, asOf.Location())
	if err != nil || nextDue.After(asOf) {
		return
	}
	rt.DueDraftDate = MostRecentOccurrence(nextDue, rt.Frequency, asOf).Format(dateLayout)
}
