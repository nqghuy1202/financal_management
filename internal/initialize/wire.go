package initialize

import (
	"financal_management/global"
	"financal_management/internal/controller"
	"financal_management/internal/middlewares"
	"financal_management/internal/pkg/token"
	"financal_management/internal/repo"
	"financal_management/internal/routers"
	"financal_management/internal/services"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// buildDeps lắp chuỗi repo → service → controller và trả về mọi thứ tầng
// route cần.
//
// Đây là composition root: nơi duy nhất trong ứng dụng biết cài đặt cụ
// thể nào được nối vào interface nào. Mọi tầng khác chỉ làm việc với
// interface, nhờ vậy thay cài đặt (ví dụ đổi repo thật sang repo giả
// trong test) không phải sửa gì ngoài file này.
func buildDeps(db *pgxpool.Pool, rdb *redis.Client) routers.Deps {
	tokenManager := token.NewManager(global.Config.JWT, rdb)

	// repo → service → controller
	userRepo := repo.NewUserRepo(db)
	authService := services.NewAuthService(userRepo, tokenManager)

	accountRepo := repo.NewAccountRepo(db)
	accountService := services.NewAccountService(accountRepo)

	categoryRepo := repo.NewCategoryRepo(db)
	categoryService := services.NewCategoryService(categoryRepo)

	return routers.Deps{
		Health:   controller.NewHealthController(db, rdb),
		Auth:     controller.NewAuthController(authService),
		Account:  controller.NewAccountController(accountService),
		Category: controller.NewCategoryController(categoryService),

		RequireAuth: middlewares.RequireAuth(tokenManager),
		LoginRateLimit: middlewares.RateLimit(
			rdb, "login", global.Config.RateLimit.Login, middlewares.KeyByIP,
		),
		UserRateLimit: middlewares.RateLimit(
			rdb, "user", global.Config.RateLimit.User, middlewares.KeyByUser,
		),
	}
}
