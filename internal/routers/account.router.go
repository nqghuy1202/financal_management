package routers

import (
	"github.com/gin-gonic/gin"
)

// registerAccountRoutes gắn các endpoint quản lý ví.
//
// Toàn bộ đều yêu cầu đăng nhập: ví là dữ liệu riêng của từng người.
func registerAccountRoutes(rg *gin.RouterGroup, d Deps) {
	accounts := rg.Group("/accounts")
	accounts.Use(d.RequireAuth, d.UserRateLimit)
	{
		accounts.POST("", d.Account.Create)
		accounts.GET("", d.Account.List)
		accounts.GET("/:id", d.Account.Get)
		accounts.PUT("/:id", d.Account.Update)
		accounts.DELETE("/:id", d.Account.Delete)
	}
}
