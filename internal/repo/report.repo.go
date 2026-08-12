package repo

import (
	"context"
	"fmt"

	"financal_management/internal/repo/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportRepo là các truy vấn tổng hợp phục vụ báo cáo.
//
// Các truy vấn ở đây chỉ ĐỌC và luôn gom nhóm sẵn trong database. Kéo
// toàn bộ giao dịch về Go rồi cộng bằng vòng lặp sẽ chậm dần theo số bản
// ghi, trong khi Postgres làm việc này bằng một lần quét index.
type ReportRepo interface {
	Summary(ctx context.Context, arg sqlc.ReportSummaryParams) (sqlc.ReportSummaryRow, error)
	ByCategory(ctx context.Context, arg sqlc.ReportByCategoryParams) ([]sqlc.ReportByCategoryRow, error)
	CashFlow(ctx context.Context, arg sqlc.ReportCashFlowParams) ([]sqlc.ReportCashFlowRow, error)
}

type reportRepo struct {
	q *sqlc.Queries
}

func NewReportRepo(db *pgxpool.Pool) ReportRepo {
	return &reportRepo{q: sqlc.New(db)}
}

func (r *reportRepo) Summary(ctx context.Context, arg sqlc.ReportSummaryParams) (sqlc.ReportSummaryRow, error) {
	row, err := r.q.ReportSummary(ctx, arg)
	if err != nil {
		return sqlc.ReportSummaryRow{}, fmt.Errorf("tổng hợp thu chi thất bại: %w", err)
	}
	return row, nil
}

func (r *reportRepo) ByCategory(ctx context.Context, arg sqlc.ReportByCategoryParams) ([]sqlc.ReportByCategoryRow, error) {
	rows, err := r.q.ReportByCategory(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("tổng hợp theo danh mục thất bại: %w", err)
	}
	return rows, nil
}

func (r *reportRepo) CashFlow(ctx context.Context, arg sqlc.ReportCashFlowParams) ([]sqlc.ReportCashFlowRow, error) {
	rows, err := r.q.ReportCashFlow(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("tổng hợp dòng tiền thất bại: %w", err)
	}
	return rows, nil
}
