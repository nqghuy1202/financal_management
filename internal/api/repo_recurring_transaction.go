package api

import (
	"context"
	"time"
)

// RecurringTransactionRepo is the data-access layer for recurring
// transaction templates. Like TransactionRepo, it takes a dbtx so the same
// code runs standalone or inside a *sql.Tx — ConfirmRecurringTransaction
// constructs one over the same tx as the TransactionRepo it inserts through,
// so the transactions INSERT and the next_due_date UPDATE commit or fail
// together (spec Design Notes: "Confirm as one atomic operation").
type RecurringTransactionRepo struct{ db dbtx }

func NewRecurringTransactionRepo(db dbtx) *RecurringTransactionRepo {
	return &RecurringTransactionRepo{db: db}
}

// List returns every recurring template for userID — active and paused
// alike, for the management list (paused ones are filtered out only via
// their Active flag, never omitted here). Ordered by next_due_date so the
// soonest-due template sorts first.
func (r *RecurringTransactionRepo) List(ctx context.Context, userID string) ([]RecurringTransaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, type, amount, COALESCE(category_id, ''), note, frequency, next_due_date, active
		 FROM recurring_transactions WHERE user_id = ? ORDER BY next_due_date, created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]RecurringTransaction, 0)
	for rows.Next() {
		var rt RecurringTransaction
		var d time.Time
		if err := rows.Scan(&rt.ID, &rt.Type, &rt.Amount, &rt.CategoryID, &rt.Note, &rt.Frequency, &d, &rt.Active); err != nil {
			return nil, err
		}
		rt.NextDueDate = d.Format(dateLayout)
		list = append(list, rt)
	}
	return list, rows.Err()
}

// Get returns the template identified by (id, userID). A zero-value result
// (ID == "") means no matching row — callers check that instead of handling
// sql.ErrNoRows themselves, mirroring CategoryRepo.Owns.
func (r *RecurringTransactionRepo) Get(ctx context.Context, userID, id string) (RecurringTransaction, error) {
	var rt RecurringTransaction
	var d time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT id, type, amount, COALESCE(category_id, ''), note, frequency, next_due_date, active
		 FROM recurring_transactions WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&rt.ID, &rt.Type, &rt.Amount, &rt.CategoryID, &rt.Note, &rt.Frequency, &d, &rt.Active)
	if err != nil {
		return RecurringTransaction{}, ignoreNoRows(err)
	}
	rt.NextDueDate = d.Format(dateLayout)
	return rt, nil
}

func (r *RecurringTransactionRepo) Create(ctx context.Context, userID string, rt RecurringTransaction) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO recurring_transactions (id, user_id, type, amount, category_id, note, frequency, next_due_date, active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rt.ID, userID, rt.Type, rt.Amount, rt.CategoryID, rt.Note, rt.Frequency, rt.NextDueDate, rt.Active,
	)
	return err
}

// Update overwrites a template's editable fields (type/amount/category/note/
// frequency/active — never next_due_date, which only ConfirmRecurringTransaction
// advances) for (id, userID). The bool return reports whether a row actually
// matched, mirroring TransactionRepo.Update, so the caller can turn a no-op
// update into a 404.
func (r *RecurringTransactionRepo) Update(ctx context.Context, userID, id string, rt RecurringTransaction) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE recurring_transactions SET type = ?, amount = ?, category_id = ?, note = ?, frequency = ?, active = ?
		 WHERE id = ? AND user_id = ?`,
		rt.Type, rt.Amount, rt.CategoryID, rt.Note, rt.Frequency, rt.Active, id, userID,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// Delete removes the template identified by (id, userID). The bool return
// reports whether a row actually matched, so the caller can turn a no-op
// delete into a 404 — already-confirmed transactions are untouched (they
// live in the separate `transactions` table with no FK back to this one).
func (r *RecurringTransactionRepo) Delete(ctx context.Context, userID, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM recurring_transactions WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// AdvanceNextDueDate persists a template's recomputed next_due_date (see
// AdvanceRecurrence) — the second write of ConfirmRecurringTransaction's
// atomic pair (transactions INSERT + this UPDATE), run inside the same
// *sql.Tx via h.withTx.
func (r *RecurringTransactionRepo) AdvanceNextDueDate(ctx context.Context, userID, id, nextDueDate string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE recurring_transactions SET next_due_date = ? WHERE id = ? AND user_id = ?`,
		nextDueDate, id, userID,
	)
	return err
}
