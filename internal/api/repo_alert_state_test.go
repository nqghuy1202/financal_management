package api

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockAlertStateRepo(t *testing.T) (*AlertStateRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewAlertStateRepo(db), mock, func() { db.Close() }
}

// TestAlertStateRepo_Trigger_Insert pins the ordinary path: the first call
// for a (user, category, cycle, threshold) key inserts a new row.
func TestAlertStateRepo_Trigger_Insert(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectExec(`INSERT INTO budget_alert_state \(id, user_id, category_id, cycle_start_date, threshold\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE threshold = VALUES\(threshold\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "c1", "2026-09-01", 70).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Trigger(context.Background(), "u1", "c1", cycleStart, 70)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAlertStateRepo_Trigger_IdempotentSecondCall pins the "Concurrent
// same-threshold crossing" guarantee at the repo level: a second Trigger
// call for the exact same key must not error — the idempotent ON DUPLICATE
// KEY UPDATE upsert absorbs it, so duplicate-row detection is unnecessary
// (structurally impossible via this repo).
func TestAlertStateRepo_Trigger_IdempotentSecondCall(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectExec(`INSERT INTO budget_alert_state \(id, user_id, category_id, cycle_start_date, threshold\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE threshold = VALUES\(threshold\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "c1", "2026-09-01", 100).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO budget_alert_state \(id, user_id, category_id, cycle_start_date, threshold\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE threshold = VALUES\(threshold\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "c1", "2026-09-01", 100).
		WillReturnResult(sqlmock.NewResult(0, 2)) // MySQL reports 2 affected rows for an ON DUPLICATE KEY UPDATE

	require.NoError(t, repo.Trigger(context.Background(), "u1", "c1", cycleStart, 100))
	require.NoError(t, repo.Trigger(context.Background(), "u1", "c1", cycleStart, 100))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAlertStateRepo_ListActive_Empty pins the "no active alerts" case:
// zero rows returns an empty (never nil) slice.
func TestAlertStateRepo_ListActive_Empty(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectQuery(`SELECT category_id, threshold FROM budget_alert_state\s+WHERE user_id = \? AND cycle_start_date = \? AND dismissed_at IS NULL\s+ORDER BY threshold DESC, triggered_at DESC`).
		WithArgs("u1", "2026-09-01").
		WillReturnRows(sqlmock.NewRows([]string{"category_id", "threshold"}))

	alerts, err := repo.ListActive(context.Background(), "u1", cycleStart)
	require.NoError(t, err)
	require.NotNil(t, alerts)
	assert.Empty(t, alerts)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAlertStateRepo_ListActive_OrderingOverBeforeNear pins that the DB's
// own ORDER BY threshold DESC naturally yields "over" (threshold 100) rows
// ahead of "near" (70/90) rows, and that Status is derived via AlertStatus.
func TestAlertStateRepo_ListActive_OrderingOverBeforeNear(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectQuery(`SELECT category_id, threshold FROM budget_alert_state\s+WHERE user_id = \? AND cycle_start_date = \? AND dismissed_at IS NULL\s+ORDER BY threshold DESC, triggered_at DESC`).
		WithArgs("u1", "2026-09-01").
		WillReturnRows(sqlmock.NewRows([]string{"category_id", "threshold"}).
			AddRow("c-over", 100).
			AddRow("c-near", 70))

	alerts, err := repo.ListActive(context.Background(), "u1", cycleStart)
	require.NoError(t, err)
	require.Len(t, alerts, 2)
	assert.Equal(t, ActiveAlert{CategoryID: "c-over", Threshold: 100, Status: "over"}, alerts[0])
	assert.Equal(t, ActiveAlert{CategoryID: "c-near", Threshold: 70, Status: "near"}, alerts[1])
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAlertStateRepo_ListActive_PastCycleExcluded pins that cycleStart is
// threaded through to the query as a real, live argument (formatted per
// call) rather than a hardcoded literal — the mock is wired to answer only
// the current cycle's exact date string, and calling ListActive with a
// different (past) cycleStart must fail to match it, since sqlmock matches
// WithArgs literally. sqlmock can't model true cross-call DB isolation (a
// second row actually persisted under a different cycle_start_date and
// proven excluded by the WHERE clause against live data), so this
// argument-mismatch check is the closest proof available at this test's
// level that the parameter is live, not hardcoded.
func TestAlertStateRepo_ListActive_PastCycleExcluded(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	pastCycleStart := mustDate("2026-08-01")
	// The mock only answers the *current* cycle's exact date argument.
	mock.ExpectQuery(`SELECT category_id, threshold FROM budget_alert_state\s+WHERE user_id = \? AND cycle_start_date = \? AND dismissed_at IS NULL\s+ORDER BY threshold DESC, triggered_at DESC`).
		WithArgs("u1", "2026-09-01").
		WillReturnRows(sqlmock.NewRows([]string{"category_id", "threshold"}))

	// Calling with a different (past) cycleStart must not match the
	// expectation above — if it did, cycleStart would have to be hardcoded
	// or ignored rather than passed through as a real argument.
	_, err := repo.ListActive(context.Background(), "u1", pastCycleStart)
	assert.Error(t, err, "cycle_start_date must be passed as the real, current query argument, not hardcoded")
}

// TestAlertStateRepo_ListActive_QueryError propagates a genuine DB error.
func TestAlertStateRepo_ListActive_QueryError(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectQuery(`SELECT category_id, threshold FROM budget_alert_state\s+WHERE user_id = \? AND cycle_start_date = \? AND dismissed_at IS NULL\s+ORDER BY threshold DESC, triggered_at DESC`).
		WithArgs("u1", "2026-09-01").
		WillReturnError(sql.ErrConnDone)

	_, err := repo.ListActive(context.Background(), "u1", cycleStart)
	assert.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAlertStateRepo_Dismiss_Found pins the ordinary path: an undismissed
// row is updated.
func TestAlertStateRepo_Dismiss_Found(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectExec(`UPDATE budget_alert_state SET dismissed_at = CURRENT_TIMESTAMP\s+WHERE user_id = \? AND category_id = \? AND cycle_start_date = \? AND threshold = \? AND dismissed_at IS NULL`).
		WithArgs("u1", "c1", "2026-09-01", 70).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Dismiss(context.Background(), "u1", "c1", cycleStart, 70)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAlertStateRepo_Dismiss_IdempotentOnRepeat pins that a second dismiss
// of an already-dismissed row (0 rows affected) is not an error.
func TestAlertStateRepo_Dismiss_IdempotentOnRepeat(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectExec(`UPDATE budget_alert_state SET dismissed_at = CURRENT_TIMESTAMP\s+WHERE user_id = \? AND category_id = \? AND cycle_start_date = \? AND threshold = \? AND dismissed_at IS NULL`).
		WithArgs("u1", "c1", "2026-09-01", 70).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Dismiss(context.Background(), "u1", "c1", cycleStart, 70)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAlertStateRepo_Dismiss_QueryError propagates a genuine DB error.
func TestAlertStateRepo_Dismiss_QueryError(t *testing.T) {
	repo, mock, closeDB := newMockAlertStateRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectExec(`UPDATE budget_alert_state SET dismissed_at = CURRENT_TIMESTAMP\s+WHERE user_id = \? AND category_id = \? AND cycle_start_date = \? AND threshold = \? AND dismissed_at IS NULL`).
		WithArgs("u1", "c1", "2026-09-01", 70).
		WillReturnError(sql.ErrConnDone)

	err := repo.Dismiss(context.Background(), "u1", "c1", cycleStart, 70)
	assert.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
