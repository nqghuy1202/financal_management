package api

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockFixedCostRepo(t *testing.T) (*FixedCostRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewFixedCostRepo(db), mock, func() { db.Close() }
}

func TestFixedCostRepo_List(t *testing.T) {
	repo, mock, closeDB := newMockFixedCostRepo(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "name", "amount"}).
		AddRow("f1", "Thuê nhà", int64(5000000)).
		AddRow("f2", "Internet", int64(300000))
	mock.ExpectQuery(`SELECT id, name, amount FROM fixed_costs WHERE user_id = \? ORDER BY created_at`).
		WithArgs("u1").
		WillReturnRows(rows)

	list, err := repo.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []FixedCost{
		{ID: "f1", Name: "Thuê nhà", Amount: 5000000},
		{ID: "f2", Name: "Internet", Amount: 300000},
	}, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFixedCostRepo_List_Empty(t *testing.T) {
	repo, mock, closeDB := newMockFixedCostRepo(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "name", "amount"})
	mock.ExpectQuery(`SELECT id, name, amount FROM fixed_costs WHERE user_id = \? ORDER BY created_at`).
		WithArgs("u1").
		WillReturnRows(rows)

	list, err := repo.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Empty(t, list)
}

func TestFixedCostRepo_Create(t *testing.T) {
	repo, mock, closeDB := newMockFixedCostRepo(t)
	defer closeDB()

	fc := FixedCost{ID: "f1", Name: "Thuê nhà", Amount: 5000000}
	mock.ExpectExec(`INSERT INTO fixed_costs \(id, user_id, name, amount\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs(fc.ID, "u1", fc.Name, fc.Amount).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), "u1", fc)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFixedCostRepo_Update_Found(t *testing.T) {
	repo, mock, closeDB := newMockFixedCostRepo(t)
	defer closeDB()

	mock.ExpectExec(`UPDATE fixed_costs SET name = \?, amount = \? WHERE id = \? AND user_id = \?`).
		WithArgs("Thuê nhà mới", int64(6000000), "f1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	found, err := repo.Update(context.Background(), "u1", "f1", "Thuê nhà mới", 6000000)
	require.NoError(t, err)
	assert.True(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFixedCostRepo_Update_NotFound pins the I/O matrix's "edit someone
// else's or a missing fixed cost" scenario: zero rows affected must report
// found=false so the handler can turn it into a 404.
func TestFixedCostRepo_Update_NotFound(t *testing.T) {
	repo, mock, closeDB := newMockFixedCostRepo(t)
	defer closeDB()

	mock.ExpectExec(`UPDATE fixed_costs SET name = \?, amount = \? WHERE id = \? AND user_id = \?`).
		WithArgs("X", int64(1000), "missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	found, err := repo.Update(context.Background(), "u1", "missing", "X", 1000)
	require.NoError(t, err)
	assert.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFixedCostRepo_Delete_Found(t *testing.T) {
	repo, mock, closeDB := newMockFixedCostRepo(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM fixed_costs WHERE id = \? AND user_id = \?`).
		WithArgs("f1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	found, err := repo.Delete(context.Background(), "u1", "f1")
	require.NoError(t, err)
	assert.True(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFixedCostRepo_Delete_NotFound(t *testing.T) {
	repo, mock, closeDB := newMockFixedCostRepo(t)
	defer closeDB()

	mock.ExpectExec(`DELETE FROM fixed_costs WHERE id = \? AND user_id = \?`).
		WithArgs("missing", "u1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	found, err := repo.Delete(context.Background(), "u1", "missing")
	require.NoError(t, err)
	assert.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}
