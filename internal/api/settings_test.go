package api

import (
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSavingsGoal_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal\)\s+VALUES \(\?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\)`).
		WithArgs("u1", int64(2000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(2000000), 1))

	body := `{"savingsGoal":2000000}`
	w, c := authedRequest("PUT", "/api/settings/savings-goal", body, "u1", nil)
	h.UpdateSavingsGoal(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	assert.Contains(t, w.Body.String(), `"savingsGoal":2000000`)
	assert.Contains(t, w.Body.String(), `"cycleStartDay":1`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSavingsGoal_InvalidGoal_Negative(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("PUT", "/api/settings/savings-goal", `{"savingsGoal":-1}`, "u1", nil)
	h.UpdateSavingsGoal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40061`)
}

func TestUpdateSavingsGoal_ZeroIsValid(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO user_settings \(user_id, savings_goal\)\s+VALUES \(\?, \?\)\s+ON DUPLICATE KEY UPDATE savings_goal = VALUES\(savings_goal\)`).
		WithArgs("u1", int64(0)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = \?`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"savings_goal", "cycle_start_day"}).AddRow(int64(0), 1))

	w, c := authedRequest("PUT", "/api/settings/savings-goal", `{"savingsGoal":0}`, "u1", nil)
	h.UpdateSavingsGoal(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSavingsGoal_MissingField(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("PUT", "/api/settings/savings-goal", `{}`, "u1", nil)
	h.UpdateSavingsGoal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40060`)
}

func TestUpdateSavingsGoal_MalformedJSON(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("PUT", "/api/settings/savings-goal", `not-json`, "u1", nil)
	h.UpdateSavingsGoal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40060`)
}
