package api

import (
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCategories_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "name", "type", "color", "icon"}).
		AddRow("c1", "Lương", "income", "#10b981", "Wallet")
	mock.ExpectQuery(`SELECT id, name, type, color, icon FROM categories WHERE user_id = \? ORDER BY created_at`).
		WithArgs("u1").
		WillReturnRows(rows)

	w, c := authedRequest("GET", "/api/categories", "", "u1", nil)
	h.ListCategories(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	assert.Contains(t, w.Body.String(), `"Lương"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateCategory_ValidationFailure(t *testing.T) {
	h, _, closeDB := newTestHandler(t)
	defer closeDB()

	// invalid "type" (must be income|expense) — handler must reject before
	// touching the DB.
	w, c := authedRequest("POST", "/api/categories", `{"name":"Test","type":"bogus"}`, "u1", nil)
	h.CreateCategory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40011`)
}

func TestCreateCategory_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "u1", "Test", "expense", "#64748b", "Tag").
		WillReturnResult(sqlmock.NewResult(1, 1))

	w, c := authedRequest("POST", "/api/categories", `{"name":"Test","type":"expense"}`, "u1", nil)
	h.CreateCategory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":20000`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteCategory_HappyPath(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM categories WHERE id = \? AND user_id = \?`).
		WithArgs("c1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w, c := authedRequest("DELETE", "/api/categories/c1", "", "u1", ginParams("id", "c1"))
	h.DeleteCategory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"deleted":true`)
	require.NoError(t, mock.ExpectationsWereMet())
}
