package initialize

import (
	"fmt"
	"os"
	"strings"

	"financal_management/global"

	"github.com/spf13/viper"
)

// LoadConfig đọc file cấu hình tương ứng với môi trường rồi nạp vào
// global.Config.
//
// Thứ tự ưu tiên (cao đè thấp):
//  1. Biến môi trường có tiền tố FM_ (ví dụ FM_POSTGRES_PASSWORD)
//  2. File config/<APP_ENV>.yaml
//
// APP_ENV mặc định là "local".
func LoadConfig() error {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	v := viper.New()
	v.AddConfigPath("./config/")
	v.SetConfigName(env)
	v.SetConfigType("yaml")

	// Cho phép override bằng env: FM_POSTGRES_PASSWORD -> postgres.password
	v.SetEnvPrefix("FM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("đọc file cấu hình %s.yaml thất bại: %w", env, err)
	}

	// AutomaticEnv của viper không tự nhận diện được các key lồng nhau khi
	// Unmarshal, nên phải BindEnv tường minh cho từng key có trong file.
	for _, key := range v.AllKeys() {
		if err := v.BindEnv(key); err != nil {
			return fmt.Errorf("bind biến môi trường cho key %q thất bại: %w", key, err)
		}
	}

	if err := v.Unmarshal(&global.Config); err != nil {
		return fmt.Errorf("giải mã cấu hình thất bại: %w", err)
	}

	if err := global.Config.Validate(); err != nil {
		return fmt.Errorf("cấu hình không hợp lệ:\n%w", err)
	}

	return nil
}
