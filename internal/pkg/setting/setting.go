package setting

import (
	"errors"
	"fmt"
	"time"
)

// Config là toàn bộ cấu hình của ứng dụng, được viper unmarshal từ file
// config/<env>.yaml và cho phép override bằng biến môi trường.
type Config struct {
	Server    ServerSetting    `mapstructure:"server"`
	Logger    LoggerSetting    `mapstructure:"logger"`
	Postgres  PostgresSetting  `mapstructure:"postgres"`
	Redis     RedisSetting     `mapstructure:"redis"`
	Kafka     KafkaSetting     `mapstructure:"kafka"`
	JWT       JWTSetting       `mapstructure:"jwt"`
	RateLimit RateLimitSetting `mapstructure:"rateLimit"`
}

type ServerSetting struct {
	// Mode: debug | release | test (giá trị của gin)
	Mode string `mapstructure:"mode"`
	Port int    `mapstructure:"port"`
	// AllowOrigins là danh sách origin được phép gọi API (dùng cho CORS).
	AllowOrigins []string `mapstructure:"allowOrigins"`
	// ShutdownTimeout là thời gian tối đa chờ các request đang xử lý hoàn tất.
	ShutdownTimeout time.Duration `mapstructure:"shutdownTimeout"`
	ReadTimeout     time.Duration `mapstructure:"readTimeout"`
	WriteTimeout    time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout     time.Duration `mapstructure:"idleTimeout"`
}

type LoggerSetting struct {
	// Level: debug | info | warn | error
	Level string `mapstructure:"level"`
	// FileName là đường dẫn file log. Bỏ trống thì chỉ ghi ra stdout.
	FileName string `mapstructure:"fileName"`
	// MaxSize tính bằng MB, khi vượt quá thì file được xoay vòng.
	MaxSize int `mapstructure:"maxSize"`
	// MaxBackups là số file log cũ được giữ lại.
	MaxBackups int `mapstructure:"maxBackups"`
	// MaxAge tính bằng ngày.
	MaxAge   int  `mapstructure:"maxAge"`
	Compress bool `mapstructure:"compress"`
}

type PostgresSetting struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbName"`
	SSLMode  string `mapstructure:"sslMode"`
	// MaxConns là số connection tối đa trong pool.
	MaxConns int32 `mapstructure:"maxConns"`
	// MinConns là số connection luôn được giữ sẵn.
	MinConns        int32         `mapstructure:"minConns"`
	MaxConnLifetime time.Duration `mapstructure:"maxConnLifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"maxConnIdleTime"`
	ConnectTimeout  time.Duration `mapstructure:"connectTimeout"`
}

// DSN trả về chuỗi kết nối dạng URL cho pgx.
func (p PostgresSetting) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.DBName, p.SSLMode,
	)
}

type RedisSetting struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"poolSize"`
}

func (r RedisSetting) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type KafkaSetting struct {
	Brokers []string `mapstructure:"brokers"`
	// TopicPrefix đứng trước mọi tên topic, ví dụ "fin" -> "fin.transaction.v1".
	TopicPrefix string `mapstructure:"topicPrefix"`
}

type JWTSetting struct {
	// Secret dùng để ký token HS256. Bắt buộc override bằng env ở production.
	Secret string `mapstructure:"secret"`
	// Issuer được ghi vào claim "iss".
	Issuer          string        `mapstructure:"issuer"`
	AccessTokenTTL  time.Duration `mapstructure:"accessTokenTTL"`
	RefreshTokenTTL time.Duration `mapstructure:"refreshTokenTTL"`
}

// RateLimitSetting cấu hình thuật toán token bucket cho rate limit.
type RateLimitSetting struct {
	Enabled bool `mapstructure:"enabled"`
	// Capacity là số token tối đa trong bucket, tức số request được phép
	// dồn dập trong một đợt burst.
	Capacity int `mapstructure:"capacity"`
	// RefillPerSecond là số token được nạp lại mỗi giây, tức tốc độ
	// request được duy trì ổn định về lâu dài.
	RefillPerSecond float64 `mapstructure:"refillPerSecond"`
}

// Validate kiểm tra các giá trị bắt buộc ngay lúc khởi động, để ứng dụng
// fail fast thay vì chạy được rồi mới lỗi giữa chừng.
func (c *Config) Validate() error {
	var errs []error

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port không hợp lệ: %d", c.Server.Port))
	}
	switch c.Server.Mode {
	case "debug", "release", "test":
	default:
		errs = append(errs, fmt.Errorf("server.mode phải là debug|release|test, đang là %q", c.Server.Mode))
	}
	if c.Postgres.Host == "" {
		errs = append(errs, errors.New("postgres.host không được rỗng"))
	}
	if c.Postgres.DBName == "" {
		errs = append(errs, errors.New("postgres.dbName không được rỗng"))
	}
	if c.Redis.Host == "" {
		errs = append(errs, errors.New("redis.host không được rỗng"))
	}
	if len(c.Kafka.Brokers) == 0 {
		errs = append(errs, errors.New("kafka.brokers không được rỗng"))
	}
	// Secret ngắn khiến token dễ bị brute-force; HS256 nên dùng khoá >= 32 byte.
	if len(c.JWT.Secret) < 32 {
		errs = append(errs, fmt.Errorf("jwt.secret phải dài ít nhất 32 ký tự, đang là %d", len(c.JWT.Secret)))
	}
	if c.JWT.AccessTokenTTL <= 0 {
		errs = append(errs, errors.New("jwt.accessTokenTTL phải lớn hơn 0"))
	}
	if c.JWT.RefreshTokenTTL <= c.JWT.AccessTokenTTL {
		errs = append(errs, errors.New("jwt.refreshTokenTTL phải lớn hơn accessTokenTTL"))
	}
	if c.RateLimit.Enabled {
		if c.RateLimit.Capacity <= 0 {
			errs = append(errs, errors.New("rateLimit.capacity phải lớn hơn 0 khi bật rate limit"))
		}
		if c.RateLimit.RefillPerSecond <= 0 {
			errs = append(errs, errors.New("rateLimit.refillPerSecond phải lớn hơn 0 khi bật rate limit"))
		}
	}

	return errors.Join(errs...)
}
