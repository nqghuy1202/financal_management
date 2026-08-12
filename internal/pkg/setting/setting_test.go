package setting

import (
	"strings"
	"testing"
	"time"
)

// validConfig trả về một cấu hình hợp lệ để mỗi test chỉ cần làm hỏng
// đúng một trường mà nó quan tâm.
func validConfig() Config {
	return Config{
		Server: ServerSetting{Mode: "debug", Port: 8080},
		Postgres: PostgresSetting{
			Host: "localhost", Port: 5432, User: "u", Password: "p",
			DBName: "fintrack", SSLMode: "disable",
		},
		Redis: RedisSetting{Host: "localhost", Port: 6379},
		Kafka: KafkaSetting{Brokers: []string{"localhost:9092"}},
		JWT: JWTSetting{
			Secret:          strings.Repeat("x", 32),
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
		},
		RateLimit: RateLimitSetting{
			IP:    RateLimitRule{Enabled: true, Capacity: 100, RefillPerSecond: 10},
			User:  RateLimitRule{Enabled: true, Capacity: 300, RefillPerSecond: 30},
			Login: RateLimitRule{Enabled: true, Capacity: 5, RefillPerSecond: 0.05},
		},
	}
}

func TestConfigValidate_HopLe(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("cấu hình hợp lệ nhưng Validate báo lỗi: %v", err)
	}
}

func TestConfigValidate_BatLoi(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantMsg string
	}{
		{
			name:    "port ngoài khoảng cho phép",
			modify:  func(c *Config) { c.Server.Port = 70000 },
			wantMsg: "server.port",
		},
		{
			name:    "mode không hợp lệ",
			modify:  func(c *Config) { c.Server.Mode = "production" },
			wantMsg: "server.mode",
		},
		{
			name:    "thiếu host postgres",
			modify:  func(c *Config) { c.Postgres.Host = "" },
			wantMsg: "postgres.host",
		},
		{
			name:    "không có broker kafka",
			modify:  func(c *Config) { c.Kafka.Brokers = nil },
			wantMsg: "kafka.brokers",
		},
		{
			name:    "jwt secret quá ngắn",
			modify:  func(c *Config) { c.JWT.Secret = "ngan" },
			wantMsg: "jwt.secret",
		},
		{
			// Refresh token ngắn hơn access token là vô nghĩa: token dùng để
			// làm mới lại hết hạn trước cả token mà nó cần làm mới.
			name:    "refresh TTL ngắn hơn access TTL",
			modify:  func(c *Config) { c.JWT.RefreshTokenTTL = time.Minute },
			wantMsg: "refreshTokenTTL",
		},
		{
			name:    "hạn mức ip bật nhưng capacity bằng 0",
			modify:  func(c *Config) { c.RateLimit.IP.Capacity = 0 },
			wantMsg: "rateLimit.ip.capacity",
		},
		{
			name:    "hạn mức login bật nhưng refill bằng 0",
			modify:  func(c *Config) { c.RateLimit.Login.RefillPerSecond = 0 },
			wantMsg: "rateLimit.login.refillPerSecond",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modify(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("mong đợi Validate báo lỗi, nhưng trả về nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("thông báo lỗi không nhắc tới %q\nnhận được: %v", tt.wantMsg, err)
			}
		})
	}
}

// Validate phải gom hết lỗi rồi trả về một lần, thay vì dừng ở lỗi đầu
// tiên — để người dùng sửa được toàn bộ file cấu hình trong một lượt.
func TestConfigValidate_GomNhieuLoi(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = -1
	cfg.Postgres.Host = ""
	cfg.JWT.Secret = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("mong đợi Validate báo lỗi")
	}

	for _, want := range []string{"server.port", "postgres.host", "jwt.secret"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("thiếu lỗi %q trong kết quả gộp:\n%v", want, err)
		}
	}
}

// Hạn mức đang tắt thì tham số của nó không cần hợp lệ — nếu không,
// muốn tắt rate limit sẽ phải điền số giả vào config.
func TestRateLimitRule_TatThiBoQuaKiemTra(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimit.User = RateLimitRule{Enabled: false, Capacity: 0, RefillPerSecond: 0}

	if err := cfg.Validate(); err != nil {
		t.Errorf("hạn mức đang tắt không nên bị kiểm tra tham số, nhưng lỗi: %v", err)
	}
}

func TestPostgresDSN(t *testing.T) {
	p := PostgresSetting{
		Host: "localhost", Port: 5432, User: "fintrack",
		Password: "secret", DBName: "fintrack", SSLMode: "disable",
	}

	want := "postgres://fintrack:secret@localhost:5432/fintrack?sslmode=disable"
	if got := p.DSN(); got != want {
		t.Errorf("DSN()\n  nhận được: %s\n  mong đợi:  %s", got, want)
	}
}

func TestRedisAddr(t *testing.T) {
	r := RedisSetting{Host: "127.0.0.1", Port: 6379}
	if got, want := r.Addr(), "127.0.0.1:6379"; got != want {
		t.Errorf("Addr() = %s, mong đợi %s", got, want)
	}
}
