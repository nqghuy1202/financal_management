package api

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type transactionInput struct {
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	CategoryID string `json:"categoryId"`
	Note       string `json:"note"`
	Date       string `json:"date"`
}

func (in transactionInput) validate() bool {
	if in.Type != "income" && in.Type != "expense" {
		return false
	}
	if in.Amount <= 0 || in.CategoryID == "" {
		return false
	}
	if _, err := time.Parse(dateLayout, in.Date); err != nil {
		return false
	}
	return true
}

func (h *Handler) ListTransactions(c *gin.Context) {
	list, err := h.transactions.List(c.Request.Context(), userIDFrom(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 50020, "Không thể tải giao dịch")
		return
	}
	ok(c, list)
}

// CreateTransaction inserts the transaction and, in the same DB transaction,
// runs the Story 2.1 budget-threshold check (see checkBudgetThreshold) so a
// crossing is recorded atomically with the write that caused it.
func (h *Handler) CreateTransaction(c *gin.Context) {
	var in transactionInput
	if err := c.ShouldBindJSON(&in); err != nil || !in.validate() {
		fail(c, http.StatusBadRequest, 40020, "Dữ liệu giao dịch không hợp lệ")
		return
	}
	ctx := c.Request.Context()
	uid := userIDFrom(c)

	owns, err := h.categories.Owns(ctx, uid, in.CategoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50025, "Lỗi máy chủ")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40021, "Danh mục không tồn tại")
		return
	}

	t := Transaction{
		ID:         uuid.NewString(),
		Type:       in.Type,
		Amount:     in.Amount,
		CategoryID: in.CategoryID,
		Note:       strings.TrimSpace(in.Note),
		Date:       in.Date,
	}

	err = h.withTx(ctx, func(tx *sql.Tx) error {
		if err := NewTransactionRepo(tx).Create(ctx, uid, t); err != nil {
			return err
		}
		// t.ID is a real, already-persisted id at this point, so it must be
		// excluded from the "before this save" spend total — see
		// checkBudgetThreshold's doc comment.
		return checkBudgetThreshold(ctx, tx, uid, t)
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50022, "Không thể tạo giao dịch")
		return
	}
	ok(c, t)
}

// UpdateTransaction overwrites the transaction and, in the same DB
// transaction, runs the Story 2.1 budget-threshold check — an update can
// raise a category's spend just as a create can (e.g. increasing the amount
// or moving the transaction into the current cycle/a budgeted category).
func (h *Handler) UpdateTransaction(c *gin.Context) {
	var in transactionInput
	if err := c.ShouldBindJSON(&in); err != nil || !in.validate() {
		fail(c, http.StatusBadRequest, 40022, "Dữ liệu giao dịch không hợp lệ")
		return
	}
	ctx := c.Request.Context()
	uid := userIDFrom(c)
	id := c.Param("id")

	owns, err := h.categories.Owns(ctx, uid, in.CategoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50026, "Lỗi máy chủ")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40023, "Danh mục không tồn tại")
		return
	}

	t := Transaction{ID: id, Type: in.Type, Amount: in.Amount, CategoryID: in.CategoryID, Note: strings.TrimSpace(in.Note), Date: in.Date}

	var found bool
	err = h.withTx(ctx, func(tx *sql.Tx) error {
		var updateErr error
		found, updateErr = NewTransactionRepo(tx).Update(ctx, uid, id, t)
		if updateErr != nil {
			return updateErr
		}
		if !found {
			return nil
		}
		// t.ID (== id) already reflects the updated row at this point, so it
		// must be excluded from the "before this save" spend total — see
		// checkBudgetThreshold's doc comment.
		return checkBudgetThreshold(ctx, tx, uid, t)
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50023, "Không thể cập nhật giao dịch")
		return
	}
	if !found {
		fail(c, http.StatusNotFound, 40420, "Không tìm thấy giao dịch")
		return
	}
	ok(c, t)
}

func (h *Handler) DeleteTransaction(c *gin.Context) {
	if err := h.transactions.Delete(c.Request.Context(), userIDFrom(c), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, 50024, "Không thể xóa giao dịch")
		return
	}
	ok(c, gin.H{"deleted": true})
}

// checkBudgetThreshold runs the Story 2.1 budget-threshold check for one
// saved expense transaction, inside the same DB transaction (tx) as the
// write that produced it (CreateTransaction/UpdateTransaction, via
// h.withTx). It is a silent no-op — nil error, no alert row — for:
//   - non-expense transactions (type != "expense"),
//   - transactions dated outside the current cycle
//     [CycleWindow(settings.CycleStartDay, time.Now())),
//   - categories with no budget row for the current cycle's month,
//   - spend that hasn't newly crossed a threshold (CrossedThreshold returns
//     crossed=false).
//
// The transaction's own id (t.ID) is always excluded from the "before this
// save" spend total: because Create/Update both run before this check,
// inside the same h.withTx, the row being saved already exists in the DB
// under its real id by the time this query runs — including it would
// double-count the row's own amount into "spend before this save".
func checkBudgetThreshold(ctx context.Context, tx *sql.Tx, userID string, t Transaction) error {
	if t.Type != "expense" {
		return nil
	}

	settings, err := NewSettingsRepo(tx).Get(ctx, userID)
	if err != nil {
		return err
	}
	cycleStart, cycleEnd := CycleWindow(settings.CycleStartDay, time.Now())

	txDate, err := time.ParseInLocation(dateLayout, t.Date, cycleStart.Location())
	if err != nil {
		return err
	}
	if txDate.Before(cycleStart) || !txDate.Before(cycleEnd) {
		return nil
	}

	month := cycleStart.Format("2006-01")
	budget, err := NewBudgetRepo(tx).GetByCategoryMonth(ctx, userID, t.CategoryID, month)
	if err = ignoreNoRows(err); err != nil {
		return err
	}
	if budget.ID == "" {
		// No budget row for this category/month: no percentage to check.
		return nil
	}

	prevSpent, err := NewTransactionRepo(tx).SumExpensesInCategoryExcluding(ctx, userID, t.CategoryID, cycleStart, cycleEnd, t.ID)
	if err != nil {
		return err
	}
	newSpent := prevSpent + t.Amount

	threshold, crossed := CrossedThreshold(prevSpent, newSpent, budget.Limit)
	if !crossed {
		return nil
	}

	return NewAlertStateRepo(tx).Trigger(ctx, userID, t.CategoryID, cycleStart, threshold)
}
