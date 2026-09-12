package api

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTransactions_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "date"})
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, date\s+FROM transactions WHERE user_id = \? ORDER BY date DESC, created_at DESC`).
		WithArgs("u1").
		WillReturnRows(rows)

	w, c := authedRequest("GET", "/api/transactions", "", "u1", nil)
	h.ListTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateTransaction_OwnershipFailure pins the I/O matrix's "Category
// ownership on transaction create" scenario: User A referencing a category
// that does not resolve under User A's id (e.g. it belongs to User B) must be
// rejected before TransactionRepo.Create ever runs.
func TestCreateTransaction_OwnershipFailure(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("categoryOfUserB", "userA").
		WillReturnError(sql.ErrNoRows)

	body := `{"type":"expense","amount":1000,"categoryId":"categoryOfUserB","date":"2026-09-12"}`
	w, c := authedRequest("POST", "/api/transactions", body, "userA", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40021`)
	// The failure envelope must carry no data, and Create must never be called
	// — mock.ExpectationsWereMet below proves no unexpected INSERT happened
	// (sqlmock fails the test if a query runs that wasn't expected).
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("c1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(1000), "c1", "", "2026-09-12").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"type":"expense","amount":1000,"categoryId":"c1","date":"2026-09-12"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"2026-09-12"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTransaction_NotFound(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("c1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectExec(`UPDATE transactions SET type = \?, amount = \?, category_id = \?, note = \?, date = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs("expense", int64(500), "c1", "", "2026-09-01", "missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	body := `{"type":"expense","amount":500,"categoryId":"c1","date":"2026-09-01"}`
	w, c := authedRequest("PUT", "/api/transactions/missing", body, "u1", ginParams("id", "missing"))
	h.UpdateTransaction(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40420`)
}

func TestDeleteTransaction_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM transactions WHERE id = \? AND user_id = \?`).
		WithArgs("t1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w, c := authedRequest("DELETE", "/api/transactions/t1", "", "u1", ginParams("id", "t1"))
	h.DeleteTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"deleted":true`)
}
