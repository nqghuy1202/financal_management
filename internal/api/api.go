// Package api implements the real MySQL-backed JSON API (auth + CRUD) used in
// production. It is self-contained (own db/models/handlers) and intentionally
// separate from the experimental controller/service/repo scaffold.
//
// All responses use the shared envelope {code, message, data}.
package api

import (
	"context"
	"database/sql"
	"log"

	"github.com/gin-gonic/gin"
)

// dbtx is the subset of *sql.DB / *sql.Tx that repositories depend on. Every
// repository takes one of these instead of a concrete *sql.DB, so the exact
// same repository code can run standalone or inside a transaction — pass a
// *sql.Tx (via withTx) when a handler needs several writes to commit or fail
// together.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Handler struct {
	db     *sql.DB
	secret []byte

	users        *UserRepo
	categories   *CategoryRepo
	transactions *TransactionRepo
	budgets      *BudgetRepo
	fixedCosts   *FixedCostRepo
	incomes      *IncomeRepo
	settings     *SettingsRepo
	recurring    *RecurringTransactionRepo
}

func NewHandler(db *sql.DB, secret []byte) *Handler {
	return &Handler{
		db:           db,
		secret:       secret,
		users:        NewUserRepo(db),
		categories:   NewCategoryRepo(db),
		transactions: NewTransactionRepo(db),
		budgets:      NewBudgetRepo(db),
		fixedCosts:   NewFixedCostRepo(db),
		incomes:      NewIncomeRepo(db),
		settings:     NewSettingsRepo(db),
		recurring:    NewRecurringTransactionRepo(db),
	}
}

// withTx runs fn inside a single database transaction: fn's writes all
// commit together, or all roll back if fn (or the commit itself) fails.
// Build repositories over the *sql.Tx passed to fn (e.g. NewUserRepo(tx)) so
// their queries participate in the same transaction.
func (h *Handler) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("withTx: rollback after panic failed: %v", rbErr)
			}
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("withTx: rollback after error failed: %v", rbErr)
		}
		return err
	}
	return tx.Commit()
}

// ---- domain models (JSON matches the frontend types) ----

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Category struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"` // income | expense
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

type Transaction struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // income | expense
	Amount     int64  `json:"amount"`
	CategoryID string `json:"categoryId"`
	Note       string `json:"note"`
	Date       string `json:"date"` // yyyy-mm-dd
}

// Budget's Spent/Percent/Status are computed fresh on every GET /budgets and
// POST /budgets response (Story 2.3, via attachBudgetStatus) — never
// stored — so the Budgets page and Dashboard, which both read this same
// field set, can never disagree.
type Budget struct {
	ID         string `json:"id"`
	CategoryID string `json:"categoryId"`
	Limit      int64  `json:"limit"`
	Month      string `json:"month"` // yyyy-mm
	Spent      int64  `json:"spent"`
	Percent    int    `json:"percent"`
	Status     string `json:"status"` // within | near | over
}

// Income is a user's declared income for one budget cycle.
type Income struct {
	ID             string `json:"id"`
	CycleStartDate string `json:"cycleStartDate"` // yyyy-mm-dd
	Amount         int64  `json:"amount"`
}

// FixedCost is a recurring committed cost (rent, subscriptions, ...). It is a
// live list — not snapshotted per cycle.
type FixedCost struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Amount int64  `json:"amount"`
}

// RecurringTransaction is a template that generates suggested-draft
// transactions on a weekly/monthly schedule — never a real transactions row
// on its own (see ConfirmRecurringTransaction). Deliberately separate from
// FixedCost (a live budgeting constant with no date/frequency/transaction
// link) — see spec-recurring-transactions.md.
//
// DueDraftDate mirrors Budget's Spent/Percent/Status: computed fresh on
// every GET /recurring-transactions response by attachDueDraft, never
// stored. It is non-empty exactly when the template is active and due
// (NextDueDate is today or earlier), and holds the date of its most recent
// unconfirmed occurrence — any number of missed occurrences collapse into
// that single date (the "catch up" behavior).
type RecurringTransaction struct {
	ID           string `json:"id"`
	Type         string `json:"type"` // income | expense
	Amount       int64  `json:"amount"`
	CategoryID   string `json:"categoryId"`
	Note         string `json:"note"`
	Frequency    string `json:"frequency"`   // weekly | monthly
	NextDueDate  string `json:"nextDueDate"` // yyyy-mm-dd — the template's own schedule
	Active       bool   `json:"active"`
	DueDraftDate string `json:"dueDraftDate,omitempty"` // yyyy-mm-dd, computed — see doc comment above
}

// Settings holds a user's 1:1 account settings.
type Settings struct {
	SavingsGoal   int64 `json:"savingsGoal"`
	CycleStartDay int   `json:"cycleStartDay"`
}

// CycleSettingsResult is the combined response of PUT /cycle-settings: the
// canonical income row for the (newly computed) current cycle, plus the
// canonical settings row — both written together in the same save.
type CycleSettingsResult struct {
	Income   Income   `json:"income"`
	Settings Settings `json:"settings"`
}

// CycleSummary is the fixed-shape response for GET /cycle/summary
// (architecture spans FR-1/FR-6/FR-7 for this contract). FR-1 (Story 1.5)
// computes SafeToSpend/DaysRemaining/Income/PreviousIncome; FR-6 (Story 2.2)
// populates ActiveAlerts with real data. Budgets ships empty until Story 2.3
// populates it, without a contract change. SavingsGoal/CycleStartDay were
// folded in later (no separate GET /settings exists) so the frontend can
// read the user's real saved settings instead of defaulting to the DB
// column defaults on every fresh session — see deferred-work.md.
type CycleSummary struct {
	SafeToSpend    int64         `json:"safeToSpend"`
	DaysRemaining  int           `json:"daysRemaining"`
	Income         *int64        `json:"income"`
	PreviousIncome *int64        `json:"previousIncome"`
	Budgets        []any         `json:"budgets"`
	ActiveAlerts   []ActiveAlert `json:"activeAlerts"`
	SavingsGoal    int64         `json:"savingsGoal"`
	CycleStartDay  int           `json:"cycleStartDay"`
	// CurrentMonth is cycleStart's calendar month ("2006-01"), i.e. the
	// budgets.month value that identifies the current cycle (AD-4, Story
	// 2.3) — the frontend uses this instead of the wall-clock's own calendar
	// month, so a cycleStartDay other than 1 doesn't mismatch which budgets
	// are "current."
	CurrentMonth string `json:"currentMonth"`
}

// ---- response helpers ----

func ok(c *gin.Context, data any) {
	c.JSON(200, gin.H{"code": 20000, "message": "OK", "data": data})
}

func fail(c *gin.Context, status, code int, message string) {
	c.JSON(status, gin.H{"code": code, "message": message, "data": nil})
	c.Abort()
}
