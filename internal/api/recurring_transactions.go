package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// recurringTransactionInput is the body of POST /recurring-transactions: the
// transaction being marked recurring (logged separately via the existing
// POST /transactions — this endpoint only creates the template that governs
// *future* occurrences), plus the chosen frequency.
type recurringTransactionInput struct {
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	CategoryID string `json:"categoryId"`
	Note       string `json:"note"`
	Frequency  string `json:"frequency"`
	Date       string `json:"date"`
}

func (in recurringTransactionInput) validate() bool {
	if in.Type != "income" && in.Type != "expense" {
		return false
	}
	if in.Amount <= 0 || in.CategoryID == "" {
		return false
	}
	if in.Frequency != "weekly" && in.Frequency != "monthly" {
		return false
	}
	if _, err := time.Parse(dateLayout, in.Date); err != nil {
		return false
	}
	return true
}

// recurringTransactionUpdateInput is the body of PUT /recurring-transactions/:id:
// a full replace of the template's editable fields, including Active — so
// pausing/resuming a template is just a PUT with active:false/true, the same
// way DELETE removes it, with no separate pause endpoint.
type recurringTransactionUpdateInput struct {
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	CategoryID string `json:"categoryId"`
	Note       string `json:"note"`
	Frequency  string `json:"frequency"`
	Active     bool   `json:"active"`
}

func (in recurringTransactionUpdateInput) validate() bool {
	if in.Type != "income" && in.Type != "expense" {
		return false
	}
	if in.Amount <= 0 || in.CategoryID == "" {
		return false
	}
	if in.Frequency != "weekly" && in.Frequency != "monthly" {
		return false
	}
	return true
}

// ListRecurringTransactions returns every template for the caller (active
// and paused), with DueDraftDate computed fresh per request (attachDueDraft)
// — no scheduler, no stored "is due" flag, computed on demand exactly like
// GET /cycle/summary computes SafeToSpend.
func (h *Handler) ListRecurringTransactions(c *gin.Context) {
	list, err := h.recurring.List(c.Request.Context(), userIDFrom(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 50060, "Không thể tải giao dịch định kỳ")
		return
	}
	now := time.Now()
	for i := range list {
		attachDueDraft(&list[i], now)
	}
	ok(c, list)
}

// CreateRecurringTransaction creates the template for future occurrences of
// a transaction the caller just marked recurring. It never writes to
// `transactions` itself — the transaction being marked recurring is logged
// normally via POST /transactions, by the same modal submit. The template's
// first NextDueDate is one frequency step after in.Date.
func (h *Handler) CreateRecurringTransaction(c *gin.Context) {
	var in recurringTransactionInput
	if err := c.ShouldBindJSON(&in); err != nil || !in.validate() {
		fail(c, http.StatusBadRequest, 40060, "Dữ liệu giao dịch định kỳ không hợp lệ")
		return
	}
	ctx := c.Request.Context()
	uid := userIDFrom(c)

	owns, err := h.categories.Owns(ctx, uid, in.CategoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50061, "Lỗi máy chủ")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40061, "Danh mục không tồn tại")
		return
	}

	baseDate, err := time.ParseInLocation(dateLayout, in.Date, time.Local)
	if err != nil {
		fail(c, http.StatusBadRequest, 40060, "Dữ liệu giao dịch định kỳ không hợp lệ")
		return
	}
	nextDue := stepFrequency(baseDate, in.Frequency)

	rt := RecurringTransaction{
		ID:          uuid.NewString(),
		Type:        in.Type,
		Amount:      in.Amount,
		CategoryID:  in.CategoryID,
		Note:        strings.TrimSpace(in.Note),
		Frequency:   in.Frequency,
		NextDueDate: nextDue.Format(dateLayout),
		Active:      true,
	}
	if err := h.recurring.Create(ctx, uid, rt); err != nil {
		fail(c, http.StatusInternalServerError, 50062, "Không thể tạo giao dịch định kỳ")
		return
	}
	ok(c, rt)
}

// UpdateRecurringTransaction overwrites a template's editable fields
// (type/amount/category/note/frequency/active). NextDueDate is never
// accepted here — only ConfirmRecurringTransaction advances it, so a
// template's schedule can't be hand-edited into desync with the catch-up
// math.
func (h *Handler) UpdateRecurringTransaction(c *gin.Context) {
	var in recurringTransactionUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil || !in.validate() {
		fail(c, http.StatusBadRequest, 40062, "Dữ liệu giao dịch định kỳ không hợp lệ")
		return
	}
	ctx := c.Request.Context()
	uid := userIDFrom(c)
	id := c.Param("id")

	owns, err := h.categories.Owns(ctx, uid, in.CategoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50063, "Lỗi máy chủ")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40063, "Danh mục không tồn tại")
		return
	}

	rt := RecurringTransaction{
		Type:       in.Type,
		Amount:     in.Amount,
		CategoryID: in.CategoryID,
		Note:       strings.TrimSpace(in.Note),
		Frequency:  in.Frequency,
		Active:     in.Active,
	}
	found, err := h.recurring.Update(ctx, uid, id, rt)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50064, "Không thể cập nhật giao dịch định kỳ")
		return
	}
	if !found {
		fail(c, http.StatusNotFound, 40460, "Không tìm thấy giao dịch định kỳ")
		return
	}

	// Return the canonical row (with its real NextDueDate, untouched by this
	// update) rather than echoing the input, which has no NextDueDate at all.
	saved, err := h.recurring.Get(ctx, uid, id)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50068, "Không thể tải giao dịch định kỳ")
		return
	}
	if saved.ID == "" {
		// Deleted by a concurrent request between the Update() write above and
		// this re-fetch — a real "not found", not a server error.
		fail(c, http.StatusNotFound, 40460, "Không tìm thấy giao dịch định kỳ")
		return
	}
	attachDueDraft(&saved, time.Now())
	ok(c, saved)
}

// DeleteRecurringTransaction removes a template. Already-confirmed
// transactions it previously produced are untouched (they're normal rows in
// `transactions`, with no link back to this template).
func (h *Handler) DeleteRecurringTransaction(c *gin.Context) {
	found, err := h.recurring.Delete(c.Request.Context(), userIDFrom(c), c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, 50065, "Không thể xóa giao dịch định kỳ")
		return
	}
	if !found {
		fail(c, http.StatusNotFound, 40460, "Không tìm thấy giao dịch định kỳ")
		return
	}
	ok(c, gin.H{"deleted": true})
}

// ConfirmRecurringTransaction is the one-tap "confirm" action for a
// suggested draft. In a single DB transaction it (a) inserts the (possibly
// edited) transaction into `transactions` exactly as POST /transactions
// would — including the Story 2.1 budget-threshold check — and (b) advances
// the template's NextDueDate from its own original schedule, never from the
// edited date (I/O matrix "Edit before confirm"). Both writes commit or fail
// together (spec Design Notes), so the template's schedule can never desync
// from what was actually confirmed.
func (h *Handler) ConfirmRecurringTransaction(c *gin.Context) {
	var in transactionInput
	if err := c.ShouldBindJSON(&in); err != nil || !in.validate() {
		fail(c, http.StatusBadRequest, 40064, "Dữ liệu giao dịch không hợp lệ")
		return
	}
	ctx := c.Request.Context()
	uid := userIDFrom(c)
	id := c.Param("id")

	owns, err := h.categories.Owns(ctx, uid, in.CategoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50066, "Lỗi máy chủ")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40065, "Danh mục không tồn tại")
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

	now := time.Now()
	var notFound bool
	err = h.withTx(ctx, func(tx *sql.Tx) error {
		recurRepo := NewRecurringTransactionRepo(tx)
		rt, getErr := recurRepo.Get(ctx, uid, id)
		if getErr != nil {
			return getErr
		}
		if rt.ID == "" || !rt.Active {
			notFound = true
			return nil
		}

		if err := NewTransactionRepo(tx).Create(ctx, uid, t); err != nil {
			return err
		}
		// t.ID is a real, already-persisted id at this point, so it must be
		// excluded from the "before this save" spend total — see
		// checkBudgetThreshold's doc comment.
		if err := checkBudgetThreshold(ctx, tx, uid, t); err != nil {
			return err
		}

		currentNextDue, err := time.ParseInLocation(dateLayout, rt.NextDueDate, now.Location())
		if err != nil {
			return err
		}
		newNextDue := AdvanceRecurrence(currentNextDue, rt.Frequency, now)
		return recurRepo.AdvanceNextDueDate(ctx, uid, id, newNextDue.Format(dateLayout))
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50067, "Không thể xác nhận giao dịch định kỳ")
		return
	}
	if notFound {
		fail(c, http.StatusNotFound, 40461, "Không tìm thấy giao dịch định kỳ")
		return
	}
	ok(c, t)
}
