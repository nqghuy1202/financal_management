package api

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockTransactionRepo(t *testing.T) (*TransactionRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewTransactionRepo(db), mock, func() { db.Close() }
}

func TestTransactionRepo_List_DateRoundTrip(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	d, err := time.Parse(dateLayout, "2026-09-12")
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "date"}).
		AddRow("t1", "expense", int64(50000), "c1", "trưa", d)
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, date\s+FROM transactions WHERE user_id = \? ORDER BY date DESC, created_at DESC`).
		WithArgs("u1").
		WillReturnRows(rows)

	list, err := repo.List(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "2026-09-12", list[0].Date, "date must round-trip with no TZ drift")
	assert.Equal(t, Transaction{ID: "t1", Type: "expense", Amount: 50000, CategoryID: "c1", Note: "trưa", Date: "2026-09-12"}, list[0])
}

func TestTransactionRepo_Create(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	tx := Transaction{ID: "t1", Type: "income", Amount: 1000, CategoryID: "c1", Note: "", Date: "2026-09-12"}
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(tx.ID, "u1", tx.Type, tx.Amount, tx.CategoryID, tx.Note, tx.Date).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), "u1", tx)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepo_Update(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo, mock, closeDB := newMockTransactionRepo(t)
		defer closeDB()

		tx := Transaction{Type: "expense", Amount: 2000, CategoryID: "c2", Note: "n", Date: "2026-09-01"}
		mock.ExpectExec(`UPDATE transactions SET type = \?, amount = \?, category_id = \?, note = \?, date = \?\s+WHERE id = \? AND user_id = \?`).
			WithArgs(tx.Type, tx.Amount, tx.CategoryID, tx.Note, tx.Date, "t1", "u1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		found, err := repo.Update(context.Background(), "u1", "t1", tx)
		require.NoError(t, err)
		assert.True(t, found)
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock, closeDB := newMockTransactionRepo(t)
		defer closeDB()

		tx := Transaction{Type: "expense", Amount: 2000, CategoryID: "c2", Note: "n", Date: "2026-09-01"}
		mock.ExpectExec(`UPDATE transactions SET type = \?, amount = \?, category_id = \?, note = \?, date = \?\s+WHERE id = \? AND user_id = \?`).
			WithArgs(tx.Type, tx.Amount, tx.CategoryID, tx.Note, tx.Date, "missing", "u1").
			WillReturnResult(sqlmock.NewResult(0, 0))

		found, err := repo.Update(context.Background(), "u1", "missing", tx)
		require.NoError(t, err)
		assert.False(t, found)
	})
}

func TestTransactionRepo_Delete(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM transactions WHERE id = ? AND user_id = ?`)).
		WithArgs("t1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), "u1", "t1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
