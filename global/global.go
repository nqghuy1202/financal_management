// Package global chứa các đối tượng dùng chung toàn ứng dụng, được khởi
// tạo một lần duy nhất ở tầng initialize.
//
// Lưu ý: đây là biến toàn cục nên chỉ được ghi trong quá trình khởi động,
// sau đó chỉ đọc. Các tầng service và repo nên nhận dependency qua
// constructor thay vì đọc thẳng biến ở đây, để còn viết test được.
package global

import (
	"financal_management/internal/pkg/setting"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	// Config là cấu hình đã được nạp và validate.
	Config setting.Config

	// Logger là logger dùng chung. Trước khi InitLogger chạy xong, biến này
	// là nil — không log gì trước thời điểm đó.
	Logger *zap.Logger

	// Postgres là connection pool tới PostgreSQL.
	Postgres *pgxpool.Pool

	// Redis là client tới Redis, dùng cho cache, rate limit và refresh token.
	Redis *redis.Client
)
