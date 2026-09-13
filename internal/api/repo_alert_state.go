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

// ActiveAlert is one undismissed budget-threshold alert surfaced to the
// frontend (Story 2.2). Fixed shape per the architecture's contract — no
// percent/spent figures, only enough to render banner copy and let the
// frontend resolve the category name locally.
type ActiveAlert struct {
	CategoryID string `json:"categoryId"`
	Threshold  int    `json:"threshold"`
	Status     string `json:"status"`
}

// ListActive returns every undismissed alert for userID in the cycle
// starting at cycleStart, ordered highest-threshold first (so "over" alerts
// naturally precede "near" ones). Never returns nil — an empty slice when
// there are no active alerts, so callers can marshal it as `[]` rather than
// `null`.
func (r *AlertStateRepo) ListActive(ctx context.Context, userID string, cycleStart time.Time) ([]ActiveAlert, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT category_id, threshold FROM budget_alert_state
		 WHERE user_id = ? AND cycle_start_date = ? AND dismissed_at IS NULL
		 ORDER BY threshold DESC, triggered_at DESC`,
		userID, cycleStart.Format(dateLayout),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := []ActiveAlert{}
	for rows.Next() {
		var categoryID string
		var threshold int
		if err := rows.Scan(&categoryID, &threshold); err != nil {
			return nil, err
		}
		alerts = append(alerts, ActiveAlert{
			CategoryID: categoryID,
			Threshold:  threshold,
			Status:     AlertStatus(threshold),
		})
	}
	return alerts, rows.Err()
}

// Dismiss marks (userID, categoryID, cycleStart, threshold)'s alert as
// dismissed for the rest of the cycle. Idempotent: dismissing an
// already-dismissed or non-existent alert affects 0 rows, which is not an
// error — the caller always gets a clean success.
func (r *AlertStateRepo) Dismiss(ctx context.Context, userID, categoryID string, cycleStart time.Time, threshold int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE budget_alert_state SET dismissed_at = CURRENT_TIMESTAMP
		 WHERE user_id = ? AND category_id = ? AND cycle_start_date = ? AND threshold = ? AND dismissed_at IS NULL`,
		userID, categoryID, cycleStart.Format(dateLayout), threshold,
	)
	return err
}
