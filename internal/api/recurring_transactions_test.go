package api

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRecurringTransactions_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "frequency", "next_due_date", "active"})
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE user_id = \? ORDER BY next_due_date, created_at`).
		WithArgs("u1").
		WillReturnRows(rows)

	w, c := authedRequest("GET", "/api/recurring-transactions", "", "u1", nil)
	h.ListRecurringTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListRecurringTransactions_WithDueDraft pins that ListRecurringTransactions
// actually runs attachDueDraft over real rows (not just the empty-list case):
// a row whose next_due_date is today and active=true must come back with
// dueDraftDate set to that same date.
func TestListRecurringTransactions_WithDueDraft(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	today := time.Now().Format(dateLayout)
	rows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "frequency", "next_due_date", "active"}).
		AddRow("r1", "expense", int64(5000000), "c1", "Rent", "monthly", mustDate(today), true)
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE user_id = \? ORDER BY next_due_date, created_at`).
		WithArgs("u1").
		WillReturnRows(rows)

	w, c := authedRequest("GET", "/api/recurring-transactions", "", "u1", nil)
	h.ListRecurringTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"dueDraftDate":"`+today+`"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateRecurringTransaction_ValidationFailure(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	// invalid frequency — rejected before any query.
	body := `{"type":"expense","amount":1000,"categoryId":"c1","date":"2026-09-21","frequency":"daily"}`
	w, c := authedRequest("POST", "/api/recurring-transactions", body, "u1", nil)
	h.CreateRecurringTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40060`)
}

func TestCreateRecurringTransaction_OwnershipFailure(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("categoryOfUserB", "userA").
		WillReturnError(sql.ErrNoRows)

	body := `{"type":"expense","amount":1000,"categoryId":"categoryOfUserB","date":"2026-09-21","frequency":"weekly"}`
	w, c := authedRequest("POST", "/api/recurring-transactions", body, "userA", nil)
	h.CreateRecurringTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40061`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateRecurringTransaction_HappyPath pins the Code Map's
// CreateRecurringTransaction contract: the template's first NextDueDate is
// one frequency step after in.Date (2026-09-21 weekly -> 2026-09-28), and no
// row is ever written to `transactions` by this handler.
func TestCreateRecurringTransaction_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectExec(`INSERT INTO recurring_transactions \(id, user_id, type, amount, category_id, note, frequency, next_due_date, active\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(5000000), "c1", "", "weekly", "2026-09-28", true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"type":"expense","amount":5000000,"categoryId":"c1","date":"2026-09-21","frequency":"weekly"}`
	w, c := authedRequest("POST", "/api/recurring-transactions", body, "u1", nil)
	h.CreateRecurringTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"nextDueDate":"2026-09-28"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateRecurringTransaction_ValidationFailure mirrors
// TestCreateRecurringTransaction_ValidationFailure for the update handler.
func TestUpdateRecurringTransaction_ValidationFailure(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	// invalid frequency — rejected before any query.
	body := `{"type":"expense","amount":1000,"categoryId":"c1","frequency":"daily","active":true}`
	w, c := authedRequest("PUT", "/api/recurring-transactions/r1", body, "u1", ginParams("id", "r1"))
	h.UpdateRecurringTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40062`)
}

// TestUpdateRecurringTransaction_OwnershipFailure mirrors
// TestCreateRecurringTransaction_OwnershipFailure for the update handler.
func TestUpdateRecurringTransaction_OwnershipFailure(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("categoryOfUserB", "userA").
		WillReturnError(sql.ErrNoRows)

	body := `{"type":"expense","amount":1000,"categoryId":"categoryOfUserB","frequency":"weekly","active":true}`
	w, c := authedRequest("PUT", "/api/recurring-transactions/r1", body, "userA", ginParams("id", "r1"))
	h.UpdateRecurringTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40063`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateRecurringTransaction_NotFound(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectExec(`UPDATE recurring_transactions SET type = \?, amount = \?, category_id = \?, note = \?, frequency = \?, active = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs("expense", int64(1000), "c1", "", "weekly", true, "missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	body := `{"type":"expense","amount":1000,"categoryId":"c1","frequency":"weekly","active":true}`
	w, c := authedRequest("PUT", "/api/recurring-transactions/missing", body, "u1", ginParams("id", "missing"))
	h.UpdateRecurringTransaction(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40460`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateRecurringTransaction_Pause pins that pausing is just a PUT with
// active:false — no separate pause endpoint — and that the response reflects
// the canonical row (re-fetched after the write) including its real
// NextDueDate, which this update never touches.
func TestUpdateRecurringTransaction_Pause(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectExec(`UPDATE recurring_transactions SET type = \?, amount = \?, category_id = \?, note = \?, frequency = \?, active = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs("expense", int64(5000000), "c1", "Rent", "monthly", false, "r1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	rows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "frequency", "next_due_date", "active"}).
		AddRow("r1", "expense", int64(5000000), "c1", "Rent", "monthly", mustDate("2026-10-01"), false)
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("r1", "u1").
		WillReturnRows(rows)

	body := `{"type":"expense","amount":5000000,"categoryId":"c1","note":"Rent","frequency":"monthly","active":false}`
	w, c := authedRequest("PUT", "/api/recurring-transactions/r1", body, "u1", ginParams("id", "r1"))
	h.UpdateRecurringTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"active":false`)
	assert.Contains(t, w.Body.String(), `"nextDueDate":"2026-10-01"`)
	// Paused: DueDraftDate must never be attached even if next_due_date is
	// already in the past for this row.
	assert.NotContains(t, w.Body.String(), `"dueDraftDate"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteRecurringTransaction_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("r1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w, c := authedRequest("DELETE", "/api/recurring-transactions/r1", "", "u1", ginParams("id", "r1"))
	h.DeleteRecurringTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"deleted":true`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteRecurringTransaction_NotFound(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	w, c := authedRequest("DELETE", "/api/recurring-transactions/missing", "", "u1", ginParams("id", "missing"))
	h.DeleteRecurringTransaction(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40460`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestConfirmRecurringTransaction_ValidationFailure mirrors
// TestCreateRecurringTransaction_ValidationFailure for the confirm handler.
func TestConfirmRecurringTransaction_ValidationFailure(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	// amount <= 0 — rejected by transactionInput.validate() before any query.
	body := `{"type":"expense","amount":0,"categoryId":"c1","date":"2026-09-21"}`
	w, c := authedRequest("POST", "/api/recurring-transactions/r1/confirm", body, "u1", ginParams("id", "r1"))
	h.ConfirmRecurringTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40064`)
}

// TestConfirmRecurringTransaction_OwnershipFailure mirrors
// TestCreateRecurringTransaction_OwnershipFailure for the confirm handler.
func TestConfirmRecurringTransaction_OwnershipFailure(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("categoryOfUserB", "userA").
		WillReturnError(sql.ErrNoRows)

	body := `{"type":"expense","amount":1000,"categoryId":"categoryOfUserB","date":"2026-09-21"}`
	w, c := authedRequest("POST", "/api/recurring-transactions/r1/confirm", body, "userA", ginParams("id", "r1"))
	h.ConfirmRecurringTransaction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40065`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestConfirmRecurringTransaction_NotFound pins the ownership-check I/O
// matrix row: confirming a template belonging to another user (or already
// deleted) returns 404, and never reaches the transactions INSERT.
func TestConfirmRecurringTransaction_NotFound(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("missing", "u1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":1000,"categoryId":"c1","date":"2026-09-21"}`
	w, c := authedRequest("POST", "/api/recurring-transactions/missing/confirm", body, "u1", ginParams("id", "missing"))
	h.ConfirmRecurringTransaction(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40461`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestConfirmRecurringTransaction_Paused pins the `!rt.Active` branch of the
// not-found check (distinct from the `rt.ID == ""` branch already covered by
// TestConfirmRecurringTransaction_NotFound): a paused template must also be
// rejected as 404, with no INSERT into `transactions` and no next_due_date
// UPDATE ever queued.
func TestConfirmRecurringTransaction_Paused(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	rtRows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "frequency", "next_due_date", "active"}).
		AddRow("r1", "expense", int64(5000000), "c1", "Rent", "monthly", mustDate("2026-06-01"), false)
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("r1", "u1").
		WillReturnRows(rtRows)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":1000,"categoryId":"c1","date":"2026-09-21"}`
	w, c := authedRequest("POST", "/api/recurring-transactions/r1/confirm", body, "u1", ginParams("id", "r1"))
	h.ConfirmRecurringTransaction(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40461`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestConfirmRecurringTransaction_HappyPath pins the core atomic-confirm
// contract end-to-end through the handler: a normal `transactions` row is
// created (no budget set for the category, so the threshold check runs but
// stops there, mirroring TestCreateTransaction_HappyPath), and the
// template's next_due_date advances by one weekly step from its own
// original schedule (due exactly today), never from the edited transaction
// date, which is a different day entirely (I/O matrix "Edit before
// confirm").
func TestConfirmRecurringTransaction_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	_, _, dateInCycle, month := currentCycleFixture()
	templateDueToday := time.Now().Format(dateLayout)
	advancedNextDue := time.Now().AddDate(0, 0, 7).Format(dateLayout)

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	rtRows := sqlmock.NewRows([]string{"id", "type", "amount", "category_id", "note", "frequency", "next_due_date", "active"}).
		AddRow("r1", "expense", int64(5000000), "c1", "Rent", "weekly", mustDate(templateDueToday), true)
	mock.ExpectQuery(`SELECT id, type, amount, COALESCE\(category_id, ''\), note, frequency, next_due_date, active\s+FROM recurring_transactions WHERE id = \? AND user_id = \?`).
		WithArgs("r1", "u1").
		WillReturnRows(rtRows)
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(5000000), "c1", "Rent (edited)", dateInCycle).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	expectBudgetNotFound(mock, "u1", "c1", month)
	mock.ExpectExec(`UPDATE recurring_transactions SET next_due_date = \? WHERE id = \? AND user_id = \?`).
		WithArgs(advancedNextDue, "r1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	body := `{"type":"expense","amount":5000000,"categoryId":"c1","note":"Rent (edited)","date":"` + dateInCycle + `"}`
	w, c := authedRequest("POST", "/api/recurring-transactions/r1/confirm", body, "u1", ginParams("id", "r1"))
	h.ConfirmRecurringTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"Rent (edited)"`)
	require.NoError(t, mock.ExpectationsWereMet())
}
