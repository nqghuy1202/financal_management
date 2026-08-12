package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo"
	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Giới hạn số bản ghi trả về mỗi lần gọi danh sách.
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// TransactionService xử lý nghiệp vụ ghi nhận thu chi.
//
// Nhận thêm repo của nguồn tiền và danh mục để kiểm tra rằng người dùng
// chỉ gắn giao dịch vào nguồn tiền và danh mục mà họ được phép dùng.
type TransactionService struct {
	transactions repo.TransactionRepo
	accounts     repo.AccountRepo
	categories   repo.CategoryRepo
}

func NewTransactionService(
	transactions repo.TransactionRepo,
	accounts repo.AccountRepo,
	categories repo.CategoryRepo,
) *TransactionService {
	return &TransactionService{
		transactions: transactions,
		accounts:     accounts,
		categories:   categories,
	}
}

type CreateTransactionInput struct {
	UserID     uuid.UUID
	AccountID  *uuid.UUID
	CategoryID *uuid.UUID
	Type       string
	Amount     decimal.Decimal
	Note       string
	OccurredAt time.Time
}

type UpdateTransactionInput struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	AccountID  *uuid.UUID
	CategoryID *uuid.UUID
	Type       string
	Amount     decimal.Decimal
	Note       string
	OccurredAt time.Time
}

// ListTransactionsInput là bộ lọc cho danh sách giao dịch.
//
// Con trỏ phân trang gồm hai phần vì thứ tự sắp xếp là (occurred_at, id).
type ListTransactionsInput struct {
	UserID     uuid.UUID
	Type       string
	CategoryID *uuid.UUID
	AccountID  *uuid.UUID
	From       *time.Time
	To         *time.Time

	CursorOccurredAt *time.Time
	CursorID         *uuid.UUID
	PageSize         int
}

// TransactionPage là một trang kết quả.
type TransactionPage struct {
	Items []sqlc.Transaction
	Total int64
	// NextCursor rỗng khi đã hết dữ liệu.
	NextCursorOccurredAt *time.Time
	NextCursorID         *uuid.UUID
}

// Create ghi nhận một khoản thu hoặc chi.
func (s *TransactionService) Create(ctx context.Context, in CreateTransactionInput) (sqlc.Transaction, error) {
	if err := s.validate(ctx, in.UserID, in.Type, in.Amount, in.AccountID, in.CategoryID); err != nil {
		return sqlc.Transaction{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Transaction{}, response.Wrap(response.CodeInternalError, err)
	}

	tx, err := s.transactions.Create(ctx, sqlc.CreateTransactionParams{
		ID:         id,
		UserID:     in.UserID,
		AccountID:  in.AccountID,
		CategoryID: in.CategoryID,
		Type:       in.Type,
		Amount:     in.Amount,
		Currency:   model.DefaultCurrency,
		Note:       strings.TrimSpace(in.Note),
		OccurredAt: in.OccurredAt,
	})
	if err != nil {
		return sqlc.Transaction{}, response.Wrap(response.CodeDatabaseError, err)
	}

	return tx, nil
}

// Get lấy một giao dịch của người dùng.
func (s *TransactionService) Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Transaction, error) {
	tx, err := s.transactions.Get(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repo.ErrTransactionNotFound) {
			return sqlc.Transaction{}, response.New(response.CodeNotFound)
		}
		return sqlc.Transaction{}, response.Wrap(response.CodeDatabaseError, err)
	}
	return tx, nil
}

// Update sửa một giao dịch.
func (s *TransactionService) Update(ctx context.Context, in UpdateTransactionInput) (sqlc.Transaction, error) {
	if err := s.validate(ctx, in.UserID, in.Type, in.Amount, in.AccountID, in.CategoryID); err != nil {
		return sqlc.Transaction{}, err
	}

	tx, err := s.transactions.Update(ctx, sqlc.UpdateTransactionParams{
		ID:         in.ID,
		UserID:     in.UserID,
		AccountID:  in.AccountID,
		CategoryID: in.CategoryID,
		Type:       in.Type,
		Amount:     in.Amount,
		Note:       strings.TrimSpace(in.Note),
		OccurredAt: in.OccurredAt,
	})
	if err != nil {
		if errors.Is(err, repo.ErrTransactionNotFound) {
			return sqlc.Transaction{}, response.New(response.CodeNotFound)
		}
		return sqlc.Transaction{}, response.Wrap(response.CodeDatabaseError, err)
	}

	return tx, nil
}

// Delete xoá mềm một giao dịch.
func (s *TransactionService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if _, err := s.transactions.SoftDelete(ctx, id, userID); err != nil {
		if errors.Is(err, repo.ErrTransactionNotFound) {
			return response.New(response.CodeNotFound)
		}
		return response.Wrap(response.CodeDatabaseError, err)
	}
	return nil
}

// List trả về một trang giao dịch theo bộ lọc.
func (s *TransactionService) List(ctx context.Context, in ListTransactionsInput) (*TransactionPage, error) {
	if in.Type != "" && in.Type != model.TransactionTypeIncome && in.Type != model.TransactionTypeExpense {
		return nil, response.Newf(response.CodeValidationFailed,
			"Loại giao dịch không hợp lệ: %s", in.Type)
	}
	if in.From != nil && in.To != nil && in.To.Before(*in.From) {
		return nil, response.New(response.CodeValidationFailed)
	}

	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	var typeFilter *string
	if in.Type != "" {
		typeFilter = &in.Type
	}

	// Lấy dư một bản ghi để biết còn trang sau hay không, mà không phải
	// chạy thêm một câu đếm riêng.
	items, err := s.transactions.List(ctx, sqlc.ListTransactionsParams{
		UserID:           in.UserID,
		Type:             typeFilter,
		CategoryID:       in.CategoryID,
		AccountID:        in.AccountID,
		From:             in.From,
		To:               in.To,
		CursorOccurredAt: in.CursorOccurredAt,
		CursorID:         in.CursorID,
		PageSize:         int32(pageSize + 1),
	})
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}

	page := &TransactionPage{}

	if len(items) > pageSize {
		last := items[pageSize-1]
		page.NextCursorOccurredAt = &last.OccurredAt
		page.NextCursorID = &last.ID
		items = items[:pageSize]
	}
	page.Items = items

	total, err := s.transactions.Count(ctx, sqlc.CountTransactionsParams{
		UserID:     in.UserID,
		Type:       typeFilter,
		CategoryID: in.CategoryID,
		AccountID:  in.AccountID,
		From:       in.From,
		To:         in.To,
	})
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}
	page.Total = total

	return page, nil
}

// validate kiểm tra các quy tắc nghiệp vụ dùng chung cho tạo và sửa.
func (s *TransactionService) validate(
	ctx context.Context,
	userID uuid.UUID,
	txType string,
	amount decimal.Decimal,
	accountID, categoryID *uuid.UUID,
) error {
	if txType != model.TransactionTypeIncome && txType != model.TransactionTypeExpense {
		return response.Newf(response.CodeValidationFailed,
			"Loại giao dịch không hợp lệ: %s", txType)
	}
	if !amount.IsPositive() {
		return response.Newf(response.CodeValidationFailed,
			"Số tiền phải lớn hơn 0")
	}

	// Nguồn tiền phải thuộc về chính người dùng. Không kiểm tra thì họ
	// gắn được giao dịch vào nguồn tiền của người khác.
	if accountID != nil {
		if _, err := s.accounts.Get(ctx, *accountID, userID); err != nil {
			if errors.Is(err, repo.ErrAccountNotFound) {
				return response.New(response.CodeNotFound)
			}
			return response.Wrap(response.CodeDatabaseError, err)
		}
	}

	if categoryID != nil {
		category, err := s.categories.Get(ctx, *categoryID, userID)
		if err != nil {
			if errors.Is(err, repo.ErrCategoryNotFound) {
				return response.New(response.CodeNotFound)
			}
			return response.Wrap(response.CodeDatabaseError, err)
		}
		// Khoản chi phải gắn danh mục chi, khoản thu phải gắn danh mục
		// thu. Nếu lẫn lộn, báo cáo "chi tiêu theo danh mục" sẽ hiện ra
		// danh mục Lương nằm trong phần chi tiêu.
		if category.Type != txType {
			return response.Newf(response.CodeValidationFailed,
				"Danh mục %q dùng cho khoản %s, không dùng được cho khoản %s",
				category.Name, viTypeName(category.Type), viTypeName(txType))
		}
	}

	return nil
}

// viTypeName đổi mã loại giao dịch sang tiếng Việt cho thông báo lỗi.
func viTypeName(t string) string {
	if t == model.TransactionTypeIncome {
		return "thu"
	}
	return "chi"
}
