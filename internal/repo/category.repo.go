package repo

import (
	"context"
	"errors"
	"fmt"

	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrCategoryNotFound trả về khi không tìm thấy danh mục, hoặc danh mục
// đó thuộc người dùng khác.
var ErrCategoryNotFound = errors.New("không tìm thấy danh mục")

// CategoryRepo là các thao tác dữ liệu với danh mục.
type CategoryRepo interface {
	// List trả về danh mục hệ thống cộng danh mục riêng của người dùng.
	// categoryType để rỗng thì lấy cả thu lẫn chi.
	List(ctx context.Context, userID uuid.UUID, categoryType string) ([]sqlc.Category, error)
	Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Category, error)
	Create(ctx context.Context, arg sqlc.CreateCategoryParams) (sqlc.Category, error)
	Update(ctx context.Context, arg sqlc.UpdateCategoryParams) (sqlc.Category, error)
	SoftDelete(ctx context.Context, id, userID uuid.UUID) (sqlc.Category, error)
	CountTransactions(ctx context.Context, categoryID uuid.UUID) (int64, error)
}

type categoryRepo struct {
	q *sqlc.Queries
}

func NewCategoryRepo(db *pgxpool.Pool) CategoryRepo {
	return &categoryRepo{q: sqlc.New(db)}
}

func (r *categoryRepo) List(ctx context.Context, userID uuid.UUID, categoryType string) ([]sqlc.Category, error) {
	arg := sqlc.ListCategoriesParams{UserID: &userID}
	// Con trỏ nil tương ứng với NULL trong câu SQL, nghĩa là không lọc.
	if categoryType != "" {
		arg.Type = &categoryType
	}

	categories, err := r.q.ListCategories(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("liệt kê danh mục thất bại: %w", err)
	}
	return categories, nil
}

func (r *categoryRepo) Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Category, error) {
	category, err := r.q.GetCategory(ctx, sqlc.GetCategoryParams{ID: id, UserID: &userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Category{}, ErrCategoryNotFound
		}
		return sqlc.Category{}, fmt.Errorf("lấy danh mục thất bại: %w", err)
	}
	return category, nil
}

func (r *categoryRepo) Create(ctx context.Context, arg sqlc.CreateCategoryParams) (sqlc.Category, error) {
	category, err := r.q.CreateCategory(ctx, arg)
	if err != nil {
		if isUniqueViolation(err) {
			return sqlc.Category{}, ErrDuplicate
		}
		return sqlc.Category{}, fmt.Errorf("tạo danh mục thất bại: %w", err)
	}
	return category, nil
}

func (r *categoryRepo) Update(ctx context.Context, arg sqlc.UpdateCategoryParams) (sqlc.Category, error) {
	category, err := r.q.UpdateCategory(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Category{}, ErrCategoryNotFound
		}
		if isUniqueViolation(err) {
			return sqlc.Category{}, ErrDuplicate
		}
		return sqlc.Category{}, fmt.Errorf("cập nhật danh mục thất bại: %w", err)
	}
	return category, nil
}

func (r *categoryRepo) SoftDelete(ctx context.Context, id, userID uuid.UUID) (sqlc.Category, error) {
	category, err := r.q.SoftDeleteCategory(ctx, sqlc.SoftDeleteCategoryParams{ID: id, UserID: &userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Category{}, ErrCategoryNotFound
		}
		return sqlc.Category{}, fmt.Errorf("xoá danh mục thất bại: %w", err)
	}
	return category, nil
}

func (r *categoryRepo) CountTransactions(ctx context.Context, categoryID uuid.UUID) (int64, error) {
	count, err := r.q.CountTransactionsByCategory(ctx, &categoryID)
	if err != nil {
		return 0, fmt.Errorf("đếm giao dịch của danh mục thất bại: %w", err)
	}
	return count, nil
}
