package initialize

import (
	"financal_management/global"
	"financal_management/internal/middlewares"
	"financal_management/internal/routers"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// InitRouter dựng gin.Engine, gắn middleware toàn cục rồi đăng ký route.
func InitRouter(rdb *redis.Client, deps routers.Deps) *gin.Engine {
	gin.SetMode(global.Config.Server.Mode)

	// gin.New() thay vì gin.Default(): Default gắn sẵn Logger và Recovery
	// mặc định của gin (ghi text ra stdout). Ở đây ta dùng bản riêng ghi
	// JSON qua zap, nên phải tự lắp từ engine trắng.
	r := gin.New()

	// Thứ tự middleware rất quan trọng, request đi từ trên xuống:
	//  1. RequestID  — sinh id trước, để mọi log phía sau đều có id
	//  2. Logger     — bọc ngoài cùng phần đo thời gian, ghi log khi xong
	//  3. Recovery   — bắt panic của các tầng bên trong
	//  4. ErrorHandler — dựng response từ lỗi handler đẩy vào c.Error()
	//  5. CORS       — trả preflight sớm
	//  6. RateLimit  — chặn trước khi chạm vào handler nghiệp vụ
	//
	// Hạn mức toàn cục khoá theo IP vì ở tầng này chưa biết người dùng là
	// ai. Hạn mức theo user được gắn riêng cho từng nhóm route đã xác
	// thực, bên trong package routers.
	r.Use(
		middlewares.RequestID(),
		middlewares.Logger(),
		middlewares.Recovery(),
		middlewares.ErrorHandler(),
		middlewares.CORS(),
		middlewares.RateLimit(rdb, "ip", global.Config.RateLimit.IP, middlewares.KeyByIP),
	)

	routers.RegisterRoutes(r, deps)

	return r
}
