package api

import (
	"context"
	"database/sql"
)

// SettingsRepo is the data-access layer for the user's 1:1 settings row.
type SettingsRepo struct{ db dbtx }

func NewSettingsRepo(db dbtx) *SettingsRepo { return &SettingsRepo{db: db} }

// Upsert sets the caller's savings goal and cycle start day together: inserts
// a new settings row or updates the existing row's savings_goal and
// cycle_start_day in place — mirrors IncomeRepo.Upsert's
// INSERT-ON-DUPLICATE-then-re-SELECT shape.
func (r *SettingsRepo) Upsert(ctx context.Context, userID string, savingsGoal int64, cycleStartDay int) (Settings, error) {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO user_settings (user_id, savings_goal, cycle_start_day)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE savings_goal = VALUES(savings_goal), cycle_start_day = VALUES(cycle_start_day)`,
		userID, savingsGoal, cycleStartDay,
	); err != nil {
		return Settings{}, err
	}

	var out Settings
	err := r.db.QueryRowContext(ctx,
		`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = ?`,
		userID,
	).Scan(&out.SavingsGoal, &out.CycleStartDay)
	if err != nil {
		// Fallback to what we just wrote if the re-SELECT itself fails.
		return Settings{SavingsGoal: savingsGoal, CycleStartDay: cycleStartDay}, nil
	}
	return out, nil
}

// Get returns the caller's settings row, or the DB column defaults
// (savings_goal 0, cycle_start_day 1) when no row exists yet — never
// sql.ErrNoRows, so callers never need a not-found branch for settings.
func (r *SettingsRepo) Get(ctx context.Context, userID string) (Settings, error) {
	var out Settings
	err := r.db.QueryRowContext(ctx,
		`SELECT savings_goal, cycle_start_day FROM user_settings WHERE user_id = ?`,
		userID,
	).Scan(&out.SavingsGoal, &out.CycleStartDay)
	if err == sql.ErrNoRows {
		return Settings{SavingsGoal: 0, CycleStartDay: 1}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	return out, nil
}
