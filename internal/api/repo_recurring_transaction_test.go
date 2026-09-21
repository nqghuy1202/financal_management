package api

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockRecurringTransactionRepo(t *testing.T) (*RecurringTransactionRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewRecurringTransactionRepo(db), mock, func() { db.Close() }
}

func TestRecurringTransactionRepo_List(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "frequency", "next_due_date", "active"}).
		AddRow("r1", "expense", int64(5000000), "c1", "Rent", "monthly", mustDate("2026-10-01"), true)
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE user_id = \? ORDER BY next_due_date, created_at`).
		WithArgs("u1").
		WillReturnRows(rows)

	list, err := repo.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []RecurringTransaction{
		{ID: "r1", Type: "expense", Amount: 5000000, CategoryID: "c1", Note: "Rent", Frequency: "monthly", NextDueDate: "2026-10-01", Active: true},
	}, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecurringTransactionRepo_Get_Found(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "frequency", "next_due_date", "active"}).
		AddRow("r1", "expense", int64(5000000), "c1", "Rent", "monthly", mustDate("2026-10-01"), true)
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("r1", "u1").
		WillReturnRows(rows)

	rt, err := repo.Get(context.Background(), "u1", "r1")
	require.NoError(t, err)
	assert.Equal(t, RecurringTransaction{ID: "r1", Type: "expense", Amount: 5000000, CategoryID: "c1", Note: "Rent", Frequency: "monthly", NextDueDate: "2026-10-01", Active: true}, rt)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestRecurringTransactionRepo_Get_NotFound pins that a missing/not-owned
// template comes back as a zero-value result (ID == "") rather than an
// error, mirroring CategoryRepo.Owns — callers branch on rt.ID, not on err.
func TestRecurringTransactionRepo_Get_NotFound(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("missing", "u1").
		WillReturnError(sql.ErrNoRows)

	rt, err := repo.Get(context.Background(), "u1", "missing")
	require.NoError(t, err)
	assert.Equal(t, RecurringTransaction{}, rt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecurringTransactionRepo_Create(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	rt := RecurringTransaction{ID: "r1", Type: "expense", Amount: 5000000, CategoryID: "c1", Note: "Rent", Frequency: "monthly", NextDueDate: "2026-10-01", Active: true}
	mock.ExpectExec(`INSERT INTO recurring_transactions \(id, user_id, type, amount, category_id, note, frequency, next_due_date, active\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(rt.ID, "u1", rt.Type, rt.Amount, rt.CategoryID, rt.Note, rt.Frequency, rt.NextDueDate, rt.Active).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), "u1", rt)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecurringTransactionRepo_Update_Found(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	rt := RecurringTransaction{Type: "expense", Amount: 6000000, CategoryID: "c1", Note: "Rent", Frequency: "monthly", Active: false}
	mock.ExpectExec(`UPDATE recurring_transactions SET type = \?, amount = \?, category_id = \?, note = \?, frequency = \?, active = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs(rt.Type, rt.Amount, rt.CategoryID, rt.Note, rt.Frequency, rt.Active, "r1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	found, err := repo.Update(context.Background(), "u1", "r1", rt)
	require.NoError(t, err)
	assert.True(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecurringTransactionRepo_Update_NotFound(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	rt := RecurringTransaction{Type: "expense", Amount: 1000, CategoryID: "c1", Frequency: "weekly", Active: true}
	mock.ExpectExec(`UPDATE recurring_transactions SET type = \?, amount = \?, category_id = \?, note = \?, frequency = \?, active = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs(rt.Type, rt.Amount, rt.CategoryID, rt.Note, rt.Frequency, rt.Active, "missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	found, err := repo.Update(context.Background(), "u1", "missing", rt)
	require.NoError(t, err)
	assert.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecurringTransactionRepo_Delete_Found(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("r1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	found, err := repo.Delete(context.Background(), "u1", "r1")
	require.NoError(t, err)
	assert.True(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecurringTransactionRepo_Delete_NotFound(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	found, err := repo.Delete(context.Background(), "u1", "missing")
	require.NoError(t, err)
	assert.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecurringTransactionRepo_AdvanceNextDueDate(t *testing.T) {
	repo, mock, closeDB := newMockRecurringTransactionRepo(t)
	defer closeDB()

	mock.ExpectExec(`UPDATE recurring_transactions SET next_due_date = \? WHERE id = \? AND user_id = \?`).
		WithArgs("2026-11-01", "r1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.AdvanceNextDueDate(context.Background(), "u1", "r1", "2026-11-01")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
