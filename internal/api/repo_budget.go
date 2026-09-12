package api

import (
	"context"

	"github.com/google/uuid"
)

// BudgetRepo is the data-access layer for budgets.
type BudgetRepo struct{ db dbtx }

func NewBudgetRepo(db dbtx) *BudgetRepo { return &BudgetRepo{db: db} }

func (r *BudgetRepo) List(ctx context.Context, userID string) ([]Budget, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Budget, 0)
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.Limit, &b.Month); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

// Upsert sets the limit for (userID, b.CategoryID, b.Month): inserts a new
// row, or updates the existing one (unique per user+category+month), then
// returns the canonical row.
func (r *BudgetRepo) Upsert(ctx context.Context, userID string, b Budget) (Budget, error) {
	id := uuid.NewString()
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO budgets (id, user_id, category_id, limit_amount, month)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE limit_amount = VALUES(limit_amount)`,
		id, userID, b.CategoryID, b.Limit, b.Month,
	); err != nil {
		return Budget{}, err
	}

	var out Budget
	err := r.db.QueryRowContext(ctx,
		`SELECT id, category_id, limit_amount, month FROM budgets
		 WHERE user_id = ? AND category_id = ? AND month = ?`,
		userID, b.CategoryID, b.Month,
	).Scan(&out.ID, &out.CategoryID, &out.Limit, &out.Month)
	if err != nil {
		// Fallback to what we just inserted (id may differ from an existing row).
		return Budget{ID: id, CategoryID: b.CategoryID, Limit: b.Limit, Month: b.Month}, nil
	}
	return out, nil
}

// GetByCategoryMonth returns the budget row for (userID, categoryID, month),
// or propagates sql.ErrNoRows as-is when no budget has been set for that
// category/month — callers use ignoreNoRows (repo_helpers.go) to turn that
// into a plain "no budget" zero value.
func (r *BudgetRepo) GetByCategoryMonth(ctx context.Context, userID, categoryID, month string) (Budget, error) {
	var b Budget
	err := r.db.QueryRowContext(ctx,
		`SELECT id, category_id, limit_amount, month FROM budgets WHERE user_id = ? AND category_id = ? AND month = ?`,
		userID, categoryID, month,
	).Scan(&b.ID, &b.CategoryID, &b.Limit, &b.Month)
	return b, err
}

func (r *BudgetRepo) Delete(ctx context.Context, userID, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM budgets WHERE id = ? AND user_id = ?`, id, userID)
	return err
}
