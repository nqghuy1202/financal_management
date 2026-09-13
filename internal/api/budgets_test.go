package api

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectSumExpensesInCategory mocks TransactionRepo.SumExpensesInCategory
// (Story 2.3's non-excluding spend sum, used by GET/POST /budgets).
func expectSumExpensesInCategory(mock sqlmock.Sqlmock, userID, categoryID, from, to string, spent int64) {
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND category_id = \? AND type = 'expense' AND date >= \? AND date < \?`).
		WithArgs(userID, categoryID, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(spent))
}

func TestListBudgets_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
		AddRow("b1", "c1", int64(3000000), "2026-09")
	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(rows)
	expectSettingsGet(mock, "u1", 0, 1)
	expectSumExpensesInCategory(mock, "u1", "c1", "2026-09-01", "2026-10-01", 2100000)

	w, c := authedRequest("GET", "/api/budgets", "", "u1", nil)
	h.ListBudgets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	assert.Contains(t, w.Body.String(), `"spent":2100000`)
	assert.Contains(t, w.Body.String(), `"percent":70`)
	assert.Contains(t, w.Body.String(), `"status":"near"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpsertBudget_OwnershipFailure: a budget referencing a category that
// doesn't resolve under the caller's user id must be rejected before
// BudgetRepo.Upsert runs.
func TestUpsertBudget_OwnershipFailure(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("categoryOfUserB", "userA").
		WillReturnError(sql.ErrNoRows)

	body := `{"categoryId":"categoryOfUserB","limit":1000000,"month":"2026-09"}`
	w, c := authedRequest("POST", "/api/budgets", body, "userA", nil)
	h.UpsertBudget(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40032`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpsertBudget_OwnershipCheckDBError: a genuine DB error while checking
// category ownership (as opposed to sql.ErrNoRows) must surface as a 500,
// not be mistaken for the "category not found" 400 case.
func TestUpsertBudget_OwnershipCheckDBError(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("c1", "u1").
		WillReturnError(sql.ErrConnDone)

	body := `{"categoryId":"c1","limit":1000000,"month":"2026-09"}`
	w, c := authedRequest("POST", "/api/budgets", body, "u1", nil)
	h.UpsertBudget(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50034`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBudget_ValidationFailure(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	// missing/invalid month format — rejected before touching the DB at all.
	body := `{"categoryId":"c1","limit":1000000,"month":"bad-month"}`
	w, c := authedRequest("POST", "/api/budgets", body, "u1", nil)
	h.UpsertBudget(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40031`)
}

// TestUpsertBudget_Idempotency pins the I/O matrix's "Budget upsert
// idempotency" scenario at the handler level: PUT-ing (POST here, per the
// route) the same category+month twice with different amounts must both
// succeed and reflect the latest amount.
func TestUpsertBudget_Idempotency(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("c1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectExec(`INSERT INTO budgets \(id, user_id, category_id, limit_amount, month\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE limit_amount = VALUES\(limit_amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "c1", int64(1000000), "2026-09").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets\s+WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs("u1", "c1", "2026-09").
		WillReturnRows(sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
			AddRow("b1", "c1", int64(1000000), "2026-09"))
	expectSettingsGet(mock, "u1", 0, 1)
	expectSumExpensesInCategory(mock, "u1", "c1", "2026-09-01", "2026-10-01", 0)

	body1 := `{"categoryId":"c1","limit":1000000,"month":"2026-09"}`
	w1, c1 := authedRequest("POST", "/api/budgets", body1, "u1", nil)
	h.UpsertBudget(c1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), `"limit":1000000`)

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("c1", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectExec(`INSERT INTO budgets \(id, user_id, category_id, limit_amount, month\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE limit_amount = VALUES\(limit_amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "c1", int64(2500000), "2026-09").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets\s+WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs("u1", "c1", "2026-09").
		WillReturnRows(sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
			AddRow("b1", "c1", int64(2500000), "2026-09"))
	expectSettingsGet(mock, "u1", 0, 1)
	expectSumExpensesInCategory(mock, "u1", "c1", "2026-09-01", "2026-10-01", 0)

	body2 := `{"categoryId":"c1","limit":2500000,"month":"2026-09"}`
	w2, c2 := authedRequest("POST", "/api/budgets", body2, "u1", nil)
	h.UpsertBudget(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), `"id":"b1"`)
	assert.Contains(t, w2.Body.String(), `"limit":2500000`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteBudget_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM budgets WHERE id = \? AND user_id = \?`).
		WithArgs("b1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w, c := authedRequest("DELETE", "/api/budgets/b1", "", "u1", ginParams("id", "b1"))
	h.DeleteBudget(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"deleted":true`)
	require.NoError(t, mock.ExpectationsWereMet())
}
