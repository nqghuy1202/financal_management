package initialize

import (
	"financal_management/internal/controller"
	"financal_management/internal/routers"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// buildDeps lắp chuỗi repo → service → controller và trả về tập
// controller cho tầng route.
//
// Đây là composition root: nơi duy nhất trong ứng dụng biết cài đặt cụ
// thể nào được nối vào interface nào. Mọi tầng khác chỉ làm việc với
// interface, nhờ vậy thay cài đặt (ví dụ đổi repo thật sang repo giả
// trong test) không phải sửa gì ngoài file này.
//
// Khi số module tăng lên, cân nhắc chuyển sang google/wire để sinh code
// tự động; hiện tại viết tay vẫn dễ đọc hơn.
func buildDeps(db *pgxpool.Pool, rdb *redis.Client) routers.Deps {
	return routers.Deps{
		Health: controller.NewHealthController(db, rdb),
	}
}
