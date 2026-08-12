package middlewares

import (
	"time"

	"financal_management/global"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS cho phép frontend ở origin khác gọi API.
//
// Danh sách origin lấy từ cấu hình chứ không để "*", vì API dùng
// Authorization header và cookie — trình duyệt sẽ chặn wildcard khi
// AllowCredentials bật.
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     global.Config.Server.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
