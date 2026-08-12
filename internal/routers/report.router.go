package routers

import (
	"github.com/gin-gonic/gin"
)

// registerReportRoutes gắn các endpoint báo cáo tổng hợp.
//
// Đây là phần cốt lõi của ứng dụng: người dùng nhập thu chi để đổi lấy
// bức tranh tài chính, và ba endpoint dưới đây chính là bức tranh đó.
func registerReportRoutes(rg *gin.RouterGroup, d Deps) {
	reports := rg.Group("/reports")
	reports.Use(d.RequireAuth, d.UserRateLimit)
	{
		// Tổng quan: thu, chi, số dư ròng, tỷ lệ tiết kiệm, so sánh kỳ trước.
		reports.GET("/summary", d.Report.Summary)
		// Tiền đi đâu: tổng theo từng danh mục kèm tỷ trọng.
		reports.GET("/by-category", d.Report.ByCategory)
		// Biến động theo thời gian: thu chi từng tháng.
		reports.GET("/cash-flow", d.Report.CashFlow)
	}
}
