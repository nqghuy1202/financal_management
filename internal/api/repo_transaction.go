package api

import (
	"context"
	"time"
)

const dateLayout = "2006-01-02"

// TransactionRepo is the data-access layer for transactions.
type TransactionRepo struct{ db dbtx }

func NewTransactionRepo(db dbtx) *TransactionRepo { return &TransactionRepo{db: db} }

func (r *TransactionRepo) List(ctx context.Context, userID string) ([]Transaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, type, amount, COALESCE(category_id, ''), note, date
		 FROM transactions WHERE user_id = ? ORDER BY date DESC, created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Transaction, 0)
	for rows.Next() {
		var t Transaction
		var d time.Time
		if err := rows.Scan(&t.ID, &t.Type, &t.Amount, &t.CategoryID, &t.Note, &d); err != nil {
			return nil, err
		}
		t.Date = d.Format(dateLayout)
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *TransactionRepo) Create(ctx context.Context, userID string, t Transaction) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO transactions (id, user_id, type, amount, category_id, note, date)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, userID, t.Type, t.Amount, t.CategoryID, t.Note, t.Date,
	)
	return err
}

// Update overwrites the transaction identified by (id, userID). The bool
// return reports whether a row actually matched, so the caller can turn a
// no-op update into a 404 instead of a false "success".
func (r *TransactionRepo) Update(ctx context.Context, userID, id string, t Transaction) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE transactions SET type = ?, amount = ?, category_id = ?, note = ?, date = ?
		 WHERE id = ? AND user_id = ?`,
		t.Type, t.Amount, t.CategoryID, t.Note, t.Date, id, userID,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *TransactionRepo) Delete(ctx context.Context, userID, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

// SumExpensesInRange returns the total amount of expense-type transactions
// with date in [from, to) for userID. Zero transactions yields 0, not an
// error (COALESCE handles the no-rows case at the SQL level).
func (r *TransactionRepo) SumExpensesInRange(ctx context.Context, userID string, from, to time.Time) (int64, error) {
	var sum int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM transactions
		 WHERE user_id = ? AND type = 'expense' AND date >= ? AND date < ?`,
		userID, from.Format(dateLayout), to.Format(dateLayout),
	).Scan(&sum)
	return sum, err
}
