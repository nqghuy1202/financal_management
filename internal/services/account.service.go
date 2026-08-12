package services

import (
	"context"
	"errors"
	"strings"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo"
	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
)

// AccountService xử lý nghiệp vụ liên quan tới nguồn tiền.
type AccountService struct {
	accounts repo.AccountRepo
}

func NewAccountService(accounts repo.AccountRepo) *AccountService {
	return &AccountService{accounts: accounts}
}

type CreateAccountInput struct {
	UserID uuid.UUID
	Name   string
	Type   string
	Icon   string
}

type UpdateAccountInput struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Name   string
	Type   string
	Icon   string
}

// Create tạo nguồn tiền mới.
func (s *AccountService) Create(ctx context.Context, in CreateAccountInput) (sqlc.Account, error) {
	if !model.AccountTypes[in.Type] {
		return sqlc.Account{}, response.Newf(response.CodeValidationFailed,
			"Loại nguồn tiền không hợp lệ: %s", in.Type)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Account{}, response.Wrap(response.CodeInternalError, err)
	}

	account, err := s.accounts.Create(ctx, sqlc.CreateAccountParams{
		ID:     id,
		UserID: in.UserID,
		Name:   strings.TrimSpace(in.Name),
		Type:   in.Type,
		Icon:   in.Icon,
	})
	if err != nil {
		// Trùng tên ví trong cùng một tài khoản.
		if errors.Is(err, repo.ErrDuplicate) {
			return sqlc.Account{}, response.Newf(response.CodeConflict,
				"Bạn đã có nguồn tiền tên %q rồi", in.Name)
		}
		return sqlc.Account{}, response.Wrap(response.CodeDatabaseError, err)
	}

	return account, nil
}

// Get lấy một nguồn tiền. Trả lỗi không tìm thấy nếu nó thuộc người khác.
func (s *AccountService) Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Account, error) {
	account, err := s.accounts.Get(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repo.ErrAccountNotFound) {
			return sqlc.Account{}, response.New(response.CodeNotFound)
		}
		return sqlc.Account{}, response.Wrap(response.CodeDatabaseError, err)
	}
	return account, nil
}

// List liệt kê nguồn tiền của một người dùng.
func (s *AccountService) List(ctx context.Context, userID uuid.UUID) ([]sqlc.Account, error) {
	accounts, err := s.accounts.List(ctx, userID)
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}
	return accounts, nil
}

// Update sửa thông tin nguồn tiền.
func (s *AccountService) Update(ctx context.Context, in UpdateAccountInput) (sqlc.Account, error) {
	if !model.AccountTypes[in.Type] {
		return sqlc.Account{}, response.Newf(response.CodeValidationFailed,
			"Loại nguồn tiền không hợp lệ: %s", in.Type)
	}

	account, err := s.accounts.Update(ctx, sqlc.UpdateAccountParams{
		ID:     in.ID,
		UserID: in.UserID,
		Name:   strings.TrimSpace(in.Name),
		Type:   in.Type,
		Icon:   in.Icon,
	})
	if err != nil {
		if errors.Is(err, repo.ErrAccountNotFound) {
			return sqlc.Account{}, response.New(response.CodeNotFound)
		}
		if errors.Is(err, repo.ErrDuplicate) {
			return sqlc.Account{}, response.Newf(response.CodeConflict,
				"Bạn đã có nguồn tiền tên %q rồi", in.Name)
		}
		return sqlc.Account{}, response.Wrap(response.CodeDatabaseError, err)
	}

	return account, nil
}

// Delete xoá mềm một nguồn tiền.
//
// Từ chối nếu còn giao dịch tham chiếu tới nó: xoá đi sẽ làm lịch sử chi
// tiêu mất một phần và các báo cáo cũ không dựng lại được.
func (s *AccountService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	// Kiểm tra quyền sở hữu trước, để ví của người khác cũng trả về
	// "không tìm thấy" chứ không lộ ra là nó có tồn tại.
	if _, err := s.Get(ctx, id, userID); err != nil {
		return err
	}

	count, err := s.accounts.CountTransactions(ctx, id)
	if err != nil {
		return response.Wrap(response.CodeDatabaseError, err)
	}
	if count > 0 {
		return response.Newf(response.CodeConflict,
			"Nguồn tiền này còn %d giao dịch, không thể xoá", count)
	}

	if _, err := s.accounts.SoftDelete(ctx, id, userID); err != nil {
		if errors.Is(err, repo.ErrAccountNotFound) {
			return response.New(response.CodeNotFound)
		}
		return response.Wrap(response.CodeDatabaseError, err)
	}

	return nil
}
