package api

import "context"

// UserRepo is the data-access layer for users. It depends on dbtx rather
// than *sql.DB, so the same code works standalone (NewUserRepo(h.db)) or
// inside a transaction (NewUserRepo(tx) from within withTx).
type UserRepo struct{ db dbtx }

func NewUserRepo(db dbtx) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var one int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE email = ?`, email).Scan(&one)
	if err != nil {
		return false, ignoreNoRows(err)
	}
	return one == 1, nil
}

func (r *UserRepo) Create(ctx context.Context, u User, passwordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)`,
		u.ID, u.Name, u.Email, passwordHash,
	)
	return err
}

// FindByEmail returns the user and their password hash (for login checks).
// The hash is returned separately since it never belongs on the User model.
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, email, password_hash FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Name, &u.Email, &hash)
	return u, hash, err
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, email FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Name, &u.Email)
	return u, err
}
