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

func newMockUserRepo(t *testing.T) (*UserRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewUserRepo(db), mock, func() { db.Close() }
}

func TestUserRepo_EmailExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		repo, mock, closeDB := newMockUserRepo(t)
		defer closeDB()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM users WHERE email = ?`)).
			WithArgs("a@b.com").
			WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

		exists, err := repo.EmailExists(context.Background(), "a@b.com")
		require.NoError(t, err)
		assert.True(t, exists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock, closeDB := newMockUserRepo(t)
		defer closeDB()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM users WHERE email = ?`)).
			WithArgs("nobody@b.com").
			WillReturnError(sql.ErrNoRows)

		exists, err := repo.EmailExists(context.Background(), "nobody@b.com")
		require.NoError(t, err)
		assert.False(t, exists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error propagates", func(t *testing.T) {
		repo, mock, closeDB := newMockUserRepo(t)
		defer closeDB()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM users WHERE email = ?`)).
			WithArgs("boom@b.com").
			WillReturnError(sql.ErrConnDone)

		_, err := repo.EmailExists(context.Background(), "boom@b.com")
		assert.ErrorIs(t, err, sql.ErrConnDone)
	})
}

func TestUserRepo_Create(t *testing.T) {
	repo, mock, closeDB := newMockUserRepo(t)
	defer closeDB()

	u := User{ID: "u1", Name: "Alice", Email: "alice@b.com"}
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)`)).
		WithArgs(u.ID, u.Name, u.Email, "hashed").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), u, "hashed")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_FindByEmail(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo, mock, closeDB := newMockUserRepo(t)
		defer closeDB()

		rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash"}).
			AddRow("u1", "Alice", "alice@b.com", "hash123")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, email, password_hash FROM users WHERE email = ?`)).
			WithArgs("alice@b.com").
			WillReturnRows(rows)

		u, hash, err := repo.FindByEmail(context.Background(), "alice@b.com")
		require.NoError(t, err)
		assert.Equal(t, User{ID: "u1", Name: "Alice", Email: "alice@b.com"}, u)
		assert.Equal(t, "hash123", hash)
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock, closeDB := newMockUserRepo(t)
		defer closeDB()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, email, password_hash FROM users WHERE email = ?`)).
			WithArgs("nobody@b.com").
			WillReturnError(sql.ErrNoRows)

		_, _, err := repo.FindByEmail(context.Background(), "nobody@b.com")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestUserRepo_FindByID(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo, mock, closeDB := newMockUserRepo(t)
		defer closeDB()

		rows := sqlmock.NewRows([]string{"id", "name", "email"}).AddRow("u1", "Alice", "alice@b.com")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = ?`)).
			WithArgs("u1").
			WillReturnRows(rows)

		u, err := repo.FindByID(context.Background(), "u1")
		require.NoError(t, err)
		assert.Equal(t, User{ID: "u1", Name: "Alice", Email: "alice@b.com"}, u)
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock, closeDB := newMockUserRepo(t)
		defer closeDB()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = ?`)).
			WithArgs("missing").
			WillReturnError(sql.ErrNoRows)

		_, err := repo.FindByID(context.Background(), "missing")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}
