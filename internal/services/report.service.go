package services

import (
	"context"
	"time"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo"
	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReportService tổng hợp số liệu thành bức tranh tài chính.
type ReportService struct {
	reports repo.ReportRepo
}

func NewReportService(reports repo.ReportRepo) *ReportService {
	return &ReportService{reports: reports}
}

// Period là khoảng thời gian nửa mở [From, To).
//
// Nửa mở để hai kỳ liền nhau không đếm trùng giao dịch nằm đúng ranh giới:
// kỳ tháng 8 là [01/08, 01/09), kỳ tháng 9 là [01/09, 01/10).
type Period struct {
	From time.Time
	To   time.Time
}

// Duration là độ dài của kỳ, dùng để suy ra kỳ liền trước.
func (p Period) Duration() time.Duration { return p.To.Sub(p.From) }

// Previous trả về kỳ liền trước có cùng độ dài.
func (p Period) Previous() Period {
	d := p.Duration()
	return Period{From: p.From.Add(-d), To: p.From}
}

// Summary là bức tranh tổng quan của một kỳ, kèm so sánh với kỳ trước.
type Summary struct {
	Period           Period
	TotalIncome      decimal.Decimal
	TotalExpense     decimal.Decimal
	NetBalance       decimal.Decimal
	SavingsRate      decimal.Decimal // phần trăm, 0-100
	TransactionCount int64

	PreviousIncome  decimal.Decimal
	PreviousExpense decimal.Decimal
	PreviousNet     decimal.Decimal

	// Thay đổi so với kỳ trước, tính bằng phần trăm. nil khi kỳ trước
	// bằng 0 — khi đó tỷ lệ phần trăm không có nghĩa.
	IncomeChangePct  *decimal.Decimal
	ExpenseChangePct *decimal.Decimal
}

// CategoryTotal là một dòng trong báo cáo chi tiêu theo danh mục.
type CategoryTotal struct {
	CategoryID    *uuid.UUID
	CategoryName  string
	CategoryIcon  string
	CategoryColor string
	TotalAmount   decimal.Decimal
	Percentage    decimal.Decimal // phần trăm trên tổng của kỳ
	Count         int64
}

// MonthlyFlow là thu chi của một tháng.
type MonthlyFlow struct {
	Month   time.Time
	Income  decimal.Decimal
	Expense decimal.Decimal
	Net     decimal.Decimal
}

var hundred = decimal.NewFromInt(100)

// Summary trả về tổng quan của kỳ, kèm so sánh với kỳ liền trước.
func (s *ReportService) Summary(ctx context.Context, userID uuid.UUID, p Period) (*Summary, error) {
	if err := validatePeriod(p); err != nil {
		return nil, err
	}

	current, err := s.reports.Summary(ctx, sqlc.ReportSummaryParams{
		UserID: userID, FromTime: p.From, ToTime: p.To,
	})
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}

	prevPeriod := p.Previous()
	previous, err := s.reports.Summary(ctx, sqlc.ReportSummaryParams{
		UserID: userID, FromTime: prevPeriod.From, ToTime: prevPeriod.To,
	})
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}

	out := &Summary{
		Period:           p,
		TotalIncome:      current.TotalIncome,
		TotalExpense:     current.TotalExpense,
		NetBalance:       current.TotalIncome.Sub(current.TotalExpense),
		TransactionCount: current.TransactionCount,
		PreviousIncome:   previous.TotalIncome,
		PreviousExpense:  previous.TotalExpense,
		PreviousNet:      previous.TotalIncome.Sub(previous.TotalExpense),
	}

	// Tỷ lệ tiết kiệm = (thu - chi) / thu. Không có thu nhập thì tỷ lệ
	// này không có nghĩa, để 0.
	if current.TotalIncome.IsPositive() {
		out.SavingsRate = out.NetBalance.Div(current.TotalIncome).Mul(hundred).Round(2)
	}

	out.IncomeChangePct = percentChange(previous.TotalIncome, current.TotalIncome)
	out.ExpenseChangePct = percentChange(previous.TotalExpense, current.TotalExpense)

	return out, nil
}

// ByCategory trả về tổng tiền theo từng danh mục, kèm tỷ trọng.
func (s *ReportService) ByCategory(
	ctx context.Context, userID uuid.UUID, p Period, txType string,
) ([]CategoryTotal, error) {
	if err := validatePeriod(p); err != nil {
		return nil, err
	}
	if txType != model.TransactionTypeIncome && txType != model.TransactionTypeExpense {
		return nil, response.Newf(response.CodeValidationFailed,
			"Loại giao dịch không hợp lệ: %s", txType)
	}

	rows, err := s.reports.ByCategory(ctx, sqlc.ReportByCategoryParams{
		UserID: userID, Type: txType, FromTime: p.From, ToTime: p.To,
	})
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}

	// Tổng của cả kỳ, dùng làm mẫu số khi tính tỷ trọng.
	total := decimal.Zero
	for _, r := range rows {
		total = total.Add(r.TotalAmount)
	}

	out := make([]CategoryTotal, 0, len(rows))
	for _, r := range rows {
		item := CategoryTotal{
			CategoryID:    r.CategoryID,
			CategoryName:  r.CategoryName,
			CategoryIcon:  r.CategoryIcon,
			CategoryColor: r.CategoryColor,
			TotalAmount:   r.TotalAmount,
			Count:         r.TransactionCount,
		}
		// Giao dịch ghi nhanh không có danh mục.
		if r.CategoryID == nil {
			item.CategoryName = "Chưa phân loại"
		}
		if total.IsPositive() {
			item.Percentage = r.TotalAmount.Div(total).Mul(hundred).Round(2)
		}
		out = append(out, item)
	}

	return out, nil
}

// CashFlow trả về thu chi từng tháng trong kỳ.
//
// Tháng không có giao dịch nào vẫn xuất hiện với giá trị 0, để biểu đồ
// không bị đứt quãng và người dùng thấy đúng là tháng đó không chi gì.
func (s *ReportService) CashFlow(ctx context.Context, userID uuid.UUID, p Period) ([]MonthlyFlow, error) {
	if err := validatePeriod(p); err != nil {
		return nil, err
	}

	rows, err := s.reports.CashFlow(ctx, sqlc.ReportCashFlowParams{
		UserID: userID, Timezone: model.AppTimezone, FromTime: p.From, ToTime: p.To,
	})
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}

	// Đưa kết quả vào map để tra theo tháng.
	byMonth := make(map[time.Time]sqlc.ReportCashFlowRow, len(rows))
	for _, r := range rows {
		byMonth[r.Period] = r
	}

	loc := model.AppLocation()

	// Duyệt lần lượt từng tháng trong kỳ và điền 0 cho tháng không có dữ liệu.
	out := make([]MonthlyFlow, 0)
	fromLocal := p.From.In(loc)

	// Kỳ là nửa mở [From, To) nên mốc To KHÔNG thuộc kỳ. Lấy tháng của
	// thời điểm cuối cùng còn nằm trong kỳ, chứ không phải tháng của To.
	// Nếu lấy tháng của To thì kỳ [01/06, 01/10) sẽ thừa ra tháng 10 —
	// biểu đồ hiện một cột rỗng của tháng không được hỏi tới.
	lastLocal := p.To.Add(-time.Nanosecond).In(loc)

	cursor := time.Date(fromLocal.Year(), fromLocal.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(lastLocal.Year(), lastLocal.Month(), 1, 0, 0, 0, 0, time.UTC)

	for !cursor.After(end) {
		flow := MonthlyFlow{
			Month:   cursor,
			Income:  decimal.Zero,
			Expense: decimal.Zero,
			Net:     decimal.Zero,
		}
		if r, ok := byMonth[cursor]; ok {
			flow.Income = r.TotalIncome
			flow.Expense = r.TotalExpense
			flow.Net = r.TotalIncome.Sub(r.TotalExpense)
		}
		out = append(out, flow)
		cursor = cursor.AddDate(0, 1, 0)
	}

	return out, nil
}

// validatePeriod kiểm tra khoảng thời gian hợp lệ.
func validatePeriod(p Period) error {
	if p.From.IsZero() || p.To.IsZero() {
		return response.Newf(response.CodeValidationFailed,
			"Phải chỉ định khoảng thời gian bằng tham số from và to")
	}
	if !p.To.After(p.From) {
		return response.Newf(response.CodeValidationFailed,
			"Thời điểm kết thúc phải sau thời điểm bắt đầu")
	}
	return nil
}

// percentChange tính phần trăm thay đổi từ old sang new.
//
// Trả về nil khi kỳ trước bằng 0: tăng từ 0 lên bất kỳ số nào cũng là
// vô hạn phần trăm, hiển thị con số đó không giúp ích gì cho người dùng.
func percentChange(old, current decimal.Decimal) *decimal.Decimal {
	if !old.IsPositive() {
		return nil
	}
	pct := current.Sub(old).Div(old).Mul(hundred).Round(2)
	return &pct
}
