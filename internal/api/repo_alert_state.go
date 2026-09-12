package api

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AlertStateRepo is the data-access layer for budget-threshold alert state
// (Story 2.1). It is only ever constructed inline over a *sql.Tx inside the
// transaction that performs the triggering transaction write — there is no
// standalone Handler field, matching the SettingsRepo/IncomeRepo precedent in
// UpdateCycleSettings.
type AlertStateRepo struct{ db dbtx }

func NewAlertStateRepo(db dbtx) *AlertStateRepo { return &AlertStateRepo{db: db} }

// Trigger records that (userID, categoryID, cycleStart, threshold) has been
// crossed this cycle. The insert is an idempotent ON DUPLICATE KEY UPDATE
// (matching every other upsert in this codebase, e.g. BudgetRepo.Upsert):
// a second call for the same key updates the existing row in place instead
// of erroring or duplicating, so two concurrent requests crossing the same
// new threshold both succeed and only one row persists.
func (r *AlertStateRepo) Trigger(ctx context.Context, userID, categoryID string, cycleStart time.Time, threshold int) error {
	id := uuid.NewString()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO budget_alert_state (id, user_id, category_id, cycle_start_date, threshold)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE threshold = VALUES(threshold)`,
		id, userID, categoryID, cycleStart.Format(dateLayout), threshold,
	)
	return err
}
