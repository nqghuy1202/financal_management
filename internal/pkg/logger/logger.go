// Package logger dựng logger zap dùng chung cho toàn ứng dụng.
//
// Log được ghi ra stdout dạng JSON để các hệ thống thu thập log (Loki,
// ELK, CloudWatch...) parse được, đồng thời ghi ra file có xoay vòng.
package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"financal_management/internal/pkg/setting"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New tạo logger từ cấu hình. Trả về lỗi thay vì panic để nơi gọi tự
// quyết định cách xử lý.
func New(cfg setting.LoggerSetting) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("logger.level %q không hợp lệ: %w", cfg.Level, err)
	}

	syncer, err := writeSyncer(cfg)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewCore(encoder(), syncer, level)

	// AddCaller: ghi file:line nơi gọi log.
	// AddStacktrace: chỉ đính kèm stacktrace từ mức error trở lên, tránh
	// làm log info phình to.
	return zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	), nil
}

// encoder định nghĩa định dạng của một dòng log.
func encoder() zapcore.Encoder {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "ts"
	cfg.LevelKey = "level"
	cfg.CallerKey = "caller"
	cfg.MessageKey = "msg"
	cfg.StacktraceKey = "stacktrace"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.EncodeDuration = zapcore.MillisDurationEncoder

	return zapcore.NewJSONEncoder(cfg)
}

// writeSyncer ghi đồng thời ra stdout và ra file (nếu có cấu hình file).
func writeSyncer(cfg setting.LoggerSetting) (zapcore.WriteSyncer, error) {
	syncers := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}

	if cfg.FileName != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.FileName), 0o755); err != nil {
			return nil, fmt.Errorf("tạo thư mục log thất bại: %w", err)
		}
		// lumberjack tự xoay vòng file khi vượt MaxSize, giữ lại MaxBackups
		// file cũ — tránh log ăn hết ổ đĩa trên server.
		syncers = append(syncers, zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.FileName,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}))
	}

	return zapcore.NewMultiWriteSyncer(syncers...), nil
}
