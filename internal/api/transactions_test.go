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

// ---- Story 2.1 test fixtures & helpers ----
//
// All threshold-check tests compute the current cycle dynamically from
// time.Now() (cycleStartDay=1), mirroring cycle_settings_test.go's style, so
// they never depend on the wall-clock date they happen to run on.

func currentCycleFixture() (cycleStart, cycleEnd time.Time, dateInCycle, month string) {
	cycleStart, cycleEnd = CycleWindow(1, time.Now())
	return cycleStart, cycleEnd, cycleStart.Format(dateLayout), cycleStart.Format("2006-01")
}

func expectOwnsCategory(mock sqlmock.Sqlmock, categoryID, userID string) {
	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs(categoryID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
}

func expectBudgetFound(mock sqlmock.Sqlmock, userID, categoryID, month string, limit int64) {
	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs(userID, categoryID, month).
		WillReturnRows(sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
			AddRow("b1", categoryID, limit, month))
}

func expectBudgetNotFound(mock sqlmock.Sqlmock, userID, categoryID, month string) {
	mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = \? AND category_id = \? AND month = \?`).
		WithArgs(userID, categoryID, month).
		WillReturnError(sql.ErrNoRows)
}

func expectSumExcluding(mock sqlmock.Sqlmock, userID, categoryID string, cycleStart, cycleEnd time.Time, excludeID string, prevSpent int64) {
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND category_id = \? AND type = 'expense' AND date >= \? AND date < \? AND id != \?`).
		WithArgs(userID, categoryID, cycleStart.Format(dateLayout), cycleEnd.Format(dateLayout), excludeID).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(prevSpent))
}

func expectAlertTrigger(mock sqlmock.Sqlmock, userID, categoryID string, cycleStart time.Time, threshold int) {
	mock.ExpectExec(`INSERT INTO budget_alert_state \(id, user_id, category_id, cycle_start_date, threshold\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE threshold = VALUES\(threshold\)`).
		WithArgs(sqlmock.AnyArg(), userID, categoryID, cycleStart.Format(dateLayout), threshold).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

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

// TestCreateTransaction_HappyPath pins the "No budget set" I/O matrix
// scenario for create: the category has no budget row for the current
// cycle's month, so the threshold check runs (settings + budget lookup) but
// stops there — no sum query, no alert row, and the write still commits.
func TestCreateTransaction_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	_, _, dateInCycle, month := currentCycleFixture()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(1000), "c1", "", dateInCycle).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	expectBudgetNotFound(mock, "u1", "c1", month)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":1000,"categoryId":"c1","date":"` + dateInCycle + `"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"`+dateInCycle+`"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateTransaction_IncomeType_SkipsCheck pins the I/O matrix's "Income
// transaction" scenario: an income-type transaction never triggers the
// threshold check at all — no settings lookup, no budget lookup.
func TestCreateTransaction_IncomeType_SkipsCheck(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	_, _, dateInCycle, _ := currentCycleFixture()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "income", int64(500000), "c1", "", dateInCycle).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body := `{"type":"income","amount":500000,"categoryId":"c1","date":"` + dateInCycle + `"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateTransaction_BackdatedDate_SkipsCheck pins the I/O matrix's
// "Backdated transaction" scenario: the transaction's date falls outside
// [cycleStart, cycleEnd), so settings are fetched (to compute the window)
// but the budget lookup never runs.
func TestCreateTransaction_BackdatedDate_SkipsCheck(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, _, _, _ := currentCycleFixture()
	backdated := cycleStart.AddDate(0, 0, -1).Format(dateLayout)

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(1000), "c1", "", backdated).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":1000,"categoryId":"c1","date":"` + backdated + `"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateTransaction_FutureCycleDate_SkipsCheck pins the symmetric half of
// TestCreateTransaction_BackdatedDate_SkipsCheck's cycle-window boundary: a
// transaction dated on/after cycleEnd also falls outside
// [cycleStart, cycleEnd), so settings are fetched (to compute the window) but
// the budget lookup never runs.
func TestCreateTransaction_FutureCycleDate_SkipsCheck(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	_, cycleEnd, _, _ := currentCycleFixture()
	futureDated := cycleEnd.Format(dateLayout)

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(1000), "c1", "", futureDated).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":1000,"categoryId":"c1","date":"` + futureDated + `"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateTransaction_FirstCrossing pins the I/O matrix's "First crossing"
// scenario end-to-end through the handler: a budgeted category whose spend
// (excluding the just-inserted row) sits below 70% is pushed past it by this
// save, writing exactly one budget_alert_state row for threshold 70.
func TestCreateTransaction_FirstCrossing(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, cycleEnd, dateInCycle, month := currentCycleFixture()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(150000), "c1", "", dateInCycle).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	expectBudgetFound(mock, "u1", "c1", month, 1000000)
	// prevSpent (excluding the just-inserted row) = 600,000; +150,000 = 750,000 = 75% > 70%.
	expectSumExcludingAnyID(mock, "u1", "c1", cycleStart, cycleEnd, 600000)
	expectAlertTriggerAnyID(mock, "u1", "c1", cycleStart, 70)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":150000,"categoryId":"c1","date":"` + dateInCycle + `"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateTransaction_MultiThresholdJump pins the I/O matrix's
// "Multi-threshold jump" scenario: one transaction pushes spend from 40% to
// 105%, and only the 100% alert row is written — never separate 70/90/100
// rows.
func TestCreateTransaction_MultiThresholdJump(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, cycleEnd, dateInCycle, month := currentCycleFixture()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(650000), "c1", "", dateInCycle).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	expectBudgetFound(mock, "u1", "c1", month, 1000000)
	// prevSpent (excluding the just-inserted row) = 400,000 (40%); +650,000 = 1,050,000 (105%).
	expectSumExcludingAnyID(mock, "u1", "c1", cycleStart, cycleEnd, 400000)
	expectAlertTriggerAnyID(mock, "u1", "c1", cycleStart, 100)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":650000,"categoryId":"c1","date":"` + dateInCycle + `"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateTransaction_AlreadyAlertedThreshold_NoNewRow pins the I/O
// matrix's "Already alerted at this threshold" scenario: spend stays above
// 90% but below 100%, so no new alert row is written (whether or not a 90%
// row already exists is irrelevant to this recomputation — the point is no
// *new* row is written for a threshold not newly crossed).
func TestCreateTransaction_AlreadyAlertedThreshold_NoNewRow(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, cycleEnd, dateInCycle, month := currentCycleFixture()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(30000), "c1", "", dateInCycle).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	expectBudgetFound(mock, "u1", "c1", month, 1000000)
	// prevSpent (excluding the just-inserted row) = 950,000 (95%); +30,000 = 980,000 (98%) — no new crossing.
	expectSumExcludingAnyID(mock, "u1", "c1", cycleStart, cycleEnd, 950000)
	// No INSERT INTO budget_alert_state expected.
	mock.ExpectCommit()

	body := `{"type":"expense","amount":30000,"categoryId":"c1","date":"` + dateInCycle + `"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateTransaction_ThresholdCheckFailure_RollsBack pins that a failure
// inside the threshold check (here, the settings lookup) rolls back the
// whole transaction — including the just-inserted row — rather than leaving
// a half-committed save, exercising h.withTx's generic rollback-on-error
// path for the create handler specifically.
func TestCreateTransaction_ThresholdCheckFailure_RollsBack(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	_, _, dateInCycle, _ := currentCycleFixture()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "expense", int64(1000), "c1", "", dateInCycle).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	body := `{"type":"expense","amount":1000,"categoryId":"c1","date":"` + dateInCycle + `"}`
	w, c := authedRequest("POST", "/api/transactions", body, "u1", nil)
	h.CreateTransaction(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50022`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTransaction_NotFound(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE transactions SET type = \?, amount = \?, category_id = \?, note = \?, date = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs("expense", int64(500), "c1", "", "2026-09-01", "missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	// No rows matched: the threshold check must not run, but the (no-op) tx
	// still commits — the 404 comes from the handler's found==false check,
	// not from a rollback.
	mock.ExpectCommit()

	body := `{"type":"expense","amount":500,"categoryId":"c1","date":"2026-09-01"}`
	w, c := authedRequest("PUT", "/api/transactions/missing", body, "u1", ginParams("id", "missing"))
	h.UpdateTransaction(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40420`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateTransaction_HappyPath pins the ordinary update path with no
// budget set for the category (skip check), mirroring
// TestCreateTransaction_HappyPath for the update handler.
func TestUpdateTransaction_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	_, _, dateInCycle, month := currentCycleFixture()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE transactions SET type = \?, amount = \?, category_id = \?, note = \?, date = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs("expense", int64(2000), "c1", "n", dateInCycle, "t1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	expectBudgetNotFound(mock, "u1", "c1", month)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":2000,"categoryId":"c1","note":"n","date":"` + dateInCycle + `"}`
	w, c := authedRequest("PUT", "/api/transactions/t1", body, "u1", ginParams("id", "t1"))
	h.UpdateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateTransaction_ThresholdCrossing_ExcludesOwnRow pins the spec's
// core bad_spec fix: on update, the row's own id ("t1") — which already
// reflects the new amount by the time the threshold check runs — must be
// excluded from prevSpent via SumExpensesInCategoryExcluding, never "".
func TestUpdateTransaction_ThresholdCrossing_ExcludesOwnRow(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, cycleEnd, dateInCycle, month := currentCycleFixture()

	expectOwnsCategory(mock, "c1", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE transactions SET type = \?, amount = \?, category_id = \?, note = \?, date = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs("expense", int64(150000), "c1", "", dateInCycle, "t1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	expectBudgetFound(mock, "u1", "c1", month, 1000000)
	// The sum query must explicitly exclude "t1" (the row just updated).
	expectSumExcluding(mock, "u1", "c1", cycleStart, cycleEnd, "t1", 600000)
	expectAlertTrigger(mock, "u1", "c1", cycleStart, 70)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":150000,"categoryId":"c1","date":"` + dateInCycle + `"}`
	w, c := authedRequest("PUT", "/api/transactions/t1", body, "u1", ginParams("id", "t1"))
	h.UpdateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateTransaction_CategoryChange_CrossesThresholdInNewCategory pins the
// spec's Design Notes claim that moving a transaction to a different category
// is handled correctly: the threshold check must run against the *new*
// category's budget/spend (not the old one) while still excluding the row's
// own id from that new category's prevSpent sum.
func TestUpdateTransaction_CategoryChange_CrossesThresholdInNewCategory(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, cycleEnd, dateInCycle, month := currentCycleFixture()

	expectOwnsCategory(mock, "cNew", "u1")
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE transactions SET type = \?, amount = \?, category_id = \?, note = \?, date = \?\s+WHERE id = \? AND user_id = \?`).
		WithArgs("expense", int64(150000), "cNew", "", dateInCycle, "t1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectSettingsGet(mock, "u1", 0, 1)
	// Budget/spend lookups must target the new category ("cNew"), never the
	// transaction's previous category.
	expectBudgetFound(mock, "u1", "cNew", month, 1000000)
	// The sum query must be scoped to "cNew" and still exclude "t1" (the row
	// just moved into it).
	expectSumExcluding(mock, "u1", "cNew", cycleStart, cycleEnd, "t1", 600000)
	expectAlertTrigger(mock, "u1", "cNew", cycleStart, 70)
	mock.ExpectCommit()

	body := `{"type":"expense","amount":150000,"categoryId":"cNew","date":"` + dateInCycle + `"}`
	w, c := authedRequest("PUT", "/api/transactions/t1", body, "u1", ginParams("id", "t1"))
	h.UpdateTransaction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
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

// ---- helpers for create-path tests, where the transaction id is a fresh
// uuid generated inside the handler (unknown to the test in advance) ----

func expectSumExcludingAnyID(mock sqlmock.Sqlmock, userID, categoryID string, cycleStart, cycleEnd time.Time, prevSpent int64) {
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND category_id = \? AND type = 'expense' AND date >= \? AND date < \? AND id != \?`).
		WithArgs(userID, categoryID, cycleStart.Format(dateLayout), cycleEnd.Format(dateLayout), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(prevSpent))
}

func expectAlertTriggerAnyID(mock sqlmock.Sqlmock, userID, categoryID string, cycleStart time.Time, threshold int) {
	mock.ExpectExec(`INSERT INTO budget_alert_state \(id, user_id, category_id, cycle_start_date, threshold\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE threshold = VALUES\(threshold\)`).
		WithArgs(sqlmock.AnyArg(), userID, categoryID, cycleStart.Format(dateLayout), threshold).
		WillReturnResult(sqlmock.NewResult(1, 1))
}
