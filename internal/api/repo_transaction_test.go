package api

import (
	"context"
	"database/sql"
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

// TestTransactionRepo_SumExpensesInRange_SomeInRange pins the ordinary path:
// the query sums only expense-type rows within [from, to).
func TestTransactionRepo_SumExpensesInRange_SomeInRange(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	from := mustDate("2026-09-01")
	to := mustDate("2026-10-01")
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND type = 'expense' AND date >= \? AND date < \?`).
		WithArgs("u1", "2026-09-01", "2026-10-01").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(350000)))

	sum, err := repo.SumExpensesInRange(context.Background(), "u1", from, to)
	require.NoError(t, err)
	assert.Equal(t, int64(350000), sum)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTransactionRepo_SumExpensesInRange_NoneInRange pins the "no expenses
// yet this cycle" I/O matrix scenario: COALESCE keeps this 0, not an error.
func TestTransactionRepo_SumExpensesInRange_NoneInRange(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	from := mustDate("2026-09-01")
	to := mustDate("2026-10-01")
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND type = 'expense' AND date >= \? AND date < \?`).
		WithArgs("u1", "2026-09-01", "2026-10-01").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))

	sum, err := repo.SumExpensesInRange(context.Background(), "u1", from, to)
	require.NoError(t, err)
	assert.Equal(t, int64(0), sum)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTransactionRepo_SumExpensesInRange_ExcludesOutOfRangeDate pins that a
// transaction dated outside [from, to) is excluded from the sum: seeded here
// is one expense inside the range (100000) and one outside it on 2026-08-31
// (999999), so the correctly-filtered sum the DB would return is only the
// in-range amount.
func TestTransactionRepo_SumExpensesInRange_ExcludesOutOfRangeDate(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	from := mustDate("2026-09-01")
	to := mustDate("2026-10-01")
	// Seeded: expense "in" (2026-09-15, 100000) inside the range, expense
	// "out" (2026-08-31, 999999) outside it — only "in" should count.
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND type = 'expense' AND date >= \? AND date < \?`).
		WithArgs("u1", "2026-09-01", "2026-10-01").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(100000)))

	sum, err := repo.SumExpensesInRange(context.Background(), "u1", from, to)
	require.NoError(t, err)
	assert.Equal(t, int64(100000), sum, "the out-of-range transaction must not be included in the sum")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTransactionRepo_SumExpensesInRange_ExcludesIncomeType pins that an
// income-type transaction inside the range is excluded from the sum: seeded
// here is one expense (100000) and one income (500000), both dated inside
// the range, so the correctly-filtered sum the DB would return counts only
// the expense.
func TestTransactionRepo_SumExpensesInRange_ExcludesIncomeType(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	from := mustDate("2026-09-01")
	to := mustDate("2026-10-01")
	// Seeded: expense (100000) and income (500000), both dated 2026-09-15 —
	// only the expense should count.
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND type = 'expense' AND date >= \? AND date < \?`).
		WithArgs("u1", "2026-09-01", "2026-10-01").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(100000)))

	sum, err := repo.SumExpensesInRange(context.Background(), "u1", from, to)
	require.NoError(t, err)
	assert.Equal(t, int64(100000), sum, "the income-type transaction must not be included in the sum")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTransactionRepo_SumExpensesInRange_QueryError pins that a genuine
// query error propagates instead of being swallowed into 0.
func TestTransactionRepo_SumExpensesInRange_QueryError(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	from := mustDate("2026-09-01")
	to := mustDate("2026-10-01")
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND type = 'expense' AND date >= \? AND date < \?`).
		WithArgs("u1", "2026-09-01", "2026-10-01").
		WillReturnError(sql.ErrConnDone)

	_, err := repo.SumExpensesInRange(context.Background(), "u1", from, to)
	assert.ErrorIs(t, err, sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTransactionRepo_SumExpensesInCategoryExcluding_ExcludesOwnRow pins the
// spec's core requirement: the transaction's own id (already persisted by
// the time this runs, per checkBudgetThreshold's contract) is excluded from
// the sum via "AND id != ?", along with category/type/date filtering.
func TestTransactionRepo_SumExpensesInCategoryExcluding_ExcludesOwnRow(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	from := mustDate("2026-09-01")
	to := mustDate("2026-10-01")
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND category_id = \? AND type = 'expense' AND date >= \? AND date < \? AND id != \?`).
		WithArgs("u1", "c1", "2026-09-01", "2026-10-01", "t1").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(400000)))

	sum, err := repo.SumExpensesInCategoryExcluding(context.Background(), "u1", "c1", from, to, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(400000), sum)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTransactionRepo_SumExpensesInCategoryExcluding_NoOtherTransactions pins
// the zero-case: COALESCE keeps the sum 0, not an error, when the excluded
// row is the only one in the category/cycle.
func TestTransactionRepo_SumExpensesInCategoryExcluding_NoOtherTransactions(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	from := mustDate("2026-09-01")
	to := mustDate("2026-10-01")
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND category_id = \? AND type = 'expense' AND date >= \? AND date < \? AND id != \?`).
		WithArgs("u1", "c1", "2026-09-01", "2026-10-01", "t1").
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))

	sum, err := repo.SumExpensesInCategoryExcluding(context.Background(), "u1", "c1", from, to, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), sum)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTransactionRepo_SumExpensesInCategoryExcluding_QueryError pins that a
// genuine query error propagates instead of being swallowed into 0.
func TestTransactionRepo_SumExpensesInCategoryExcluding_QueryError(t *testing.T) {
	repo, mock, closeDB := newMockTransactionRepo(t)
	defer closeDB()

	from := mustDate("2026-09-01")
	to := mustDate("2026-10-01")
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND category_id = \? AND type = 'expense' AND date >= \? AND date < \? AND id != \?`).
		WithArgs("u1", "c1", "2026-09-01", "2026-10-01", "t1").
		WillReturnError(sql.ErrConnDone)

	_, err := repo.SumExpensesInCategoryExcluding(context.Background(), "u1", "c1", from, to, "t1")
	assert.ErrorIs(t, err, sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())
}
