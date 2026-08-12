package initialize

import (
	"financal_management/global"
	"financal_management/internal/pkg/logger"
)

// InitLogger dựng logger từ cấu hình và gán vào global.Logger.
func InitLogger() error {
	l, err := logger.New(global.Config.Logger)
	if err != nil {
		return err
	}
	global.Logger = l
	return nil
}
