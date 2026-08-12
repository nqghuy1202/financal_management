package middlewares

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"financal_management/global"
	"financal_management/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// tokenBucketScript cài đặt thuật toán token bucket trong Redis.
//
// Vì sao dùng Lua: các bước đọc token, tính lượng nạp thêm rồi ghi lại
// phải diễn ra nguyên tử. Nếu làm bằng nhiều lệnh Redis riêng lẻ, hai
// request đồng thời có thể cùng đọc ra số token cũ và cùng được cho qua.
// Redis chạy mỗi script Lua như một thao tác nguyên tử nên tránh được
// race đó mà không cần khoá phân tán.
//
// So với fixed window counter, token bucket không bị lỗi "burst gấp đôi"
// ở ranh giới cửa sổ, đồng thời vẫn cho phép burst có kiểm soát.
var tokenBucketScript = redis.NewScript(`
local key      = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill   = tonumber(ARGV[2])  -- token nạp thêm mỗi giây
local now      = tonumber(ARGV[3])  -- thời điểm hiện tại, tính bằng ms

local bucket = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(bucket[1])
local ts     = tonumber(bucket[2])

if tokens == nil then
  tokens = capacity
  ts     = now
end

-- Nạp lại token theo thời gian đã trôi qua kể từ lần gọi trước.
local elapsed = math.max(0, now - ts) / 1000
tokens = math.min(capacity, tokens + elapsed * refill)

local allowed = 0
if tokens >= 1 then
  tokens  = tokens - 1
  allowed = 1
end

redis.call('HSET', key, 'tokens', tokens, 'ts', now)
-- Hết hạn sau khoảng thời gian đủ để bucket đầy lại, tránh rác trong Redis.
redis.call('PEXPIRE', key, math.ceil(capacity / refill * 1000) + 1000)

return {allowed, math.floor(tokens)}
`)

// RateLimit giới hạn số request theo từng client.
//
// Khoá được tính theo user id nếu request đã xác thực, ngược lại theo IP.
// Nhờ vậy nhiều người dùng sau cùng một NAT không bị tính chung hạn mức.
func RateLimit() gin.HandlerFunc {
	cfg := global.Config.RateLimit

	if !cfg.Enabled {
		// Trả về middleware rỗng thay vì kiểm tra cờ ở mỗi request.
		return func(c *gin.Context) { c.Next() }
	}

	capacity := strconv.Itoa(cfg.Capacity)
	refill := strconv.FormatFloat(cfg.RefillPerSecond, 'f', -1, 64)

	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		key := fmt.Sprintf("ratelimit:%s", clientKey(c))
		now := strconv.FormatInt(time.Now().UnixMilli(), 10)

		res, err := tokenBucketScript.Run(ctx, global.Redis, []string{key}, capacity, refill, now).Slice()
		if err != nil {
			// Fail-open: Redis lỗi thì cho request đi qua thay vì chặn toàn
			// bộ hệ thống. Rate limit là lớp bảo vệ, không phải lớp bắt buộc.
			global.Logger.Warn("rate limit không kiểm tra được, cho request đi qua",
				zap.String("key", key),
				zap.Error(err),
			)
			c.Next()
			return
		}

		allowed, _ := res[0].(int64)
		remaining, _ := res[1].(int64)

		c.Writer.Header().Set("X-RateLimit-Limit", capacity)
		c.Writer.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if allowed == 0 {
			response.ErrorWithCode(c, response.CodeTooManyRequests)
			c.Abort()
			return
		}

		c.Next()
	}
}

// clientKey ưu tiên định danh theo người dùng đã đăng nhập, không có thì
// rơi về địa chỉ IP.
func clientKey(c *gin.Context) string {
	if userID := c.GetString(ContextUserIDKey); userID != "" {
		return "user:" + userID
	}
	return "ip:" + c.ClientIP()
}
