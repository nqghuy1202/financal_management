package api

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockIncomeRepo(t *testing.T) (*IncomeRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewIncomeRepo(db), mock, func() { db.Close() }
}

func TestIncomeRepo_Upsert_Insert(t *testing.T) {
	repo, mock, closeDB := newMockIncomeRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "2026-09-01", int64(15000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", "2026-09-01").
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).
			AddRow("i1", cycleStart, int64(15000000)))

	out, err := repo.Upsert(context.Background(), "u1", cycleStart, 15000000)
	require.NoError(t, err)
	assert.Equal(t, Income{ID: "i1", CycleStartDate: "2026-09-01", Amount: 15000000}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestIncomeRepo_Upsert_UpdateInPlace pins the "re-declare same cycle" I/O
// matrix scenario: a 2nd Upsert for the same (user, cycle_start_date) with a
// different amount must update the existing row in place, not insert a 2nd
// row.
func TestIncomeRepo_Upsert_UpdateInPlace(t *testing.T) {
	repo, mock, closeDB := newMockIncomeRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")

	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "2026-09-01", int64(15000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", "2026-09-01").
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).
			AddRow("i1", cycleStart, int64(15000000)))

	firstOut, err := repo.Upsert(context.Background(), "u1", cycleStart, 15000000)
	require.NoError(t, err)
	assert.Equal(t, int64(15000000), firstOut.Amount)

	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "2026-09-01", int64(20000000)).
		WillReturnResult(sqlmock.NewResult(0, 2)) // MySQL reports 2 affected rows for an ON DUPLICATE KEY UPDATE
	// Re-SELECT-after-write returns the SAME row id ("i1") with the new
	// amount, proving it was updated in place rather than inserted as a
	// second row.
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", "2026-09-01").
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).
			AddRow("i1", cycleStart, int64(20000000)))

	secondOut, err := repo.Upsert(context.Background(), "u1", cycleStart, 20000000)
	require.NoError(t, err)
	assert.Equal(t, "i1", secondOut.ID, "id must stay the same row, not a newly inserted one")
	assert.Equal(t, int64(20000000), secondOut.Amount)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestIncomeRepo_Upsert_ReSelectErrorFallsBack pins the fallback path when
// the post-write re-SELECT fails with a genuine (non-"not found") error:
// Upsert must still return the row it just wrote, with a nil error, rather
// than propagating the SELECT failure.
func TestIncomeRepo_Upsert_ReSelectErrorFallsBack(t *testing.T) {
	repo, mock, closeDB := newMockIncomeRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "2026-09-01", int64(15000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", "2026-09-01").
		WillReturnError(sql.ErrConnDone)

	out, err := repo.Upsert(context.Background(), "u1", cycleStart, 15000000)
	require.NoError(t, err)
	assert.Equal(t, "2026-09-01", out.CycleStartDate)
	assert.Equal(t, int64(15000000), out.Amount)
	assert.NotEmpty(t, out.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestIncomeRepo_Get_NotFound pins the "no carry-over" I/O matrix scenario:
// a cycle with nothing declared for it returns sql.ErrNoRows, not a zero
// value silently.
func TestIncomeRepo_Get_NotFound(t *testing.T) {
	repo, mock, closeDB := newMockIncomeRepo(t)
	defer closeDB()

	nextCycle := mustDate("2026-10-01")
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", "2026-10-01").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.Get(context.Background(), "u1", nextCycle)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIncomeRepo_Get_Found(t *testing.T) {
	repo, mock, closeDB := newMockIncomeRepo(t)
	defer closeDB()

	cycleStart := mustDate("2026-09-01")
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", "2026-09-01").
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).
			AddRow("i1", cycleStart, int64(15000000)))

	out, err := repo.Get(context.Background(), "u1", cycleStart)
	require.NoError(t, err)
	assert.Equal(t, Income{ID: "i1", CycleStartDate: "2026-09-01", Amount: 15000000}, out)
}
