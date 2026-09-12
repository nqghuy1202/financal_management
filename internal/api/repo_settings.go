package api

import "context"

// SettingsRepo is the data-access layer for the user's 1:1 settings row.
type SettingsRepo struct{ db dbtx }

func NewSettingsRepo(db dbtx) *SettingsRepo { return &SettingsRepo{db: db} }

// UpsertSavingsGoal sets the caller's savings goal: inserts a new settings
// row (cycle_start_day keeps its default, 1) or updates the existing row's
// savings_goal in place, leaving cycle_start_day untouched — mirrors
// IncomeRepo.Upsert's INSERT-ON-DUPLICATE-then-re-SELECT shape.
func (r *SettingsRepo) UpsertSavingsGoal(ctx context.Context, userID string, savingsGoal int64) (Settings, error) {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO user_settings (user_id, savings_goal)
		 VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE savings_goal = VALUES(savings_goal)`,
		userID, savingsGoal,
	); err != nil {
		return Settings{}, err
	}

	var out Settings
	err := r.db.QueryRowContext(ctx,
		`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = ?`,
		userID,
	).Scan(&out.SavingsGoal, &out.CycleStartDay)
	if err != nil {
		// Fallback to what we just wrote (cycle_start_day defaults to 1 for a
		// brand-new row) if the re-SELECT itself fails.
		return Settings{SavingsGoal: savingsGoal, CycleStartDay: 1}, nil
	}
	return out, nil
}
