package api

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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
