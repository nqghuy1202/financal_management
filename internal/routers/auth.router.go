package routers

import (
	"github.com/gin-gonic/gin"
)

// registerAuthRoutes gắn các endpoint đăng ký, đăng nhập và quản lý phiên.
func registerAuthRoutes(rg *gin.RouterGroup, d Deps) {
	auth := rg.Group("/auth")

	// Endpoint công khai — chưa cần token.
	auth.POST("/register", d.Auth.Register)

	// Riêng /login gắn thêm hạn mức chặt (5 lần thử) để chống dò mật khẩu.
	// Hạn mức toàn cục 100 request là quá rộng cho việc này.
	auth.POST("/login", d.LoginRateLimit, d.Auth.Login)

	auth.POST("/refresh", d.Auth.Refresh)
	auth.POST("/logout", d.Auth.Logout)

	// Endpoint cần đăng nhập.
	//
	// Thứ tự middleware quan trọng: RequireAuth chạy TRƯỚC UserRateLimit,
	// vì hạn mức theo user cần biết người dùng là ai. Nếu đảo lại, mọi
	// request sẽ bị tính chung theo IP.
	me := rg.Group("/auth")
	me.Use(d.RequireAuth, d.UserRateLimit)
	{
		me.GET("/me", d.Auth.Me)
	}
}
