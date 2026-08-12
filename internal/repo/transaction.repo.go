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

// ErrTransactionNotFound trả về khi không tìm thấy giao dịch, hoặc giao
// dịch đó thuộc người dùng khác.
var ErrTransactionNotFound = errors.New("không tìm thấy giao dịch")

// TransactionRepo là các thao tác dữ liệu với giao dịch.
type TransactionRepo interface {
	Create(ctx context.Context, arg sqlc.CreateTransactionParams) (sqlc.Transaction, error)
	Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Transaction, error)
	Update(ctx context.Context, arg sqlc.UpdateTransactionParams) (sqlc.Transaction, error)
	SoftDelete(ctx context.Context, id, userID uuid.UUID) (sqlc.Transaction, error)
	List(ctx context.Context, arg sqlc.ListTransactionsParams) ([]sqlc.Transaction, error)
	Count(ctx context.Context, arg sqlc.CountTransactionsParams) (int64, error)
}

type transactionRepo struct {
	q *sqlc.Queries
}

func NewTransactionRepo(db *pgxpool.Pool) TransactionRepo {
	return &transactionRepo{q: sqlc.New(db)}
}

func (r *transactionRepo) Create(ctx context.Context, arg sqlc.CreateTransactionParams) (sqlc.Transaction, error) {
	tx, err := r.q.CreateTransaction(ctx, arg)
	if err != nil {
		return sqlc.Transaction{}, fmt.Errorf("tạo giao dịch thất bại: %w", err)
	}
	return tx, nil
}

func (r *transactionRepo) Get(ctx context.Context, id, userID uuid.UUID) (sqlc.Transaction, error) {
	tx, err := r.q.GetTransaction(ctx, sqlc.GetTransactionParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Transaction{}, ErrTransactionNotFound
		}
		return sqlc.Transaction{}, fmt.Errorf("lấy giao dịch thất bại: %w", err)
	}
	return tx, nil
}

func (r *transactionRepo) Update(ctx context.Context, arg sqlc.UpdateTransactionParams) (sqlc.Transaction, error) {
	tx, err := r.q.UpdateTransaction(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Transaction{}, ErrTransactionNotFound
		}
		return sqlc.Transaction{}, fmt.Errorf("cập nhật giao dịch thất bại: %w", err)
	}
	return tx, nil
}

func (r *transactionRepo) SoftDelete(ctx context.Context, id, userID uuid.UUID) (sqlc.Transaction, error) {
	tx, err := r.q.SoftDeleteTransaction(ctx, sqlc.SoftDeleteTransactionParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Transaction{}, ErrTransactionNotFound
		}
		return sqlc.Transaction{}, fmt.Errorf("xoá giao dịch thất bại: %w", err)
	}
	return tx, nil
}

func (r *transactionRepo) List(ctx context.Context, arg sqlc.ListTransactionsParams) ([]sqlc.Transaction, error) {
	items, err := r.q.ListTransactions(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("liệt kê giao dịch thất bại: %w", err)
	}
	return items, nil
}

func (r *transactionRepo) Count(ctx context.Context, arg sqlc.CountTransactionsParams) (int64, error) {
	count, err := r.q.CountTransactions(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("đếm giao dịch thất bại: %w", err)
	}
	return count, nil
}
