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

// expectCycleSettingsWrite sets up the mock expectations for one successful
// UpdateCycleSettings save: settings upsert first, then the income upsert
// keyed to the cycle computed from cycleStartDay.
func expectCycleSettingsWrite(mock sqlmock.Sqlmock, userID string, income, savingsGoal int64, cycleStartDay int) string {
	cycleStart, _ := CycleWindow(cycleStartDay, time.Now())
	cycleStr := cycleStart.Format(dateLayout)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal, cycle_start_day\)\s+VALUES \(\?, \?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\), cycle_start_day = VALUES\(cycle_start_day\)`).
		WithArgs(userID, savingsGoal, cycleStartDay).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(savingsGoal, cycleStartDay))
	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), userID, cycleStr, income).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs(userID, cycleStr).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).
			AddRow("i1", cycleStart, income))
	mock.ExpectCommit()

	return cycleStr
}

func TestUpdateCycleSettings_HappyPath_Day1(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStr := expectCycleSettingsWrite(mock, "u1", 15000000, 2000000, 1)

	body := `{"income":15000000,"savingsGoal":2000000,"cycleStartDay":1}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	assert.Contains(t, w.Body.String(), `"amount":15000000`)
	assert.Contains(t, w.Body.String(), `"savingsGoal":2000000`)
	assert.Contains(t, w.Body.String(), `"cycleStartDay":1`)
	assert.Contains(t, w.Body.String(), cycleStr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateCycleSettings_ChangeCycleStartDayMidSave pins the spec's core
// scenario: cycleStartDay changes from its previous value (1) to 15 in this
// same request, and the income written in this request must be keyed by
// CycleWindow(15, now) — the NEW day — not the old one.
func TestUpdateCycleSettings_ChangeCycleStartDayMidSave(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	oldCycleStart, _ := CycleWindow(1, time.Now())
	newCycleStr := expectCycleSettingsWrite(mock, "u1", 20000000, 5000000, 15)

	body := `{"income":20000000,"savingsGoal":5000000,"cycleStartDay":15}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), newCycleStr)
	assert.NotContains(t, w.Body.String(), oldCycleStart.Format(dateLayout))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateCycleSettings_InvalidIncome_Zero(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	body := `{"income":0,"savingsGoal":0,"cycleStartDay":1}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40071`)
}

func TestUpdateCycleSettings_InvalidIncome_Negative(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	body := `{"income":-1,"savingsGoal":0,"cycleStartDay":1}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40071`)
}

func TestUpdateCycleSettings_InvalidSavingsGoal_Negative(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	body := `{"income":15000000,"savingsGoal":-1,"cycleStartDay":1}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40072`)
}

// TestUpdateCycleSettings_MissingSavingsGoal pins the bug this endpoint must
// not repeat: omitting savingsGoal entirely must not silently default to 0
// and overwrite an existing goal. No mock expectations are registered, so an
// unexpected query would surface as a driver error rather than passing quietly.
func TestUpdateCycleSettings_MissingSavingsGoal(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	body := `{"income":15000000,"cycleStartDay":1}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40072`)
}

// TestUpdateCycleSettings_ZeroIsValid mirrors the deleted
// TestUpdateSavingsGoal_ZeroIsValid: an explicit 0 is a legitimate value
// (distinct from "omitted") and must be persisted and returned as-is.
func TestUpdateCycleSettings_ZeroIsValid(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectCycleSettingsWrite(mock, "u1", 15000000, 0, 1)

	body := `{"income":15000000,"savingsGoal":0,"cycleStartDay":1}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"savingsGoal":0`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateCycleSettings_HappyPath_Day31 pins the valid upper boundary of
// cycleStartDay.
func TestUpdateCycleSettings_HappyPath_Day31(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectCycleSettingsWrite(mock, "u1", 15000000, 2000000, 31)

	body := `{"income":15000000,"savingsGoal":2000000,"cycleStartDay":31}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"cycleStartDay":31`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateCycleSettings_InvalidCycleStartDay_TooLow(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	body := `{"income":15000000,"savingsGoal":0,"cycleStartDay":0}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40073`)
}

func TestUpdateCycleSettings_InvalidCycleStartDay_TooHigh(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	body := `{"income":15000000,"savingsGoal":0,"cycleStartDay":32}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40073`)
}

func TestUpdateCycleSettings_MalformedJSON(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("PUT", "/api/cycle-settings", `not-json`, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40070`)
}

// TestUpdateCycleSettings_PartialFailureRollsBack pins the I/O matrix's
// rollback scenario: the settings write succeeds but the income write fails
// inside the same transaction — neither must be left committed.
func TestUpdateCycleSettings_PartialFailureRollsBack(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, _ := CycleWindow(1, time.Now())
	cycleStr := cycleStart.Format(dateLayout)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal, cycle_start_day\)\s+VALUES \(\?, \?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\), cycle_start_day = VALUES\(cycle_start_day\)`).
		WithArgs("u1", int64(2000000), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(2000000), 1))
	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", cycleStr, int64(15000000)).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	body := `{"income":15000000,"savingsGoal":2000000,"cycleStartDay":1}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50070`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateCycleSettings_SettingsUpsertFailsRollsBack pins the other half of
// the rollback matrix: the settings upsert itself fails, before the income
// write is ever attempted. No income INSERT expectation is registered, so if
// the code regressed to attempt it anyway, ExpectationsWereMet would still
// pass (mock allows unexpected calls to fail) but the query would error
// against an unmatched expectation, and ExpectRollback below pins that the
// transaction is rolled back rather than committed.
func TestUpdateCycleSettings_SettingsUpsertFailsRollsBack(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal, cycle_start_day\)\s+VALUES \(\?, \?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\), cycle_start_day = VALUES\(cycle_start_day\)`).
		WithArgs("u1", int64(2000000), 1).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	body := `{"income":15000000,"savingsGoal":2000000,"cycleStartDay":1}`
	w, c := authedRequest("PUT", "/api/cycle-settings", body, "u1", nil)
	h.UpdateCycleSettings(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50070`)
	require.NoError(t, mock.ExpectationsWereMet())
}
