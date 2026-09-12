package api

import (
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFixedCosts_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "name", "amount"}).
		AddRow("f1", "Thuê nhà", int64(5000000))
	mock.ExpectQuery(`SELECT id, name, amount FROM fixed_costs WHERE user_id = \? ORDER BY created_at`).
		WithArgs("u1").
		WillReturnRows(rows)

	w, c := authedRequest("GET", "/api/fixed-costs", "", "u1", nil)
	h.ListFixedCosts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	assert.Contains(t, w.Body.String(), `"Thuê nhà"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFixedCost_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO fixed_costs \(id, user_id, name, amount\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "Thuê nhà", int64(5000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"name":"Thuê nhà","amount":5000000}`
	w, c := authedRequest("POST", "/api/fixed-costs", body, "u1", nil)
	h.CreateFixedCost(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFixedCost_ValidationFailure_EmptyName(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/fixed-costs", `{"name":"","amount":5000000}`, "u1", nil)
	h.CreateFixedCost(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40051`)
}

func TestCreateFixedCost_ValidationFailure_WhitespaceOnlyName(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/fixed-costs", `{"name":"   ","amount":5000000}`, "u1", nil)
	h.CreateFixedCost(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40051`)
}

func TestCreateFixedCost_ValidationFailure_NonPositiveAmount(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/fixed-costs", `{"name":"Thuê nhà","amount":0}`, "u1", nil)
	h.CreateFixedCost(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40051`)
}

func TestCreateFixedCost_MalformedJSON(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/fixed-costs", `not-json`, "u1", nil)
	h.CreateFixedCost(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40050`)
}

func TestUpdateFixedCost_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`UPDATE fixed_costs SET name = \?, amount = \? WHERE id = \? AND user_id = \?`).
		WithArgs("Thuê nhà mới", int64(6000000), "f1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	body := `{"name":"Thuê nhà mới","amount":6000000}`
	w, c := authedRequest("PUT", "/api/fixed-costs/f1", body, "u1", ginParams("id", "f1"))
	h.UpdateFixedCost(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateFixedCost_NotFound pins the I/O matrix's "edit someone else's or
// a missing fixed cost" scenario.
func TestUpdateFixedCost_NotFound(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`UPDATE fixed_costs SET name = \?, amount = \? WHERE id = \? AND user_id = \?`).
		WithArgs("X", int64(1000), "missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	body := `{"name":"X","amount":1000}`
	w, c := authedRequest("PUT", "/api/fixed-costs/missing", body, "u1", ginParams("id", "missing"))
	h.UpdateFixedCost(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40450`)
}

func TestUpdateFixedCost_ValidationFailure(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("PUT", "/api/fixed-costs/f1", `{"name":"","amount":1000}`, "u1", ginParams("id", "f1"))
	h.UpdateFixedCost(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40051`)
}

func TestDeleteFixedCost_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM fixed_costs WHERE id = \? AND user_id = \?`).
		WithArgs("f1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w, c := authedRequest("DELETE", "/api/fixed-costs/f1", "", "u1", ginParams("id", "f1"))
	h.DeleteFixedCost(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"deleted":true`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteFixedCost_NotFound pins the I/O matrix's "delete someone else's
// or a missing fixed cost" scenario.
func TestDeleteFixedCost_NotFound(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM fixed_costs WHERE id = \? AND user_id = \?`).
		WithArgs("missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	w, c := authedRequest("DELETE", "/api/fixed-costs/missing", "", "u1", ginParams("id", "missing"))
	h.DeleteFixedCost(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40450`)
}
