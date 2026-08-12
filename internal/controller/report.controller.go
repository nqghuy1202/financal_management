package controller

import (
	"time"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ReportController struct {
	reports *services.ReportService
}

func NewReportController(reports *services.ReportService) *ReportController {
	return &ReportController{reports: reports}
}

// ---------------------------------------------------------------------
// Kiểu dữ liệu trả về
// ---------------------------------------------------------------------

// Mọi số tiền trả về dưới dạng chuỗi, cùng lý do với model.Money:
// JavaScript đọc số JSON thành float64 và làm tròn sai với số lớn.

type summaryResponse struct {
	From             time.Time `json:"from"`
	To               time.Time `json:"to"`
	Currency         string    `json:"currency"`
	TotalIncome      string    `json:"total_income"`
	TotalExpense     string    `json:"total_expense"`
	NetBalance       string    `json:"net_balance"`
	SavingsRatePct   string    `json:"savings_rate_pct"`
	TransactionCount int64     `json:"transaction_count"`

	Previous previousPeriodResponse `json:"previous_period"`
}

type previousPeriodResponse struct {
	TotalIncome  string `json:"total_income"`
	TotalExpense string `json:"total_expense"`
	NetBalance   string `json:"net_balance"`
	// null khi kỳ trước không có số liệu để so sánh.
	IncomeChangePct  *string `json:"income_change_pct"`
	ExpenseChangePct *string `json:"expense_change_pct"`
}

type categoryTotalResponse struct {
	CategoryID    *uuid.UUID `json:"category_id"`
	CategoryName  string     `json:"category_name"`
	CategoryIcon  string     `json:"category_icon"`
	CategoryColor string     `json:"category_color"`
	TotalAmount   string     `json:"total_amount"`
	Percentage    string     `json:"percentage"`
	Count         int64      `json:"transaction_count"`
}

type monthlyFlowResponse struct {
	// Dạng "2026-08" cho tiện vẽ biểu đồ.
	Month   string `json:"month"`
	Income  string `json:"income"`
	Expense string `json:"expense"`
	Net     string `json:"net"`
}

// ---------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------

// Summary trả về tổng quan tài chính của một kỳ, kèm so sánh kỳ trước.
// GET /api/v1/reports/summary?from=2026-08-01&to=2026-09-01
func (rc *ReportController) Summary(c *gin.Context) {
	userID, period, ok := reportParams(c)
	if !ok {
		return
	}

	s, err := rc.reports.Summary(c.Request.Context(), userID, period)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, summaryResponse{
		From:             s.Period.From,
		To:               s.Period.To,
		Currency:         model.DefaultCurrency,
		TotalIncome:      s.TotalIncome.String(),
		TotalExpense:     s.TotalExpense.String(),
		NetBalance:       s.NetBalance.String(),
		SavingsRatePct:   s.SavingsRate.String(),
		TransactionCount: s.TransactionCount,
		Previous: previousPeriodResponse{
			TotalIncome:      s.PreviousIncome.String(),
			TotalExpense:     s.PreviousExpense.String(),
			NetBalance:       s.PreviousNet.String(),
			IncomeChangePct:  decimalPtrToString(s.IncomeChangePct),
			ExpenseChangePct: decimalPtrToString(s.ExpenseChangePct),
		},
	})
}

// ByCategory trả về chi tiêu (hoặc thu nhập) theo từng danh mục.
// GET /api/v1/reports/by-category?from=&to=&type=expense
func (rc *ReportController) ByCategory(c *gin.Context) {
	userID, period, ok := reportParams(c)
	if !ok {
		return
	}

	// Mặc định xem chi tiêu, vì đó là thứ người dùng quan tâm nhất.
	txType := c.Query("type")
	if txType == "" {
		txType = model.TransactionTypeExpense
	}

	totals, err := rc.reports.ByCategory(c.Request.Context(), userID, period, txType)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]categoryTotalResponse, 0, len(totals))
	for _, t := range totals {
		items = append(items, categoryTotalResponse{
			CategoryID:    t.CategoryID,
			CategoryName:  t.CategoryName,
			CategoryIcon:  t.CategoryIcon,
			CategoryColor: t.CategoryColor,
			TotalAmount:   t.TotalAmount.String(),
			Percentage:    t.Percentage.String(),
			Count:         t.Count,
		})
	}

	response.Success(c, gin.H{
		keyItems:   items,
		"type":     txType,
		"currency": model.DefaultCurrency,
	})
}

// CashFlow trả về thu chi từng tháng trong kỳ.
// GET /api/v1/reports/cash-flow?from=2026-01-01&to=2027-01-01
func (rc *ReportController) CashFlow(c *gin.Context) {
	userID, period, ok := reportParams(c)
	if !ok {
		return
	}

	flows, err := rc.reports.CashFlow(c.Request.Context(), userID, period)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]monthlyFlowResponse, 0, len(flows))
	for _, f := range flows {
		items = append(items, monthlyFlowResponse{
			Month:   f.Month.Format("2006-01"),
			Income:  f.Income.String(),
			Expense: f.Expense.String(),
			Net:     f.Net.String(),
		})
	}

	response.Success(c, gin.H{
		keyItems:   items,
		"currency": model.DefaultCurrency,
	})
}

// ---------------------------------------------------------------------
// Hàm phụ trợ
// ---------------------------------------------------------------------

// reportParams đọc user id và khoảng thời gian dùng chung cho mọi báo cáo.
func reportParams(c *gin.Context) (uuid.UUID, services.Period, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return uuid.Nil, services.Period{}, false
	}

	from, err := optionalTimeQuery(c, "from")
	if err != nil {
		_ = c.Error(err)
		return uuid.Nil, services.Period{}, false
	}
	to, err := optionalTimeQuery(c, "to")
	if err != nil {
		_ = c.Error(err)
		return uuid.Nil, services.Period{}, false
	}
	if from == nil || to == nil {
		_ = c.Error(response.Newf(response.CodeInvalidParam,
			"Phải chỉ định cả hai tham số from và to"))
		return uuid.Nil, services.Period{}, false
	}

	return userID, services.Period{From: *from, To: *to}, true
}

func decimalPtrToString(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.String()
	return &s
}
