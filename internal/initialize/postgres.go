package initialize

import (
	"context"
	"fmt"

	"financal_management/global"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// InitPostgres tạo connection pool tới PostgreSQL và kiểm tra kết nối
// bằng một lệnh ping.
//
// Dùng pgxpool thay vì database/sql vì pgx là driver native cho Postgres:
// nhanh hơn, hỗ trợ đầy đủ kiểu dữ liệu của Postgres (numeric, jsonb,
// array) và là driver mà sqlc sinh code cho ở các phase sau.
func InitPostgres(ctx context.Context) error {
	cfg := global.Config.Postgres

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return fmt.Errorf("phân tích DSN postgres thất bại: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("tạo pool postgres thất bại: %w", err)
	}

	// NewWithConfig là lazy — chưa thực sự mở kết nối nào. Ping ở đây để
	// ứng dụng fail fast ngay lúc khởi động nếu DB không truy cập được.
	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return fmt.Errorf("ping postgres thất bại (%s:%d): %w", cfg.Host, cfg.Port, err)
	}

	global.Postgres = pool
	global.Logger.Info("đã kết nối PostgreSQL",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.DBName),
		zap.Int32("maxConns", cfg.MaxConns),
	)

	return nil
}
