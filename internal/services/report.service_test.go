package services

import (
	"context"
	"testing"
	"time"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------
// Repo giả
// ---------------------------------------------------------------------

// fakeReportRepo trả về số liệu đặt sẵn theo từng kỳ, để test tập trung
// vào phần TÍNH TOÁN của service chứ không phải phần SQL.
type fakeReportRepo struct {
	summaries  map[string]sqlc.ReportSummaryRow
	byCategory []sqlc.ReportByCategoryRow
	cashFlow   []sqlc.ReportCashFlowRow
	forceErr   error
}

func newFakeReportRepo() *fakeReportRepo {
	return &fakeReportRepo{summaries: make(map[string]sqlc.ReportSummaryRow)}
}

func periodKey(from, to time.Time) string {
	return from.Format(time.RFC3339) + "|" + to.Format(time.RFC3339)
}

func (f *fakeReportRepo) setSummary(from, to time.Time, income, expense string, count int64) {
	f.summaries[periodKey(from, to)] = sqlc.ReportSummaryRow{
		TotalIncome:      decimal.RequireFromString(income),
		TotalExpense:     decimal.RequireFromString(expense),
		TransactionCount: count,
	}
}

func (f *fakeReportRepo) Summary(_ context.Context, arg sqlc.ReportSummaryParams) (sqlc.ReportSummaryRow, error) {
	if f.forceErr != nil {
		return sqlc.ReportSummaryRow{}, f.forceErr
	}
	row, ok := f.summaries[periodKey(arg.FromTime, arg.ToTime)]
	if !ok {
		// Kỳ không có dữ liệu: SQL thật trả về 0 nhờ COALESCE.
		return sqlc.ReportSummaryRow{
			TotalIncome: decimal.Zero, TotalExpense: decimal.Zero,
		}, nil
	}
	return row, nil
}

func (f *fakeReportRepo) ByCategory(_ context.Context, _ sqlc.ReportByCategoryParams) ([]sqlc.ReportByCategoryRow, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	return f.byCategory, nil
}

func (f *fakeReportRepo) CashFlow(_ context.Context, _ sqlc.ReportCashFlowParams) ([]sqlc.ReportCashFlowRow, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	return f.cashFlow, nil
}

// ---------------------------------------------------------------------
// Hàm dựng test
// ---------------------------------------------------------------------

func newReportService() (*ReportService, *fakeReportRepo) {
	r := newFakeReportRepo()
	return NewReportService(r), r
}

// thang trả về kỳ của một tháng: [ngày 1 tháng đó, ngày 1 tháng sau).
func thang(year int, month time.Month) Period {
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	return Period{From: from, To: from.AddDate(0, 1, 0)}
}

// ---------------------------------------------------------------------
// Period
// ---------------------------------------------------------------------

// Kỳ liền trước phải có cùng độ dài và nối liền, không chồng lấn.
func TestPeriod_Previous(t *testing.T) {
	p := thang(2026, time.August)
	prev := p.Previous()

	if !prev.To.Equal(p.From) {
		t.Errorf("kỳ trước phải kết thúc đúng lúc kỳ này bắt đầu: %v vs %v", prev.To, p.From)
	}
	if prev.Duration() != p.Duration() {
		t.Errorf("hai kỳ phải cùng độ dài: %v vs %v", prev.Duration(), p.Duration())
	}
	if want := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC); !prev.From.Equal(want) {
		t.Errorf("kỳ trước bắt đầu %v, mong đợi %v", prev.From, want)
	}
}

// ---------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------

func TestSummary_TinhDungSoDuVaTyLeTietKiem(t *testing.T) {
	svc, fake := newReportService()
	p := thang(2026, time.August)
	// Thu 20 triệu, chi 15 triệu → dư 5 triệu, tiết kiệm 25%.
	fake.setSummary(p.From, p.To, "20000000", "15000000", 12)

	s, err := svc.Summary(context.Background(), uuid.New(), p)
	if err != nil {
		t.Fatalf("Summary lỗi: %v", err)
	}

	if got := s.NetBalance.String(); got != "5000000" {
		t.Errorf("NetBalance = %s, mong đợi 5000000", got)
	}
	if got := s.SavingsRate.String(); got != "25" {
		t.Errorf("SavingsRate = %s, mong đợi 25", got)
	}
	if s.TransactionCount != 12 {
		t.Errorf("TransactionCount = %d, mong đợi 12", s.TransactionCount)
	}
}

// Chi nhiều hơn thu thì số dư âm và tỷ lệ tiết kiệm âm — đó là thông tin
// đúng và hữu ích, không được cắt về 0.
func TestSummary_ChiNhieuHonThu(t *testing.T) {
	svc, fake := newReportService()
	p := thang(2026, time.August)
	fake.setSummary(p.From, p.To, "10000000", "12000000", 5)

	s, err := svc.Summary(context.Background(), uuid.New(), p)
	if err != nil {
		t.Fatalf("Summary lỗi: %v", err)
	}

	if !s.NetBalance.IsNegative() {
		t.Errorf("NetBalance = %s, phải là số âm", s.NetBalance)
	}
	if got := s.SavingsRate.String(); got != "-20" {
		t.Errorf("SavingsRate = %s, mong đợi -20", got)
	}
}

// Không có thu nhập thì tỷ lệ tiết kiệm không tính được — phải trả 0 chứ
// không được chia cho 0 và làm sập chương trình.
func TestSummary_KhongCoThuNhap(t *testing.T) {
	svc, fake := newReportService()
	p := thang(2026, time.August)
	fake.setSummary(p.From, p.To, "0", "500000", 3)

	s, err := svc.Summary(context.Background(), uuid.New(), p)
	if err != nil {
		t.Fatalf("Summary lỗi: %v", err)
	}
	if !s.SavingsRate.IsZero() {
		t.Errorf("SavingsRate = %s, mong đợi 0 khi không có thu nhập", s.SavingsRate)
	}
}

func TestSummary_SoSanhKyTruoc(t *testing.T) {
	svc, fake := newReportService()
	p := thang(2026, time.August)
	prev := p.Previous()

	fake.setSummary(p.From, p.To, "20000000", "12000000", 10)
	fake.setSummary(prev.From, prev.To, "20000000", "10000000", 8)

	s, err := svc.Summary(context.Background(), uuid.New(), p)
	if err != nil {
		t.Fatalf("Summary lỗi: %v", err)
	}

	// Chi tăng từ 10 lên 12 triệu = tăng 20%.
	if s.ExpenseChangePct == nil {
		t.Fatal("phải có tỷ lệ thay đổi chi tiêu")
	}
	if got := s.ExpenseChangePct.String(); got != "20" {
		t.Errorf("ExpenseChangePct = %s, mong đợi 20", got)
	}
	// Thu không đổi = 0%.
	if got := s.IncomeChangePct.String(); got != "0" {
		t.Errorf("IncomeChangePct = %s, mong đợi 0", got)
	}
}

// Kỳ trước bằng 0 thì phần trăm thay đổi là vô nghĩa (tăng vô hạn lần),
// phải trả về null để frontend hiện dấu gạch thay vì một con số vô lý.
func TestSummary_KyTruocBangKhongThiKhongCoPhanTram(t *testing.T) {
	svc, fake := newReportService()
	p := thang(2026, time.August)
	fake.setSummary(p.From, p.To, "5000000", "3000000", 4)
	// Kỳ trước không đặt gì → repo giả trả về 0.

	s, err := svc.Summary(context.Background(), uuid.New(), p)
	if err != nil {
		t.Fatalf("Summary lỗi: %v", err)
	}
	if s.IncomeChangePct != nil {
		t.Errorf("kỳ trước bằng 0 thì IncomeChangePct phải là nil, đang là %s", s.IncomeChangePct)
	}
	if s.ExpenseChangePct != nil {
		t.Errorf("kỳ trước bằng 0 thì ExpenseChangePct phải là nil, đang là %s", s.ExpenseChangePct)
	}
}

func TestSummary_KhoangThoiGianKhongHopLe(t *testing.T) {
	svc, _ := newReportService()
	ctx := context.Background()
	userID := uuid.New()

	t.Run("thiếu tham số", func(t *testing.T) {
		_, err := svc.Summary(ctx, userID, Period{})
		wantCode(t, err, response.CodeValidationFailed)
	})

	t.Run("kết thúc trước khi bắt đầu", func(t *testing.T) {
		now := time.Now()
		_, err := svc.Summary(ctx, userID, Period{From: now, To: now.AddDate(0, 0, -1)})
		wantCode(t, err, response.CodeValidationFailed)
	})

	t.Run("hai mốc trùng nhau", func(t *testing.T) {
		now := time.Now()
		_, err := svc.Summary(ctx, userID, Period{From: now, To: now})
		wantCode(t, err, response.CodeValidationFailed)
	})
}

// ---------------------------------------------------------------------
// ByCategory
// ---------------------------------------------------------------------

func TestByCategory_TinhDungTyTrong(t *testing.T) {
	svc, fake := newReportService()
	catA, catB := uuid.New(), uuid.New()
	fake.byCategory = []sqlc.ReportByCategoryRow{
		{CategoryID: &catA, CategoryName: "Ăn uống", TotalAmount: decimal.RequireFromString("750000"), TransactionCount: 10},
		{CategoryID: &catB, CategoryName: "Đi lại", TotalAmount: decimal.RequireFromString("250000"), TransactionCount: 4},
	}

	items, err := svc.ByCategory(context.Background(), uuid.New(),
		thang(2026, time.August), model.TransactionTypeExpense)
	if err != nil {
		t.Fatalf("ByCategory lỗi: %v", err)
	}

	if got := items[0].Percentage.String(); got != "75" {
		t.Errorf("tỷ trọng danh mục đầu = %s, mong đợi 75", got)
	}
	if got := items[1].Percentage.String(); got != "25" {
		t.Errorf("tỷ trọng danh mục sau = %s, mong đợi 25", got)
	}

	// Tổng các tỷ trọng phải bằng 100.
	total := items[0].Percentage.Add(items[1].Percentage)
	if got := total.String(); got != "100" {
		t.Errorf("tổng tỷ trọng = %s, mong đợi 100", got)
	}
}

// Giao dịch ghi nhanh không có danh mục vẫn phải hiện trong báo cáo, gom
// vào nhóm "Chưa phân loại". Nếu bỏ sót, tổng báo cáo sẽ nhỏ hơn tổng chi
// thực tế mà người dùng không hiểu vì sao.
func TestByCategory_GomKhoanChuaPhanLoai(t *testing.T) {
	svc, fake := newReportService()
	cat := uuid.New()
	fake.byCategory = []sqlc.ReportByCategoryRow{
		{CategoryID: &cat, CategoryName: "Ăn uống", TotalAmount: decimal.RequireFromString("800000"), TransactionCount: 8},
		{CategoryID: nil, CategoryName: "", TotalAmount: decimal.RequireFromString("200000"), TransactionCount: 2},
	}

	items, err := svc.ByCategory(context.Background(), uuid.New(),
		thang(2026, time.August), model.TransactionTypeExpense)
	if err != nil {
		t.Fatalf("ByCategory lỗi: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("có %d dòng, mong đợi 2", len(items))
	}
	if items[1].CategoryName != "Chưa phân loại" {
		t.Errorf("dòng không có danh mục tên %q, mong đợi %q",
			items[1].CategoryName, "Chưa phân loại")
	}
}

func TestByCategory_KhongCoDuLieu(t *testing.T) {
	svc, _ := newReportService()

	items, err := svc.ByCategory(context.Background(), uuid.New(),
		thang(2026, time.August), model.TransactionTypeExpense)
	if err != nil {
		t.Fatalf("ByCategory lỗi: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("có %d dòng, mong đợi 0", len(items))
	}
}

func TestByCategory_LoaiKhongHopLe(t *testing.T) {
	svc, _ := newReportService()
	_, err := svc.ByCategory(context.Background(), uuid.New(),
		thang(2026, time.August), "transfer")
	wantCode(t, err, response.CodeValidationFailed)
}

// ---------------------------------------------------------------------
// CashFlow
// ---------------------------------------------------------------------

// Tháng không có giao dịch vẫn phải xuất hiện với giá trị 0, nếu không
// biểu đồ sẽ nhảy cóc và trông như tháng đó không tồn tại.
func TestCashFlow_DienThangTrong(t *testing.T) {
	svc, fake := newReportService()

	// Chỉ tháng 1 và tháng 3 có dữ liệu.
	fake.cashFlow = []sqlc.ReportCashFlowRow{
		{
			Period:       time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
			TotalIncome:  decimal.RequireFromString("10000000"),
			TotalExpense: decimal.RequireFromString("6000000"),
		},
		{
			Period:       time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
			TotalIncome:  decimal.RequireFromString("11000000"),
			TotalExpense: decimal.RequireFromString("7000000"),
		},
	}

	// Dựng kỳ theo múi giờ ứng dụng, giống hệt cách controller dựng từ
	// tham số ?from=2026-01-01&to=2026-04-01.
	loc := model.AppLocation()
	p := Period{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, loc),
		To:   time.Date(2026, time.April, 1, 0, 0, 0, 0, loc),
	}

	flows, err := svc.CashFlow(context.Background(), uuid.New(), p)
	if err != nil {
		t.Fatalf("CashFlow lỗi: %v", err)
	}

	// Kỳ [01/01, 01/04) gồm đúng ba tháng: 1, 2, 3. Mốc 01/04 không
	// thuộc kỳ nên tháng 4 không được xuất hiện.
	if len(flows) != 3 {
		got := make([]string, 0, len(flows))
		for _, f := range flows {
			got = append(got, f.Month.Format("2006-01"))
		}
		t.Fatalf("có %d tháng (%v), mong đợi đúng 3", len(flows), got)
	}

	if got := flows[0].Net.String(); got != "4000000" {
		t.Errorf("số dư tháng 1 = %s, mong đợi 4000000", got)
	}
	// Tháng 2 không có giao dịch.
	if !flows[1].Income.IsZero() || !flows[1].Expense.IsZero() {
		t.Errorf("tháng 2 phải bằng 0, đang là thu=%s chi=%s", flows[1].Income, flows[1].Expense)
	}
	if got := flows[2].Net.String(); got != "4000000" {
		t.Errorf("số dư tháng 3 = %s, mong đợi 4000000", got)
	}

	// Các tháng phải liên tiếp, không nhảy cóc.
	for i := 1; i < len(flows); i++ {
		want := flows[i-1].Month.AddDate(0, 1, 0)
		if !flows[i].Month.Equal(want) {
			t.Errorf("tháng thứ %d là %v, mong đợi %v (phải liên tiếp)",
				i, flows[i].Month, want)
		}
	}
}

// Kỳ không kết thúc đúng đầu tháng thì tháng chứa mốc kết thúc vẫn phải
// có mặt, vì một phần của nó nằm trong kỳ.
func TestCashFlow_KyKetThucGiuaThang(t *testing.T) {
	svc, _ := newReportService()

	loc := model.AppLocation()
	p := Period{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, loc),
		To:   time.Date(2026, time.March, 15, 0, 0, 0, 0, loc),
	}

	flows, err := svc.CashFlow(context.Background(), uuid.New(), p)
	if err != nil {
		t.Fatalf("CashFlow lỗi: %v", err)
	}
	if len(flows) != 3 {
		got := make([]string, 0, len(flows))
		for _, f := range flows {
			got = append(got, f.Month.Format("2006-01"))
		}
		t.Fatalf("có %d tháng (%v), mong đợi 3 (tháng 1, 2, 3)", len(flows), got)
	}
}

func TestCashFlow_KhoangThoiGianKhongHopLe(t *testing.T) {
	svc, _ := newReportService()
	_, err := svc.CashFlow(context.Background(), uuid.New(), Period{})
	wantCode(t, err, response.CodeValidationFailed)
}
