package api

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockSettingsRepo(t *testing.T) (*SettingsRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewSettingsRepo(db), mock, func() { db.Close() }
}

func TestSettingsRepo_UpsertSavingsGoal_Insert(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal\)\s+VALUES \(\?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\)`).
		WithArgs("u1", int64(2000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(2000000), 1))

	out, err := repo.UpsertSavingsGoal(context.Background(), "u1", 2000000)
	require.NoError(t, err)
	assert.Equal(t, Settings{SavingsGoal: 2000000, CycleStartDay: 1}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingsRepo_UpsertSavingsGoal_UpdateInPlace pins the acceptance
// criterion: re-setting the savings goal on an existing row updates
// savings_goal in place and leaves the still-default cycle_start_day (1)
// untouched.
func TestSettingsRepo_UpsertSavingsGoal_UpdateInPlace(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal\)\s+VALUES \(\?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\)`).
		WithArgs("u1", int64(2000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(2000000), 1))

	firstOut, err := repo.UpsertSavingsGoal(context.Background(), "u1", 2000000)
	require.NoError(t, err)
	assert.Equal(t, int64(2000000), firstOut.SavingsGoal)

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal\)\s+VALUES \(\?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\)`).
		WithArgs("u1", int64(3500000)).
		WillReturnResult(sqlmock.NewResult(0, 2)) // MySQL reports 2 affected rows for an ON DUPLICATE KEY UPDATE
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(3500000), 1))

	secondOut, err := repo.UpsertSavingsGoal(context.Background(), "u1", 3500000)
	require.NoError(t, err)
	assert.Equal(t, Settings{SavingsGoal: 3500000, CycleStartDay: 1}, secondOut, "cycle_start_day must stay at its default")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingsRepo_UpsertSavingsGoal_ReSelectErrorFallsBack pins the
// fallback path when the post-write re-SELECT fails with a genuine error:
// UpsertSavingsGoal must still return what it just wrote, with a nil error.
func TestSettingsRepo_UpsertSavingsGoal_ReSelectErrorFallsBack(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal\)\s+VALUES \(\?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\)`).
		WithArgs("u1", int64(2000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnError(sql.ErrConnDone)

	out, err := repo.UpsertSavingsGoal(context.Background(), "u1", 2000000)
	require.NoError(t, err)
	assert.Equal(t, Settings{SavingsGoal: 2000000, CycleStartDay: 1}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}
