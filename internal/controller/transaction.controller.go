package controller

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo/sqlc"
	"financal_management/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionController struct {
	transactions *services.TransactionService
}

func NewTransactionController(transactions *services.TransactionService) *TransactionController {
	return &TransactionController{transactions: transactions}
}

// ---------------------------------------------------------------------
// Kiểu dữ liệu vào/ra
// ---------------------------------------------------------------------

type transactionRequest struct {
	Type string `json:"type" binding:"required,oneof=income expense"`
	// Số tiền gửi dưới dạng CHUỖI, ví dụ "125000.50". Không dùng kiểu số
	// vì JavaScript đọc số JSON thành float64 và làm tròn sai với số lớn.
	Amount     string     `json:"amount"      binding:"required,numeric"`
	CategoryID *uuid.UUID `json:"category_id"`
	AccountID  *uuid.UUID `json:"account_id"`
	Note       string     `json:"note"        binding:"max=500"`
	// Thời điểm thu/chi thực tế. Bỏ trống thì lấy thời điểm hiện tại.
	OccurredAt *time.Time `json:"occurred_at"`
}

type transactionResponse struct {
	ID         uuid.UUID   `json:"id"`
	Type       string      `json:"type"`
	Amount     model.Money `json:"amount"`
	CategoryID *uuid.UUID  `json:"category_id"`
	AccountID  *uuid.UUID  `json:"account_id"`
	Note       string      `json:"note"`
	OccurredAt time.Time   `json:"occurred_at"`
	CreatedAt  time.Time   `json:"created_at"`
}

func toTransactionResponse(t sqlc.Transaction) (transactionResponse, error) {
	amount, err := model.NewMoney(t.Amount, t.Currency)
	if err != nil {
		return transactionResponse{}, err
	}
	return transactionResponse{
		ID:         t.ID,
		Type:       t.Type,
		Amount:     amount,
		CategoryID: t.CategoryID,
		AccountID:  t.AccountID,
		Note:       t.Note,
		OccurredAt: t.OccurredAt,
		CreatedAt:  t.CreatedAt,
	}, nil
}

// ---------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------

// Create ghi nhận một khoản thu hoặc chi. POST /api/v1/transactions
func (tc *TransactionController) Create(c *gin.Context) {
	var req transactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		_ = c.Error(response.Newf(response.CodeValidationFailed,
			"Số tiền không hợp lệ: %s", req.Amount))
		return
	}

	tx, err := tc.transactions.Create(c.Request.Context(), services.CreateTransactionInput{
		UserID:     userID,
		AccountID:  req.AccountID,
		CategoryID: req.CategoryID,
		Type:       req.Type,
		Amount:     amount,
		Note:       req.Note,
		OccurredAt: occurredAtOrNow(req.OccurredAt),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	respondTransaction(c, tx)
}

// Get lấy chi tiết một giao dịch. GET /api/v1/transactions/:id
func (tc *TransactionController) Get(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	tx, err := tc.transactions.Get(c.Request.Context(), id, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	respondTransaction(c, tx)
}

// Update sửa một giao dịch. PUT /api/v1/transactions/:id
func (tc *TransactionController) Update(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req transactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		_ = c.Error(response.Newf(response.CodeValidationFailed,
			"Số tiền không hợp lệ: %s", req.Amount))
		return
	}

	tx, err := tc.transactions.Update(c.Request.Context(), services.UpdateTransactionInput{
		ID:         id,
		UserID:     userID,
		AccountID:  req.AccountID,
		CategoryID: req.CategoryID,
		Type:       req.Type,
		Amount:     amount,
		Note:       req.Note,
		OccurredAt: occurredAtOrNow(req.OccurredAt),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	respondTransaction(c, tx)
}

// Delete xoá một giao dịch. DELETE /api/v1/transactions/:id
func (tc *TransactionController) Delete(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := tc.transactions.Delete(c.Request.Context(), id, userID); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{keyMessage: "Đã xoá giao dịch"})
}

// List liệt kê giao dịch có lọc và phân trang.
//
// GET /api/v1/transactions?type=expense&from=2026-08-01&to=2026-09-01
//
//	&category_id=...&account_id=...&page_size=20&cursor=...
func (tc *TransactionController) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	in := services.ListTransactionsInput{
		UserID: userID,
		Type:   c.Query("type"),
	}

	var err error
	if in.CategoryID, err = optionalUUIDQuery(c, "category_id"); err != nil {
		_ = c.Error(err)
		return
	}
	if in.AccountID, err = optionalUUIDQuery(c, "account_id"); err != nil {
		_ = c.Error(err)
		return
	}
	if in.From, err = optionalTimeQuery(c, "from"); err != nil {
		_ = c.Error(err)
		return
	}
	if in.To, err = optionalTimeQuery(c, "to"); err != nil {
		_ = c.Error(err)
		return
	}
	if in.CursorOccurredAt, in.CursorID, err = decodeCursor(c.Query("cursor")); err != nil {
		_ = c.Error(err)
		return
	}
	if size := c.Query("page_size"); size != "" {
		if in.PageSize, err = strconv.Atoi(size); err != nil {
			_ = c.Error(response.Newf(response.CodeInvalidParam,
				"page_size phải là số: %s", size))
			return
		}
	}

	page, err := tc.transactions.List(c.Request.Context(), in)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]transactionResponse, 0, len(page.Items))
	for _, tx := range page.Items {
		item, err := toTransactionResponse(tx)
		if err != nil {
			_ = c.Error(response.Wrap(response.CodeInternalError, err))
			return
		}
		items = append(items, item)
	}

	response.Success(c, gin.H{
		keyItems: items,
		"total":  page.Total,
		// Client gửi lại nguyên văn giá trị này ở tham số `cursor` để lấy
		// trang kế tiếp. Là null khi đã hết dữ liệu.
		"next_cursor": encodeCursor(page.NextCursorOccurredAt, page.NextCursorID),
	})
}

// ---------------------------------------------------------------------
// Hàm phụ trợ
// ---------------------------------------------------------------------

func respondTransaction(c *gin.Context, tx sqlc.Transaction) {
	item, err := toTransactionResponse(tx)
	if err != nil {
		_ = c.Error(response.Wrap(response.CodeInternalError, err))
		return
	}
	response.Success(c, item)
}

// occurredAtOrNow lấy thời điểm người dùng nhập, hoặc hiện tại nếu bỏ trống.
func occurredAtOrNow(t *time.Time) time.Time {
	if t != nil {
		return *t
	}
	return time.Now()
}

// cursorSeparator ngăn cách hai phần của con trỏ. Dùng ký tự không bao
// giờ xuất hiện trong chuỗi thời gian hay UUID.
const cursorSeparator = "|"

// encodeCursor gói (thời điểm, id) thành một chuỗi mờ cho client.
//
// Vì sao phải mã hoá thay vì trả thẳng hai tham số:
//
//   - Trong query string, dấu `+` được giải mã thành dấu cách. Múi giờ
//     Việt Nam viết là "+07:00", nên gửi thẳng chuỗi RFC3339 qua query
//     sẽ thành "2026-08-12T12:30:00 07:00" và không parse được nữa.
//     Base64url không chứa `+`, `/` hay `=` nên an toàn tuyệt đối.
//   - Client không cần biết bên trong con trỏ có gì. Về sau đổi cách
//     phân trang cũng không phá vỡ client đang chạy.
//
// Trả về nil khi đã hết dữ liệu, để JSON hiện null.
func encodeCursor(occurredAt *time.Time, id *uuid.UUID) *string {
	if occurredAt == nil || id == nil {
		return nil
	}
	raw := occurredAt.Format(time.RFC3339Nano) + cursorSeparator + id.String()
	encoded := base64.RawURLEncoding.EncodeToString([]byte(raw))
	return &encoded
}

// decodeCursor mở chuỗi con trỏ ra thành (thời điểm, id).
//
// Con trỏ rỗng nghĩa là lấy trang đầu tiên.
func decodeCursor(cursor string) (*time.Time, *uuid.UUID, error) {
	if cursor == "" {
		return nil, nil, nil
	}

	invalid := response.New(response.CodeInvalidParam)

	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, nil, invalid
	}

	parts := strings.SplitN(string(decoded), cursorSeparator, 2)
	if len(parts) != 2 {
		return nil, nil, invalid
	}

	occurredAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, nil, invalid
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, nil, invalid
	}

	return &occurredAt, &id, nil
}

// optionalUUIDQuery đọc một tham số UUID tuỳ chọn trên query string.
func optionalUUIDQuery(c *gin.Context, name string) (*uuid.UUID, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, response.Newf(response.CodeInvalidParam,
			"Tham số %s phải là UUID hợp lệ", name)
	}
	return &id, nil
}

// optionalTimeQuery đọc một tham số thời gian tuỳ chọn.
//
// Chấp nhận cả dạng đầy đủ RFC3339 ("2026-08-12T00:00:00+07:00") lẫn dạng
// chỉ có ngày ("2026-08-12").
//
// Dạng chỉ có ngày được hiểu theo MÚI GIỜ ỨNG DỤNG, không phải UTC. Khi
// người dùng lọc "từ 2026-08-01" họ muốn nói 00:00 giờ Việt Nam. Nếu hiểu
// là 00:00 UTC thì mọi giao dịch từ 00:00 tới 07:00 sáng ngày đầu kỳ sẽ
// bị đẩy sang kỳ trước — số liệu báo cáo sai mà không có dấu hiệu gì.
func optionalTimeQuery(c *gin.Context, name string) (*time.Time, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	// RFC3339 tự mang múi giờ nên dùng nguyên.
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation(time.DateOnly, raw, model.AppLocation()); err == nil {
		return &t, nil
	}
	return nil, response.Newf(response.CodeInvalidParam,
		"Tham số %s phải theo định dạng 2026-08-12 hoặc RFC3339", name)
}
