package api

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockCategoryRepo(t *testing.T) (*CategoryRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewCategoryRepo(db), mock, func() { db.Close() }
}

func TestCategoryRepo_List(t *testing.T) {
	repo, mock, closeDB := newMockCategoryRepo(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "name", "type", "color", "icon"}).
		AddRow("c1", "Lương", "income", "#10b981", "Wallet").
		AddRow("c2", "Ăn uống", "expense", "#f97316", "Utensils")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, type, color, icon FROM categories WHERE user_id = ? ORDER BY created_at`)).
		WithArgs("u1").
		WillReturnRows(rows)

	list, err := repo.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []Category{
		{ID: "c1", Name: "Lương", Type: "income", Color: "#10b981", Icon: "Wallet"},
		{ID: "c2", Name: "Ăn uống", Type: "expense", Color: "#f97316", Icon: "Utensils"},
	}, list)
}

func TestCategoryRepo_List_Empty(t *testing.T) {
	repo, mock, closeDB := newMockCategoryRepo(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "name", "type", "color", "icon"})
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, type, color, icon FROM categories WHERE user_id = ? ORDER BY created_at`)).
		WithArgs("u1").
		WillReturnRows(rows)

	list, err := repo.List(context.Background(), "u1")
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Empty(t, list)
}

func TestCategoryRepo_Create(t *testing.T) {
	repo, mock, closeDB := newMockCategoryRepo(t)
	defer closeDB()

	cat := Category{ID: "c1", Name: "Ăn uống", Type: "expense", Color: "#f97316", Icon: "Utensils"}
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO categories (id, user_id, name, type, color, icon) VALUES (?, ?, ?, ?, ?, ?)`)).
		WithArgs(cat.ID, "u1", cat.Name, cat.Type, cat.Color, cat.Icon).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), "u1", cat)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepo_Delete(t *testing.T) {
	repo, mock, closeDB := newMockCategoryRepo(t)
	defer closeDB()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM categories WHERE id = ? AND user_id = ?`)).
		WithArgs("c1", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), "u1", "c1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepo_Owns(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		repo, mock, closeDB := newMockCategoryRepo(t)
		defer closeDB()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM categories WHERE id = ? AND user_id = ?`)).
			WithArgs("c1", "u1").
			WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

		owns, err := repo.Owns(context.Background(), "u1", "c1")
		require.NoError(t, err)
		assert.True(t, owns)
	})

	t.Run("false when owned by another user", func(t *testing.T) {
		repo, mock, closeDB := newMockCategoryRepo(t)
		defer closeDB()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM categories WHERE id = ? AND user_id = ?`)).
			WithArgs("c1", "userB").
			WillReturnError(sql.ErrNoRows)

		owns, err := repo.Owns(context.Background(), "userB", "c1")
		require.NoError(t, err)
		assert.False(t, owns)
	})

	t.Run("db error propagates", func(t *testing.T) {
		repo, mock, closeDB := newMockCategoryRepo(t)
		defer closeDB()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM categories WHERE id = ? AND user_id = ?`)).
			WithArgs("c1", "u1").
			WillReturnError(sql.ErrConnDone)

		_, err := repo.Owns(context.Background(), "u1", "c1")
		assert.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestCategoryRepo_ByName(t *testing.T) {
	repo, mock, closeDB := newMockCategoryRepo(t)
	defer closeDB()

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow("c1", "Lương").
		AddRow("c2", "Ăn uống")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name FROM categories WHERE user_id = ?`)).
		WithArgs("u1").
		WillReturnRows(rows)

	byName, err := repo.ByName(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Lương": "c1", "Ăn uống": "c2"}, byName)
}

func TestCategoryRepo_SeedDefaults(t *testing.T) {
	repo, mock, closeDB := newMockCategoryRepo(t)
	defer closeDB()

	for range defaultCategories {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO categories (id, user_id, name, type, color, icon) VALUES (?, ?, ?, ?, ?, ?)`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	err := repo.SeedDefaults(context.Background(), "u1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepo_SeedDefaults_StopsOnFirstError(t *testing.T) {
	repo, mock, closeDB := newMockCategoryRepo(t)
	defer closeDB()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO categories (id, user_id, name, type, color, icon) VALUES (?, ?, ?, ?, ?, ?)`)).
		WillReturnError(sql.ErrConnDone)

	err := repo.SeedDefaults(context.Background(), "u1")
	assert.ErrorIs(t, err, sql.ErrConnDone)
}
