// Package routers khai báo các route của ứng dụng, nhóm theo module
// nghiệp vụ.
//
// Phân chia trách nhiệm:
//   - internal/initialize dựng gin.Engine, khởi tạo dependency và lắp
//     chúng vào Deps.
//   - package này chỉ khai báo đường dẫn và nối vào controller tương ứng.
//
// Deps được khai báo ở đây chứ không phải ở initialize, để tránh phụ
// thuộc vòng: initialize import routers, còn routers thì không import
// ngược lại.
package routers

import (
	"financal_management/internal/controller"

	"github.com/gin-gonic/gin"
)

// Deps gom toàn bộ controller mà tầng route cần.
//
// Truyền tường minh thay vì đọc biến toàn cục, để test có thể dựng router
// với controller giả mà không cần database thật.
type Deps struct {
	Health *controller.HealthController
}

// RegisterRoutes gắn toàn bộ route của ứng dụng vào engine.
func RegisterRoutes(r *gin.Engine, d Deps) {
	registerHealthRoutes(r, d.Health)

	// API nghiệp vụ đều nằm dưới /api/v1 để về sau còn ra được v2 mà
	// không phá vỡ client cũ.
	v1 := r.Group("/api/v1")

	// Các module sẽ được thêm ở các phase sau:
	//   registerAuthRoutes(v1, d.Auth)              — Phase 1
	//   registerAccountRoutes(v1, d.Account)        — Phase 2
	//   registerCategoryRoutes(v1, d.Category)      — Phase 2
	//   registerTransactionRoutes(v1, d.Transaction)— Phase 2
	//   registerReportRoutes(v1, d.Report)          — Phase 4
	//   registerBudgetRoutes(v1, d.Budget)          — Phase 5
	_ = v1
}
