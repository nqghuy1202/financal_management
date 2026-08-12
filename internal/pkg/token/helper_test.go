package token

import (
	"strings"
	"testing"
	"time"

	"financal_management/internal/pkg/setting"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newRedisManager tạo Manager nối tới một Redis giả chạy trong bộ nhớ.
//
// miniredis cài đặt giao thức Redis bằng Go thuần, nên test chạy được mà
// không cần Docker và không đụng tới dữ liệu Redis thật. Nhờ vậy các test
// này nằm trong `make test` chứ không phải `make itest`.
//
// t.Cleanup đảm bảo server giả được tắt sau mỗi test, và mỗi test có một
// Redis riêng nên không ảnh hưởng lẫn nhau.
func newRedisManager(t *testing.T) *Manager {
	t.Helper()

	srv := miniredis.RunT(t)

	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return NewManager(setting.JWTSetting{
		Secret:          strings.Repeat("k", 32),
		Issuer:          "fintrack-test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}, rdb)
}
