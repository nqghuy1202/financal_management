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

// Deps gom mọi thứ mà tầng route cần.
//
// Truyền tường minh thay vì đọc biến toàn cục, để test có thể dựng router
// với controller giả mà không cần database thật.
type Deps struct {
	Health  *controller.HealthController
	Auth    *controller.AuthController
	Account *controller.AccountController

	// Middleware được dựng sẵn ở initialize rồi truyền vào, vì chúng cần
	// dependency (Redis, token manager) mà package này không nên biết tới.
	RequireAuth    gin.HandlerFunc
	LoginRateLimit gin.HandlerFunc
	UserRateLimit  gin.HandlerFunc
}

// RegisterRoutes gắn toàn bộ route của ứng dụng vào engine.
func RegisterRoutes(r *gin.Engine, d Deps) {
	registerHealthRoutes(r, d.Health)

	// API nghiệp vụ đều nằm dưới /api/v1 để về sau còn ra được v2 mà
	// không phá vỡ client cũ.
	v1 := r.Group("/api/v1")

	registerAuthRoutes(v1, d)
	registerAccountRoutes(v1, d)

	// Các module sẽ được thêm ở các phase sau:
	//   registerCategoryRoutes(v1, d)       — Phase 2
	//   registerTransactionRoutes(v1, d)    — Phase 2
	//   registerReportRoutes(v1, d)         — Phase 4
	//   registerBudgetRoutes(v1, d)         — Phase 5
}
