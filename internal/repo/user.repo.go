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

// ErrUserNotFound trả về khi không tìm thấy người dùng.
//
// Khai báo lỗi riêng thay vì để lộ pgx.ErrNoRows ra ngoài, để tầng
// service không phải biết dự án đang dùng driver database nào.
var ErrUserNotFound = errors.New("không tìm thấy người dùng")

// UserRepo là các thao tác dữ liệu liên quan tới người dùng.
//
// Khai báo interface để tầng service phụ thuộc vào nó thay vì vào cài
// đặt cụ thể — nhờ vậy test service dùng được bản giả, không cần database.
type UserRepo interface {
	Create(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)
	GetByEmail(ctx context.Context, email string) (sqlc.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
}

type userRepo struct {
	q *sqlc.Queries
}

// NewUserRepo tạo repo từ connection pool.
func NewUserRepo(db *pgxpool.Pool) UserRepo {
	return &userRepo{q: sqlc.New(db)}
}

func (r *userRepo) Create(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	user, err := r.q.CreateUser(ctx, arg)
	if err != nil {
		return sqlc.User{}, fmt.Errorf("tạo người dùng thất bại: %w", err)
	}
	return user, nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	user, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		// pgx trả ErrNoRows khi truy vấn :one không có kết quả. Đổi thành
		// lỗi của package này để tầng trên xử lý được mà không import pgx.
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.User{}, ErrUserNotFound
		}
		return sqlc.User{}, fmt.Errorf("lấy người dùng theo id thất bại: %w", err)
	}
	return user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (sqlc.User, error) {
	user, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.User{}, ErrUserNotFound
		}
		return sqlc.User{}, fmt.Errorf("lấy người dùng theo email thất bại: %w", err)
	}
	return user, nil
}

func (r *userRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	exists, err := r.q.EmailExists(ctx, email)
	if err != nil {
		return false, fmt.Errorf("kiểm tra email tồn tại thất bại: %w", err)
	}
	return exists, nil
}
