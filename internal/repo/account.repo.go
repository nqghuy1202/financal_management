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

// ErrAccountNotFound trả về khi không tìm thấy ví, hoặc ví đó thuộc về
// người dùng khác.
//
// Cố tình không phân biệt hai trường hợp: nếu trả về "ví này của người
// khác" thì kẻ tấn công dò được id nào đang tồn tại trong hệ thống.
var ErrAccountNotFound = errors.New("không tìm thấy ví")

// AccountRepo là các thao tác dữ liệu với ví.
//
// Mọi phương thức đều nhận userID và truyền xuống câu SQL, nên không có
// đường nào đọc hay sửa ví của người khác.
type AccountRepo interface {
	Create(ctx context.Context, arg sqlc.CreateAccountParams) (sqlc.Account, error)
	Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Account, error)
	List(ctx context.Context, userID uuid.UUID) ([]sqlc.Account, error)
	Update(ctx context.Context, arg sqlc.UpdateAccountParams) (sqlc.Account, error)
	SoftDelete(ctx context.Context, id, userID uuid.UUID) (sqlc.Account, error)
	CountTransactions(ctx context.Context, accountID uuid.UUID) (int64, error)
}

type accountRepo struct {
	q *sqlc.Queries
}

func NewAccountRepo(db *pgxpool.Pool) AccountRepo {
	return &accountRepo{q: sqlc.New(db)}
}

func (r *accountRepo) Create(ctx context.Context, arg sqlc.CreateAccountParams) (sqlc.Account, error) {
	account, err := r.q.CreateAccount(ctx, arg)
	if err != nil {
		if isUniqueViolation(err) {
			return sqlc.Account{}, ErrDuplicate
		}
		return sqlc.Account{}, fmt.Errorf("tạo ví thất bại: %w", err)
	}
	return account, nil
}

func (r *accountRepo) Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Account, error) {
	account, err := r.q.GetAccount(ctx, sqlc.GetAccountParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Account{}, ErrAccountNotFound
		}
		return sqlc.Account{}, fmt.Errorf("lấy ví thất bại: %w", err)
	}
	return account, nil
}

func (r *accountRepo) List(ctx context.Context, userID uuid.UUID) ([]sqlc.Account, error) {
	accounts, err := r.q.ListAccounts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("liệt kê ví thất bại: %w", err)
	}
	return accounts, nil
}

func (r *accountRepo) Update(ctx context.Context, arg sqlc.UpdateAccountParams) (sqlc.Account, error) {
	account, err := r.q.UpdateAccount(ctx, arg)
	if err != nil {
		// Không có dòng nào được cập nhật nghĩa là ví không tồn tại hoặc
		// thuộc người dùng khác — điều kiện user_id nằm ngay trong câu SQL.
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Account{}, ErrAccountNotFound
		}
		if isUniqueViolation(err) {
			return sqlc.Account{}, ErrDuplicate
		}
		return sqlc.Account{}, fmt.Errorf("cập nhật ví thất bại: %w", err)
	}
	return account, nil
}

func (r *accountRepo) SoftDelete(ctx context.Context, id, userID uuid.UUID) (sqlc.Account, error) {
	account, err := r.q.SoftDeleteAccount(ctx, sqlc.SoftDeleteAccountParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Account{}, ErrAccountNotFound
		}
		return sqlc.Account{}, fmt.Errorf("xoá ví thất bại: %w", err)
	}
	return account, nil
}

func (r *accountRepo) CountTransactions(ctx context.Context, accountID uuid.UUID) (int64, error) {
	// Cột account_id cho phép NULL nên sqlc sinh tham số kiểu con trỏ.
	count, err := r.q.CountTransactionsByAccount(ctx, &accountID)
	if err != nil {
		return 0, fmt.Errorf("đếm giao dịch của ví thất bại: %w", err)
	}
	return count, nil
}
