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

func TestSettingsRepo_Upsert_Insert(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal, cycle_start_day\)\s+VALUES \(\?, \?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\), cycle_start_day = VALUES\(cycle_start_day\)`).
		WithArgs("u1", int64(2000000), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(2000000), 1))

	out, err := repo.Upsert(context.Background(), "u1", 2000000, 1)
	require.NoError(t, err)
	assert.Equal(t, Settings{SavingsGoal: 2000000, CycleStartDay: 1}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingsRepo_Get_Found pins the ordinary path: an existing row is
// returned as-is.
func TestSettingsRepo_Get_Found(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(3500000), 15))

	out, err := repo.Get(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, Settings{SavingsGoal: 3500000, CycleStartDay: 15}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingsRepo_Get_DefaultsOnNoRow pins the spec's requirement: no row
// yet must return the DB-column defaults (savings_goal 0, cycle_start_day
// 1) with a nil error, never sql.ErrNoRows.
func TestSettingsRepo_Get_DefaultsOnNoRow(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnError(sql.ErrNoRows)

	out, err := repo.Get(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, Settings{SavingsGoal: 0, CycleStartDay: 1}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingsRepo_Get_OtherErrorPropagates pins that a genuine DB error
// (not "no rows") is not swallowed into the default.
func TestSettingsRepo_Get_OtherErrorPropagates(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnError(sql.ErrConnDone)

	_, err := repo.Get(context.Background(), "u1")
	assert.ErrorIs(t, err, sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingsRepo_Upsert_UpdateInPlace pins the acceptance criterion:
// re-saving settings on an existing row updates both savings_goal and
// cycle_start_day in place, not as a second row.
func TestSettingsRepo_Upsert_UpdateInPlace(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal, cycle_start_day\)\s+VALUES \(\?, \?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\), cycle_start_day = VALUES\(cycle_start_day\)`).
		WithArgs("u1", int64(2000000), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(2000000), 1))

	firstOut, err := repo.Upsert(context.Background(), "u1", 2000000, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2000000), firstOut.SavingsGoal)

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal, cycle_start_day\)\s+VALUES \(\?, \?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\), cycle_start_day = VALUES\(cycle_start_day\)`).
		WithArgs("u1", int64(3500000), 15).
		WillReturnResult(sqlmock.NewResult(0, 2)) // MySQL reports 2 affected rows for an ON DUPLICATE KEY UPDATE
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(3500000), 15))

	secondOut, err := repo.Upsert(context.Background(), "u1", 3500000, 15)
	require.NoError(t, err)
	assert.Equal(t, Settings{SavingsGoal: 3500000, CycleStartDay: 15}, secondOut, "cycle_start_day must be persisted and updated in place")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSettingsRepo_Upsert_ReSelectErrorFallsBack pins the fallback path when
// the post-write re-SELECT fails with a genuine error: Upsert must still
// return what it just wrote, with a nil error.
func TestSettingsRepo_Upsert_ReSelectErrorFallsBack(t *testing.T) {
	repo, mock, closeDB := newMockSettingsRepo(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal, cycle_start_day\)\s+VALUES \(\?, \?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\), cycle_start_day = VALUES\(cycle_start_day\)`).
		WithArgs("u1", int64(2000000), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnError(sql.ErrConnDone)

	out, err := repo.Upsert(context.Background(), "u1", 2000000, 1)
	require.NoError(t, err)
	assert.Equal(t, Settings{SavingsGoal: 2000000, CycleStartDay: 1}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}
