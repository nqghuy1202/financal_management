// Package routers khai báo các route của ứng dụng, nhóm theo module
// nghiệp vụ.
//
// Phân chia trách nhiệm:
//   - internal/initialize/router.go dựng gin.Engine và gắn middleware
//     toàn cục.
//   - package này chỉ khai báo đường dẫn và nối vào controller tương ứng.
//
// Nhờ tách như vậy, thêm một module mới chỉ cần thêm một file router
// trong package này và một dòng gọi trong RegisterRoutes.
package routers

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes gắn toàn bộ route của ứng dụng vào engine.
func RegisterRoutes(r *gin.Engine) {
	registerHealthRoutes(r)

	// API nghiệp vụ đều nằm dưới /api/v1 để về sau còn ra được v2 mà
	// không phá vỡ client cũ.
	v1 := r.Group("/api/v1")

	// Các module sẽ được thêm ở các phase sau:
	//   registerAuthRoutes(v1)          — Phase 1
	//   registerAccountRoutes(v1)       — Phase 2
	//   registerCategoryRoutes(v1)      — Phase 2
	//   registerTransactionRoutes(v1)   — Phase 2
	//   registerReportRoutes(v1)        — Phase 4
	//   registerBudgetRoutes(v1)        — Phase 5
	//   registerNotificationRoutes(v1)  — Phase 5
	_ = v1
}
