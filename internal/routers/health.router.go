package routers

import (
	"financal_management/global"
	"financal_management/internal/controller"

	"github.com/gin-gonic/gin"
)

// registerHealthRoutes gắn các endpoint kiểm tra tình trạng hệ thống.
//
// Đặt ngoài /api/v1 vì đây là endpoint vận hành, không phải API nghiệp
// vụ — chúng không bao giờ đổi phiên bản.
func registerHealthRoutes(r *gin.Engine) {
	hc := controller.NewHealthController(global.Postgres, global.Redis)

	r.GET("/healthz", hc.Live)
	r.GET("/readyz", hc.Ready)
}
