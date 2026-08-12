package initialize

import (
	"context"
	"fmt"

	"financal_management/global"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// InitRedis tạo client Redis và kiểm tra kết nối.
//
// Redis được dùng cho: lưu refresh token (để revoke được), rate limit,
// và cache kết quả báo cáo ở các phase sau.
//
// Trả về client cho nơi gọi thay vì gán vào biến toàn cục — xem lý do ở
// package global.
func InitRedis(ctx context.Context) (*redis.Client, error) {
	cfg := global.Config.Redis

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis thất bại (%s): %w", cfg.Addr(), err)
	}

	global.Logger.Info("đã kết nối Redis",
		zap.String("addr", cfg.Addr()),
		zap.Int("db", cfg.DB),
	)

	return client, nil
}
