package api

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// IncomeRepo is the data-access layer for declared income.
type IncomeRepo struct{ db dbtx }

func NewIncomeRepo(db dbtx) *IncomeRepo { return &IncomeRepo{db: db} }

// Upsert declares amount as the income for (userID, cycleStartDate): inserts
// a new row, or updates the existing one (unique per user+cycle_start_date),
// then returns the canonical row.
func (r *IncomeRepo) Upsert(ctx context.Context, userID string, cycleStartDate time.Time, amount int64) (Income, error) {
	id := uuid.NewString()
	cycleStr := cycleStartDate.Format(dateLayout)
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO incomes (id, user_id, cycle_start_date, amount)
		 VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE amount = VALUES(amount)`,
		id, userID, cycleStr, amount,
	); err != nil {
		return Income{}, err
	}

	var out Income
	var d time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT id, cycle_start_date, amount FROM incomes
		 WHERE user_id = ? AND cycle_start_date = ?`,
		userID, cycleStr,
	).Scan(&out.ID, &d, &out.Amount)
	if err != nil {
		// Fallback to what we just inserted (id may differ from an existing row).
		return Income{ID: id, CycleStartDate: cycleStr, Amount: amount}, nil
	}
	out.CycleStartDate = d.Format(dateLayout)
	return out, nil
}

// Get returns the declared income for (userID, cycleStartDate), or
// sql.ErrNoRows when nothing has been declared for that cycle.
func (r *IncomeRepo) Get(ctx context.Context, userID string, cycleStartDate time.Time) (Income, error) {
	var out Income
	var d time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT id, cycle_start_date, amount FROM incomes
		 WHERE user_id = ? AND cycle_start_date = ?`,
		userID, cycleStartDate.Format(dateLayout),
	).Scan(&out.ID, &d, &out.Amount)
	if err != nil {
		return Income{}, err
	}
	out.CycleStartDate = d.Format(dateLayout)
	return out, nil
}
