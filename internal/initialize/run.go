package initialize

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"financal_management/global"

	"go.uber.org/zap"
)

// Run khởi động API server và chặn cho tới khi nhận tín hiệu dừng.
//
// Trả về lỗi thay vì panic, để cmd/api/main.go quyết định exit code.
func Run() error {
	// Context này bị huỷ khi nhận SIGINT (Ctrl+C) hoặc SIGTERM (docker stop,
	// kubernetes gửi khi xoá pod).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := LoadConfig(); err != nil {
		return err
	}
	if err := InitLogger(); err != nil {
		return err
	}
	// Đảm bảo log còn nằm trong buffer được ghi xuống đĩa trước khi thoát.
	defer func() { _ = global.Logger.Sync() }()

	if err := InitPostgres(ctx); err != nil {
		return err
	}
	defer global.Postgres.Close()

	if err := InitRedis(ctx); err != nil {
		return err
	}
	defer func() { _ = global.Redis.Close() }()

	cfg := global.Config.Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      InitRouter(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Chạy server ở goroutine riêng để goroutine chính còn chờ tín hiệu dừng.
	serverErr := make(chan error, 1)
	go func() {
		global.Logger.Info("API server đang lắng nghe",
			zap.Int("port", cfg.Port),
			zap.String("mode", cfg.Mode),
		)
		// ErrServerClosed là kết quả bình thường của Shutdown, không phải lỗi.
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("http server lỗi: %w", err)
	case <-ctx.Done():
		global.Logger.Info("nhận tín hiệu dừng, đang tắt server")
	}

	// Ngừng nhận kết nối mới nhưng chờ các request đang xử lý hoàn tất.
	// Dùng context.Background() vì ctx đã bị huỷ ở trên.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("tắt server không sạch sau %s: %w", cfg.ShutdownTimeout, err)
	}

	// Sau khi return, các defer bên trên chạy theo thứ tự ngược: đóng
	// Redis, đóng pool Postgres, cuối cùng flush log xuống đĩa.
	global.Logger.Info("server đã dừng", zap.Duration("shutdownTimeout", cfg.ShutdownTimeout))

	return nil
}
