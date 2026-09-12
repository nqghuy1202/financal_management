package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cycleSummaryFixture computes the cycle boundaries the exact same way
// GetCycleSummary does, so tests can build matching mock expectations and
// expected response values without duplicating CycleWindow/PreviousCycles
// logic (and without depending on a fixed "now").
type cycleSummaryFixture struct {
	cycleStart, cycleEnd time.Time
	prevStart            time.Time
	daysRemaining        int
}

func newCycleSummaryFixture(cycleStartDay int) cycleSummaryFixture {
	now := time.Now()
	start, end := CycleWindow(cycleStartDay, now)
	prev := PreviousCycles(cycleStartDay, now, 1)[0]
	return cycleSummaryFixture{
		cycleStart:    start,
		cycleEnd:      end,
		prevStart:     prev.Start,
		daysRemaining: DaysRemaining(end, now),
	}
}

func expectSettingsGet(mock sqlmock.Sqlmock, userID string, savingsGoal int64, cycleStartDay int) {
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(savingsGoal, cycleStartDay))
}

func expectSettingsGetError(mock sqlmock.Sqlmock, userID string, err error) {
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs(userID).
		WillReturnError(err)
}

func expectIncomeGet(mock sqlmock.Sqlmock, userID, cycleStr string, amount int64) {
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs(userID, cycleStr).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).AddRow("i1", mustDate(cycleStr), amount))
}

func expectIncomeGetNotFound(mock sqlmock.Sqlmock, userID, cycleStr string) {
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs(userID, cycleStr).
		WillReturnError(sql.ErrNoRows)
}

func expectIncomeGetError(mock sqlmock.Sqlmock, userID, cycleStr string, err error) {
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs(userID, cycleStr).
		WillReturnError(err)
}

func expectFixedCostsList(mock sqlmock.Sqlmock, userID string, amounts ...int64) {
	rows := sqlmock.NewRows([]string{"id", "name", "amount"})
	for i, a := range amounts {
		rows.AddRow(fmt.Sprintf("f%d", i), fmt.Sprintf("cost%d", i), a)
	}
	mock.ExpectQuery(`SELECT id, name, amount FROM fixed_costs WHERE user_id = \? ORDER BY created_at`).
		WithArgs(userID).
		WillReturnRows(rows)
}

func expectFixedCostsListError(mock sqlmock.Sqlmock, userID string, err error) {
	mock.ExpectQuery(`SELECT id, name, amount FROM fixed_costs WHERE user_id = \? ORDER BY created_at`).
		WithArgs(userID).
		WillReturnError(err)
}

func expectSumExpenses(mock sqlmock.Sqlmock, userID, from, to string, sum int64) {
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND type = 'expense' AND date >= \? AND date < \?`).
		WithArgs(userID, from, to).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(sum))
}

func expectSumExpensesError(mock sqlmock.Sqlmock, userID, from, to string, err error) {
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\) FROM transactions\s+WHERE user_id = \? AND type = 'expense' AND date >= \? AND date < \?`).
		WithArgs(userID, from, to).
		WillReturnError(err)
}

// TestGetCycleSummary_FullInputsDeclared pins the I/O matrix's main
// scenario: income, fixed costs and savings goal all declared, with some
// expenses already saved this cycle.
func TestGetCycleSummary_FullInputsDeclared(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	fx := newCycleSummaryFixture(1)
	cycleStr := fx.cycleStart.Format(dateLayout)
	prevStr := fx.prevStart.Format(dateLayout)
	endStr := fx.cycleEnd.Format(dateLayout)

	expectSettingsGet(mock, "u1", 2000000, 1)
	expectIncomeGet(mock, "u1", cycleStr, 15000000)
	expectIncomeGet(mock, "u1", prevStr, 14000000)
	expectFixedCostsList(mock, "u1", 3000000, 2000000)
	expectSumExpenses(mock, "u1", cycleStr, endStr, 1000000)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	expected := SafeToSpend(15000000, 5000000, 2000000, 1000000, fx.daysRemaining)
	body := w.Body.String()
	assert.Contains(t, body, fmt.Sprintf(`"safeToSpend":%d`, expected))
	assert.Contains(t, body, fmt.Sprintf(`"daysRemaining":%d`, fx.daysRemaining))
	assert.Contains(t, body, `"income":15000000`)
	assert.Contains(t, body, `"previousIncome":14000000`)
	assert.Contains(t, body, `"budgets":[]`)
	assert.Contains(t, body, `"activeAlerts":[]`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCycleSummary_IncomeNotDeclared pins the I/O matrix's "income not
// declared" scenario: income is null in the response but safeToSpend is
// still computed, treating income as 0. Also doubles as the "no expenses
// yet this cycle" scenario (spent = 0, not an error).
func TestGetCycleSummary_IncomeNotDeclared(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	fx := newCycleSummaryFixture(1)
	cycleStr := fx.cycleStart.Format(dateLayout)
	prevStr := fx.prevStart.Format(dateLayout)
	endStr := fx.cycleEnd.Format(dateLayout)

	expectSettingsGet(mock, "u1", 2000000, 1)
	expectIncomeGetNotFound(mock, "u1", cycleStr)
	expectIncomeGet(mock, "u1", prevStr, 14000000)
	expectFixedCostsList(mock, "u1", 5000000)
	expectSumExpenses(mock, "u1", cycleStr, endStr, 0)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	expected := SafeToSpend(0, 5000000, 2000000, 0, fx.daysRemaining)
	body := w.Body.String()
	assert.Contains(t, body, fmt.Sprintf(`"safeToSpend":%d`, expected))
	assert.Contains(t, body, `"income":null`)
	assert.Contains(t, body, `"previousIncome":14000000`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCycleSummary_NoFixedCostsOrSavingsGoal pins the I/O matrix's "fixed
// costs / savings goal not set" scenario: no settings row (defaults to
// savings_goal 0, cycle_start_day 1) and an empty fixed_costs list both
// default to 0 in the formula.
func TestGetCycleSummary_NoFixedCostsOrSavingsGoal(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	fx := newCycleSummaryFixture(1)
	cycleStr := fx.cycleStart.Format(dateLayout)
	prevStr := fx.prevStart.Format(dateLayout)
	endStr := fx.cycleEnd.Format(dateLayout)

	expectSettingsGetError(mock, "u1", sql.ErrNoRows)
	expectIncomeGetNotFound(mock, "u1", cycleStr)
	expectIncomeGetNotFound(mock, "u1", prevStr)
	expectFixedCostsList(mock, "u1")
	expectSumExpenses(mock, "u1", cycleStr, endStr, 0)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := w.Body.String()
	assert.Contains(t, body, `"safeToSpend":0`)
	assert.Contains(t, body, `"income":null`)
	assert.Contains(t, body, `"previousIncome":null`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCycleSummary_Overspent_NegativeSafeToSpend pins AD-8: when spent
// exceeds the envelope, safeToSpend is a real negative number, never
// clamped to 0.
func TestGetCycleSummary_Overspent_NegativeSafeToSpend(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	fx := newCycleSummaryFixture(1)
	cycleStr := fx.cycleStart.Format(dateLayout)
	prevStr := fx.prevStart.Format(dateLayout)
	endStr := fx.cycleEnd.Format(dateLayout)

	expectSettingsGet(mock, "u1", 2000000, 1)
	expectIncomeGet(mock, "u1", cycleStr, 5000000)
	expectIncomeGet(mock, "u1", prevStr, 5000000)
	expectFixedCostsList(mock, "u1", 3000000)
	expectSumExpenses(mock, "u1", cycleStr, endStr, 4000000)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	expected := SafeToSpend(5000000, 3000000, 2000000, 4000000, fx.daysRemaining)
	require.Negative(t, expected, "test fixture must actually exercise the overspent case")
	body := w.Body.String()
	assert.Contains(t, body, fmt.Sprintf(`"safeToSpend":%d`, expected))
	assert.NotContains(t, body, `"safeToSpend":0`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCycleSummary_PreviousIncomeNotDeclared pins the I/O matrix's
// "previous cycle also undeclared" scenario: previousIncome is null (not
// 0), independent of the current cycle's income.
func TestGetCycleSummary_PreviousIncomeNotDeclared(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	fx := newCycleSummaryFixture(1)
	cycleStr := fx.cycleStart.Format(dateLayout)
	prevStr := fx.prevStart.Format(dateLayout)
	endStr := fx.cycleEnd.Format(dateLayout)

	expectSettingsGet(mock, "u1", 0, 1)
	expectIncomeGet(mock, "u1", cycleStr, 10000000)
	expectIncomeGetNotFound(mock, "u1", prevStr)
	expectFixedCostsList(mock, "u1")
	expectSumExpenses(mock, "u1", cycleStr, endStr, 0)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := w.Body.String()
	assert.Contains(t, body, `"income":10000000`)
	assert.Contains(t, body, `"previousIncome":null`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCycleSummary_SettingsGetError_Returns500 pins that a genuine
// settings-lookup DB error (not "no rows", which SettingsRepo.Get already
// absorbs into defaults) surfaces as a 500, not a silent default.
func TestGetCycleSummary_SettingsGetError_Returns500(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectSettingsGetError(mock, "u1", sql.ErrConnDone)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50080`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCycleSummary_IncomeGetError_Returns500 pins that a genuine income
// lookup error (distinct from sql.ErrNoRows) propagates as a 500 instead of
// being treated as "not declared".
func TestGetCycleSummary_IncomeGetError_Returns500(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	fx := newCycleSummaryFixture(1)
	cycleStr := fx.cycleStart.Format(dateLayout)

	expectSettingsGet(mock, "u1", 0, 1)
	expectIncomeGetError(mock, "u1", cycleStr, sql.ErrConnDone)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50081`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCycleSummary_FixedCostsListError_Returns500 pins that a fixed-costs
// lookup error surfaces as a 500.
func TestGetCycleSummary_FixedCostsListError_Returns500(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	fx := newCycleSummaryFixture(1)
	cycleStr := fx.cycleStart.Format(dateLayout)
	prevStr := fx.prevStart.Format(dateLayout)

	expectSettingsGet(mock, "u1", 0, 1)
	expectIncomeGetNotFound(mock, "u1", cycleStr)
	expectIncomeGetNotFound(mock, "u1", prevStr)
	expectFixedCostsListError(mock, "u1", sql.ErrConnDone)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50083`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCycleSummary_SumExpensesError_Returns500 pins that a
// SumExpensesInRange error surfaces as a 500.
func TestGetCycleSummary_SumExpensesError_Returns500(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	fx := newCycleSummaryFixture(1)
	cycleStr := fx.cycleStart.Format(dateLayout)
	prevStr := fx.prevStart.Format(dateLayout)
	endStr := fx.cycleEnd.Format(dateLayout)

	expectSettingsGet(mock, "u1", 0, 1)
	expectIncomeGetNotFound(mock, "u1", cycleStr)
	expectIncomeGetNotFound(mock, "u1", prevStr)
	expectFixedCostsList(mock, "u1")
	expectSumExpensesError(mock, "u1", cycleStr, endStr, sql.ErrConnDone)

	w, c := authedRequest("GET", "/api/cycle/summary", "", "u1", nil)
	h.GetCycleSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50084`)
	require.NoError(t, mock.ExpectationsWereMet())
}
