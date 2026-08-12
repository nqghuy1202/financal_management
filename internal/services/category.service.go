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

// CategoryService xử lý nghiệp vụ liên quan tới danh mục thu/chi.
type CategoryService struct {
	categories repo.CategoryRepo
}

func NewCategoryService(categories repo.CategoryRepo) *CategoryService {
	return &CategoryService{categories: categories}
}

type CreateCategoryInput struct {
	UserID uuid.UUID
	Name   string
	Type   string
	Icon   string
	Color  string
}

type UpdateCategoryInput struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Name   string
	Icon   string
	Color  string
}

// List trả về danh mục hệ thống cộng danh mục riêng của người dùng.
//
// categoryType để rỗng thì lấy cả thu lẫn chi.
func (s *CategoryService) List(ctx context.Context, userID uuid.UUID, categoryType string) ([]sqlc.Category, error) {
	if categoryType != "" && categoryType != model.CategoryTypeIncome && categoryType != model.CategoryTypeExpense {
		return nil, response.Newf(response.CodeValidationFailed,
			"Loại danh mục không hợp lệ: %s", categoryType)
	}

	categories, err := s.categories.List(ctx, userID, categoryType)
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}
	return categories, nil
}

// Get lấy một danh mục, kể cả danh mục hệ thống.
func (s *CategoryService) Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Category, error) {
	category, err := s.categories.Get(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repo.ErrCategoryNotFound) {
			return sqlc.Category{}, response.New(response.CodeNotFound)
		}
		return sqlc.Category{}, response.Wrap(response.CodeDatabaseError, err)
	}
	return category, nil
}

// Create tạo danh mục riêng cho người dùng.
func (s *CategoryService) Create(ctx context.Context, in CreateCategoryInput) (sqlc.Category, error) {
	if in.Type != model.CategoryTypeIncome && in.Type != model.CategoryTypeExpense {
		return sqlc.Category{}, response.Newf(response.CodeValidationFailed,
			"Loại danh mục không hợp lệ: %s", in.Type)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Category{}, response.Wrap(response.CodeInternalError, err)
	}

	category, err := s.categories.Create(ctx, sqlc.CreateCategoryParams{
		ID:     id,
		UserID: &in.UserID,
		Name:   strings.TrimSpace(in.Name),
		Type:   in.Type,
		Icon:   in.Icon,
		Color:  in.Color,
	})
	if err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			return sqlc.Category{}, response.Newf(response.CodeConflict,
				"Bạn đã có danh mục tên %q rồi", in.Name)
		}
		return sqlc.Category{}, response.Wrap(response.CodeDatabaseError, err)
	}

	return category, nil
}

// Update sửa danh mục của chính người dùng.
//
// Không sửa được danh mục hệ thống: câu SQL lọc theo user_id, mà danh mục
// hệ thống có user_id NULL nên không bao giờ khớp. Cũng không cho đổi
// loại thu/chi — các giao dịch cũ đã gắn với loại đó rồi, đổi đi sẽ làm
// báo cáo cũ sai.
func (s *CategoryService) Update(ctx context.Context, in UpdateCategoryInput) (sqlc.Category, error) {
	category, err := s.categories.Update(ctx, sqlc.UpdateCategoryParams{
		ID:     in.ID,
		UserID: &in.UserID,
		Name:   strings.TrimSpace(in.Name),
		Icon:   in.Icon,
		Color:  in.Color,
	})
	if err != nil {
		if errors.Is(err, repo.ErrCategoryNotFound) {
			return sqlc.Category{}, response.New(response.CodeNotFound)
		}
		if errors.Is(err, repo.ErrDuplicate) {
			return sqlc.Category{}, response.Newf(response.CodeConflict,
				"Bạn đã có danh mục tên %q rồi", in.Name)
		}
		return sqlc.Category{}, response.Wrap(response.CodeDatabaseError, err)
	}
	return category, nil
}

// Delete xoá mềm một danh mục của người dùng.
func (s *CategoryService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	count, err := s.categories.CountTransactions(ctx, id)
	if err != nil {
		return response.Wrap(response.CodeDatabaseError, err)
	}
	if count > 0 {
		return response.Newf(response.CodeConflict,
			"Danh mục này còn %d giao dịch, không thể xoá", count)
	}

	if _, err := s.categories.SoftDelete(ctx, id, userID); err != nil {
		if errors.Is(err, repo.ErrCategoryNotFound) {
			// Cũng rơi vào đây khi người dùng cố xoá danh mục hệ thống.
			return response.New(response.CodeNotFound)
		}
		return response.Wrap(response.CodeDatabaseError, err)
	}

	return nil
}
