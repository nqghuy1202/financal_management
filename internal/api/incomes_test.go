package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertIncome_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, _ := CycleWindow(1, time.Now())
	cycleStr := cycleStart.Format(dateLayout)

	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", cycleStr, int64(15000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", cycleStr).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).
			AddRow("i1", cycleStart, int64(15000000)))

	body := `{"amount":15000000}`
	w, c := authedRequest("POST", "/api/incomes", body, "u1", nil)
	h.UpsertIncome(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	assert.Contains(t, w.Body.String(), `"amount":15000000`)
	assert.Contains(t, w.Body.String(), fmt.Sprintf(`"cycleStartDate":"%s"`, cycleStr))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpsertIncome_Idempotency pins the I/O matrix's "re-declare same cycle"
// scenario at the handler level: a 2nd POST with a different amount must
// update the same row (no 2nd row), reflecting the latest amount.
func TestUpsertIncome_Idempotency(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, _ := CycleWindow(1, time.Now())
	cycleStr := cycleStart.Format(dateLayout)

	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", cycleStr, int64(15000000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", cycleStr).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).
			AddRow("i1", cycleStart, int64(15000000)))

	w1, c1 := authedRequest("POST", "/api/incomes", `{"amount":15000000}`, "u1", nil)
	h.UpsertIncome(c1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), fmt.Sprintf(`"cycleStartDate":"%s"`, cycleStr))

	mock.ExpectExec(`INSERT INTO incomes \(id, user_id, cycle_start_date, amount\)\s+VALUES \(\?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE amount = VALUES\(amount\)`).
		WithArgs(sqlmock.AnyArg(), "u1", cycleStr, int64(20000000)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", cycleStr).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cycle_start_date", "amount"}).
			AddRow("i1", cycleStart, int64(20000000)))

	w2, c2 := authedRequest("POST", "/api/incomes", `{"amount":20000000}`, "u1", nil)
	h.UpsertIncome(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), `"id":"i1"`)
	assert.Contains(t, w2.Body.String(), `"amount":20000000`)
	assert.Contains(t, w2.Body.String(), fmt.Sprintf(`"cycleStartDate":"%s"`, cycleStr))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertIncome_InvalidAmount_Zero(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/incomes", `{"amount":0}`, "u1", nil)
	h.UpsertIncome(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40041`)
}

func TestUpsertIncome_InvalidAmount_Negative(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/incomes", `{"amount":-100}`, "u1", nil)
	h.UpsertIncome(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40041`)
}

func TestUpsertIncome_InvalidAmount_Missing(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/incomes", `{}`, "u1", nil)
	h.UpsertIncome(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40041`)
}

func TestUpsertIncome_MalformedJSON(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/incomes", `not-json`, "u1", nil)
	h.UpsertIncome(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40040`)
}

// TestNoCarryOver pins the I/O matrix's "no carry-over" scenario directly
// against IncomeRepo.Get: income declared for one cycle must not be visible
// under a later cycle's start date.
func TestNoCarryOver(t *testing.T) {
	repo, mock, closeDB := newMockIncomeRepo(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT id, cycle_start_date, amount FROM incomes\s+WHERE user_id = \? AND cycle_start_date = \?`).
		WithArgs("u1", "2026-10-01").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.Get(context.Background(), "u1", mustDate("2026-10-01"))
	assert.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}
