package api

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockBudgetRepo(t *testing.T) (*BudgetRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewBudgetRepo(db), mock, func() { db.Close() }
}

func TestBudgetRepo_List(t *testing.T) {
	repo, mock, closeDB := newMockBudgetRepo(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
		AddRow("b1", "c1", int64(3000000), "2026-09")
	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(rows)

	list, err := repo.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []Budget{{ID: "b1", CategoryID: "c1", Limit: 3000000, Month: "2026-09"}}, list)
}

// TestBudgetRepo_List_QueryError pins that a genuine query error propagates
// instead of being swallowed into an empty list.
func TestBudgetRepo_List_QueryError(t *testing.T) {
	repo, mock, closeDB := newMockBudgetRepo(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnError(sql.ErrConnDone)

	_, err := repo.List(context.Background(), "u1")
	assert.ErrorIs(t, err, sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBudgetRepo_Upsert_Insert(t *testing.T) {
	repo, mock, closeDB := newMockBudgetRepo(t)
	defer closeDB()

	in := Budget{CategoryID: "c1", Limit: 1000000, Month: "2026-09"}
	mock.ExpectExec(`INSERT INTO budgets \(id, user_id, category_id, limit_amount, month\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE limit_amount = VALUES\(limit_amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", in.CategoryID, in.Limit, in.Month).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets\s+WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs("u1", in.CategoryID, in.Month).
		WillReturnRows(sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
			AddRow("b1", in.CategoryID, in.Limit, in.Month))

	out, err := repo.Upsert(context.Background(), "u1", in)
	require.NoError(t, err)
	assert.Equal(t, Budget{ID: "b1", CategoryID: "c1", Limit: 1000000, Month: "2026-09"}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestBudgetRepo_Upsert_UpdateInPlace pins the ON-DUPLICATE-KEY re-write path:
// upserting the same category+month twice with a different amount must leave
// exactly one row, reflecting the latest amount (I/O matrix: "Budget upsert
// idempotency").
func TestBudgetRepo_Upsert_UpdateInPlace(t *testing.T) {
	repo, mock, closeDB := newMockBudgetRepo(t)
	defer closeDB()

	first := Budget{CategoryID: "c1", Limit: 1000000, Month: "2026-09"}
	mock.ExpectExec(`INSERT INTO budgets \(id, user_id, category_id, limit_amount, month\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE limit_amount = VALUES\(limit_amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", first.CategoryID, first.Limit, first.Month).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets\s+WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs("u1", first.CategoryID, first.Month).
		WillReturnRows(sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
			AddRow("b1", first.CategoryID, first.Limit, first.Month))

	firstOut, err := repo.Upsert(context.Background(), "u1", first)
	require.NoError(t, err)
	assert.Equal(t, int64(1000000), firstOut.Limit)

	second := Budget{CategoryID: "c1", Limit: 2500000, Month: "2026-09"}
	mock.ExpectExec(`INSERT INTO budgets \(id, user_id, category_id, limit_amount, month\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE limit_amount = VALUES\(limit_amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", second.CategoryID, second.Limit, second.Month).
		WillReturnResult(sqlmock.NewResult(0, 2)) // MySQL reports 2 affected rows for an ON DUPLICATE KEY UPDATE
	// Re-SELECT-after-write returns the SAME row id ("b1") with the new amount,
	// proving it was updated in place rather than inserted as a second row.
	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets\s+WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs("u1", second.CategoryID, second.Month).
		WillReturnRows(sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
			AddRow("b1", second.CategoryID, second.Limit, second.Month))

	secondOut, err := repo.Upsert(context.Background(), "u1", second)
	require.NoError(t, err)
	assert.Equal(t, "b1", secondOut.ID, "id must stay the same row, not a newly inserted one")
	assert.Equal(t, int64(2500000), secondOut.Limit, "list should reflect the latest amount")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestBudgetRepo_GetByCategoryMonth_Found pins the ordinary path: a budget
// row exists for the given category/month.
func TestBudgetRepo_GetByCategoryMonth_Found(t *testing.T) {
	repo, mock, closeDB := newMockBudgetRepo(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs("u1", "c1", "2026-09").
		WillReturnRows(sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
			AddRow("b1", "c1", int64(1000000), "2026-09"))

	out, err := repo.GetByCategoryMonth(context.Background(), "u1", "c1", "2026-09")
	require.NoError(t, err)
	assert.Equal(t, Budget{ID: "b1", CategoryID: "c1", Limit: 1000000, Month: "2026-09"}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestBudgetRepo_GetByCategoryMonth_NotFound pins the I/O matrix's "No
// budget set" scenario: sql.ErrNoRows propagates as-is (not wrapped, not
// swallowed) so callers can distinguish it via ignoreNoRows/errors.Is.
func TestBudgetRepo_GetByCategoryMonth_NotFound(t *testing.T) {
	repo, mock, closeDB := newMockBudgetRepo(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs("u1", "c1", "2026-09").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByCategoryMonth(context.Background(), "u1", "c1", "2026-09")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBudgetRepo_Delete(t *testing.T) {
	repo, mock, closeDB := newMockBudgetRepo(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM budgets WHERE id = \? AND user_id = \?`).
		WithArgs("b1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), "u1", "b1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
