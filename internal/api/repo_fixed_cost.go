package api

import "context"

// FixedCostRepo is the data-access layer for fixed (recurring) costs. Unlike
// income/budgets, fixed costs are a live list — not snapshotted per cycle.
type FixedCostRepo struct{ db dbtx }

func NewFixedCostRepo(db dbtx) *FixedCostRepo { return &FixedCostRepo{db: db} }

func (r *FixedCostRepo) List(ctx context.Context, userID string) ([]FixedCost, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, amount FROM fixed_costs WHERE user_id = ? ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]FixedCost, 0)
	for rows.Next() {
		var fc FixedCost
		if err := rows.Scan(&fc.ID, &fc.Name, &fc.Amount); err != nil {
			return nil, err
		}
		list = append(list, fc)
	}
	return list, rows.Err()
}

func (r *FixedCostRepo) Create(ctx context.Context, userID string, fc FixedCost) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fixed_costs (id, user_id, name, amount) VALUES (?, ?, ?, ?)`,
		fc.ID, userID, fc.Name, fc.Amount,
	)
	return err
}

// Update overwrites the fixed cost identified by (id, userID). The bool
// return reports whether a row actually matched, mirroring
// TransactionRepo.Update, so the caller can turn a no-op update into a 404.
func (r *FixedCostRepo) Update(ctx context.Context, userID, id string, name string, amount int64) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE fixed_costs SET name = ?, amount = ? WHERE id = ? AND user_id = ?`,
		name, amount, id, userID,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// Delete removes the fixed cost identified by (id, userID). The bool return
// reports whether a row actually matched, mirroring Update, so the caller
// can turn a no-op delete into a 404 instead of a false "success".
func (r *FixedCostRepo) Delete(ctx context.Context, userID, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM fixed_costs WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
