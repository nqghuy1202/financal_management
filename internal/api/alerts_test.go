package api

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dismissParams builds the two-segment gin.Params DismissAlert reads via
// c.Param("categoryId") / c.Param("threshold").
func dismissParams(categoryID, threshold string) gin.Params {
	return gin.Params{
		{Key: "categoryId", Value: categoryID},
		{Key: "threshold", Value: threshold},
	}
}

// TestDismissAlert_HappyPath pins the ordinary path: an owned category and a
// valid threshold dismiss cleanly.
func TestDismissAlert_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, _, _, _ := currentCycleFixture()
	cycleStr := cycleStart.Format(dateLayout)

	expectOwnsCategory(mock, "c1", "u1")
	expectSettingsGet(mock, "u1", 0, 1)
	mock.ExpectExec(`UPDATE budget_alert_state SET dismissed_at = CURRENT_TIMESTAMP\s+WHERE user_id = \? AND category_id = \? AND cycle_start_date = \? AND threshold = \? AND dismissed_at IS NULL`).
		WithArgs("u1", "c1", cycleStr, 100).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w, c := authedRequest("POST", "/api/alerts/c1/100/dismiss", "", "u1", dismissParams("c1", "100"))
	h.DismissAlert(c)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"dismissed":true`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDismissAlert_IdempotentOnRepeat pins that dismissing an already-
// dismissed alert (0 rows affected) still returns 200, never an error.
func TestDismissAlert_IdempotentOnRepeat(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, _, _, _ := currentCycleFixture()
	cycleStr := cycleStart.Format(dateLayout)

	expectOwnsCategory(mock, "c1", "u1")
	expectSettingsGet(mock, "u1", 0, 1)
	mock.ExpectExec(`UPDATE budget_alert_state SET dismissed_at = CURRENT_TIMESTAMP\s+WHERE user_id = \? AND category_id = \? AND cycle_start_date = \? AND threshold = \? AND dismissed_at IS NULL`).
		WithArgs("u1", "c1", cycleStr, 70).
		WillReturnResult(sqlmock.NewResult(0, 0))

	w, c := authedRequest("POST", "/api/alerts/c1/70/dismiss", "", "u1", dismissParams("c1", "70"))
	h.DismissAlert(c)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"dismissed":true`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDismissAlert_InvalidThreshold_NonNumeric pins that a non-numeric
// threshold segment is rejected before any DB access.
func TestDismissAlert_InvalidThreshold_NonNumeric(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/alerts/c1/abc/dismiss", "", "u1", dismissParams("c1", "abc"))
	h.DismissAlert(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40090`)
}

// TestDismissAlert_InvalidThreshold_NotInAllowedSet pins that a numeric but
// out-of-band threshold (e.g. 50) is rejected before any DB access.
func TestDismissAlert_InvalidThreshold_NotInAllowedSet(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	w, c := authedRequest("POST", "/api/alerts/c1/50/dismiss", "", "u1", dismissParams("c1", "50"))
	h.DismissAlert(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40090`)
}

// TestDismissAlert_NotOwnedCategory pins that a category which doesn't
// resolve under the caller's user id is rejected with a 400, not a 404/500.
func TestDismissAlert_NotOwnedCategory(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("categoryOfUserB", "userA").
		WillReturnError(sql.ErrNoRows)

	w, c := authedRequest("POST", "/api/alerts/categoryOfUserB/100/dismiss", "", "userA", dismissParams("categoryOfUserB", "100"))
	h.DismissAlert(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40091`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDismissAlert_OwnershipCheckDBError pins that a genuine DB error while
// checking ownership (as opposed to not-found) surfaces as a 500.
func TestDismissAlert_OwnershipCheckDBError(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("c1", "u1").
		WillReturnError(sql.ErrConnDone)

	w, c := authedRequest("POST", "/api/alerts/c1/100/dismiss", "", "u1", dismissParams("c1", "100"))
	h.DismissAlert(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50090`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDismissAlert_SettingsGetError pins that a genuine settings-lookup
// error surfaces as a 500.
func TestDismissAlert_SettingsGetError(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	expectOwnsCategory(mock, "c1", "u1")
	expectSettingsGetError(mock, "u1", sql.ErrConnDone)

	w, c := authedRequest("POST", "/api/alerts/c1/100/dismiss", "", "u1", dismissParams("c1", "100"))
	h.DismissAlert(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50090`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDismissAlert_RepoError pins that a genuine Dismiss exec error
// surfaces as a 500.
func TestDismissAlert_RepoError(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	cycleStart, _, _, _ := currentCycleFixture()
	cycleStr := cycleStart.Format(dateLayout)

	expectOwnsCategory(mock, "c1", "u1")
	expectSettingsGet(mock, "u1", 0, 1)
	mock.ExpectExec(`UPDATE budget_alert_state SET dismissed_at = CURRENT_TIMESTAMP\s+WHERE user_id = \? AND category_id = \? AND cycle_start_date = \? AND threshold = \? AND dismissed_at IS NULL`).
		WithArgs("u1", "c1", cycleStr, 100).
		WillReturnError(sql.ErrConnDone)

	w, c := authedRequest("POST", "/api/alerts/c1/100/dismiss", "", "u1", dismissParams("c1", "100"))
	h.DismissAlert(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"code":50090`)
	require.NoError(t, mock.ExpectationsWereMet())
}
