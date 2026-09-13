package api

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestRegisterUser_Success(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM users WHERE email = \?`).
		WithArgs("new@b.com").
		WillReturnError(sql.ErrNoRows)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO users \(id, name, email, password_hash\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "New User", "new@b.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	for _, cat := range defaultCategories {
		mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), cat.Name, cat.Type, cat.Color, cat.Icon).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	body := `{"name":"New User","email":"new@b.com","password":"secret1"}`
	w, c := plainRequest("POST", "/api/auth/register", body, map[string]string{"Content-Type": "application/json"})
	h.RegisterUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token"`)
	assert.Contains(t, w.Body.String(), `"new@b.com"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestRegisterUser_DuplicateEmail pins the I/O matrix's "Register + duplicate
// email" scenario: EmailExists short-circuits Create — no transaction is
// opened and no row is inserted a second time.
func TestRegisterUser_DuplicateEmail(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM users WHERE email = \?`).
		WithArgs("dup@b.com").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	body := `{"name":"Dup","email":"dup@b.com","password":"secret1"}`
	w, c := plainRequest("POST", "/api/auth/register", body, map[string]string{"Content-Type": "application/json"})
	h.RegisterUser(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40900`)
	// ExpectationsWereMet proves no INSERT (no 2nd row) followed the EmailExists
	// check — sqlmock would fail on any unexpected query.
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_Success(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT id, name, email, password_hash FROM users WHERE email = \?`).
		WithArgs("user@b.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password_hash"}).
			AddRow("u1", "User", "user@b.com", string(hash)))

	body := `{"email":"user@b.com","password":"correct-password"}`
	w, c := plainRequest("POST", "/api/auth/login", body, map[string]string{"Content-Type": "application/json"})
	h.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestLogin_WrongPassword pins the I/O matrix's "Login wrong password"
// scenario: a generic auth error, and no user data (name/email/hash) leaked
// in the response body.
func TestLogin_WrongPassword(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT id, name, email, password_hash FROM users WHERE email = \?`).
		WithArgs("user@b.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password_hash"}).
			AddRow("u1", "User", "user@b.com", string(hash)))

	body := `{"email":"user@b.com","password":"totally-wrong"}`
	w, c := plainRequest("POST", "/api/auth/login", body, map[string]string{"Content-Type": "application/json"})
	h.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"code":40102`)
	assert.NotContains(t, w.Body.String(), "user@b.com")
	assert.NotContains(t, w.Body.String(), string(hash))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMe(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		h, mock, closeDB := newTestHandler(t)
		defer closeDB()

		token, err := h.issueToken("u1")
		require.NoError(t, err)

		w, c := plainRequest("GET", "/api/auth/me", "", map[string]string{"Authorization": "Bearer " + token})
		h.AuthMiddleware()(c)
		require.False(t, c.IsAborted(), "middleware must accept a validly-signed, unexpired token")

		mock.ExpectQuery(`SELECT id, name, email FROM users WHERE id = \?`).
			WithArgs("u1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).AddRow("u1", "Alice", "alice@b.com"))

		h.Me(c)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"alice@b.com"`)
	})

	t.Run("invalid token", func(t *testing.T) {
		h, _, closeDB := newTestHandler(t)
		defer closeDB()

		w, c := plainRequest("GET", "/api/auth/me", "", map[string]string{"Authorization": "Bearer not-a-real-token"})
		h.AuthMiddleware()(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), `"code":40101`)
	})

	t.Run("missing token", func(t *testing.T) {
		h, _, closeDB := newTestHandler(t)
		defer closeDB()

		w, c := plainRequest("GET", "/api/auth/me", "", nil)
		h.AuthMiddleware()(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), `"code":40100`)
	})
}

// TestDemo_Seeding pins the withTx orchestration path in Demo: a demo user is
// created, seeded with the standard default categories, then populated with
// sample transactions and budgets — all inside a single transaction that
// commits once, atomically.
func TestDemo_Seeding(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectBegin()

	mock.ExpectExec(`INSERT INTO users \(id, name, email, password_hash\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "Demo", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	seededNames := make([]string, 0, len(defaultCategories))
	for _, cat := range defaultCategories {
		mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), cat.Name, cat.Type, cat.Color, cat.Icon).
			WillReturnResult(sqlmock.NewResult(1, 1))
		seededNames = append(seededNames, cat.Name)
	}

	byNameRows := sqlmock.NewRows([]string{"id", "name"})
	for _, name := range seededNames {
		byNameRows.AddRow("id-"+name, name)
	}
	mock.ExpectQuery(`SELECT id, name FROM categories WHERE user_id = \?`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(byNameRows)

	for i := 0; i < len(sampleTransactions); i++ {
		mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	for i := 0; i < len(sampleBudgets); i++ {
		mock.ExpectExec(`INSERT INTO budgets \(id, user_id, category_id, limit_amount, month\)\s+VALUES \(\?, \?, \?, \?, \?\)\s+ON DUPLICATE KEY UPDATE limit_amount = VALUES\(limit_amount\)`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(`SELECT id, category_id, limit_amount, month FROM budgets\s+WHERE user_id = \? AND category_id = \? AND month = \?`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "category_id", "limit_amount", "month"}).
				AddRow("bid", "cid", int64(1000000), "2026-09"))
	}

	mock.ExpectCommit()

	w, c := plainRequest("POST", "/api/auth/demo", "", nil)
	h.Demo(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestRegisterUser_SeedingFailureRollsBack: if SeedDefaults fails partway
// through (inside the same transaction as the user insert), the whole signup
// must roll back — no token/user data returned, and the transaction rolled
// back rather than committed.
func TestRegisterUser_SeedingFailureRollsBack(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectQuery(`SELECT 1 FROM users WHERE email = \?`).
		WithArgs("new3@b.com").
		WillReturnError(sql.ErrNoRows)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO users \(id, name, email, password_hash\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "New User", "new3@b.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// first category insert succeeds...
	mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// ...second one fails, aborting SeedDefaults mid-way.
	mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	body := `{"name":"New User","email":"new3@b.com","password":"secret1"}`
	w, c := plainRequest("POST", "/api/auth/register", body, map[string]string{"Content-Type": "application/json"})
	h.RegisterUser(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), `"token"`)
	assert.NotContains(t, w.Body.String(), "new3@b.com")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDemo_SeedingFailureRollsBack mirrors TestRegisterUser_SeedingFailureRollsBack
// for the Demo flow: a mid-transaction insert failure during SeedDefaults must
// roll back the whole demo-account creation, with no token/user leaked.
func TestDemo_SeedingFailureRollsBack(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO users \(id, name, email, password_hash\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "Demo", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// first two category inserts succeed...
	mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// ...third one fails, aborting before seedSampleData ever runs.
	mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	w, c := plainRequest("POST", "/api/auth/demo", "", nil)
	h.Demo(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), `"token"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDemo_SampleDataFailureRollsBack pins seedSampleData's own error
// propagation (added by Story 1.1's refactor): unlike the pre-refactor code,
// which discarded each sample insert's error and let the demo account
// succeed with partial data, a failure inside seedSampleData now aborts and
// rolls back the whole withTx — including the already-inserted user and
// categories. TestDemo_SeedingFailureRollsBack only exercises a failure
// during SeedDefaults, before seedSampleData ever runs; this test reaches
// seedSampleData itself by letting user + category creation succeed first.
func TestDemo_SampleDataFailureRollsBack(t *testing.T) {
	h, mock, closeDB := newTestHandler(t)
	defer closeDB()

	mock.ExpectBegin()

	mock.ExpectExec(`INSERT INTO users \(id, name, email, password_hash\) VALUES \(\?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), "Demo", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	seededNames := make([]string, 0, len(defaultCategories))
	for _, cat := range defaultCategories {
		mock.ExpectExec(`INSERT INTO categories \(id, user_id, name, type, color, icon\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), cat.Name, cat.Type, cat.Color, cat.Icon).
			WillReturnResult(sqlmock.NewResult(1, 1))
		seededNames = append(seededNames, cat.Name)
	}

	byNameRows := sqlmock.NewRows([]string{"id", "name"})
	for _, name := range seededNames {
		byNameRows.AddRow("id-"+name, name)
	}
	mock.ExpectQuery(`SELECT id, name FROM categories WHERE user_id = \?`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(byNameRows)

	// The very first sample transaction insert fails — everything already
	// staged in this transaction (user + all seeded categories) must roll
	// back rather than commit with a demo account left half-seeded.
	mock.ExpectExec(`INSERT INTO transactions \(id, user_id, type, amount, category_id, note, date\)\s+VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	w, c := plainRequest("POST", "/api/auth/demo", "", nil)
	h.Demo(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), `"token"`)
	require.NoError(t, mock.ExpectationsWereMet())
}
