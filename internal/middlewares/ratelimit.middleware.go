package middlewares

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"financal_management/global"
	"financal_management/internal/pkg/response"
	"financal_management/internal/pkg/setting"

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

// KeyFunc trích ra định danh dùng làm khoá hạn mức cho một request.
type KeyFunc func(c *gin.Context) string

// KeyByIP khoá theo địa chỉ IP của client. Dùng cho request chưa xác thực.
//
// Không gắn tiền tố "ip:" vì scope truyền cho RateLimit đã nằm trong khoá
// Redis rồi; thêm nữa chỉ tạo ra khoá kiểu "ratelimit:ip:ip:...".
func KeyByIP(c *gin.Context) string {
	return c.ClientIP()
}

// KeyByUser khoá theo người dùng đã đăng nhập.
//
// Chỉ dùng được cho route nằm SAU middleware xác thực. Nếu đặt trước, giá
// trị user id chưa tồn tại và mọi request sẽ rơi về khoá IP — biến hạn
// mức theo user thành vô nghĩa. Rơi về IP ở đây là lưới an toàn, không
// phải cách dùng dự kiến.
//
// Hai nhánh dùng tiền tố khác nhau để một user id không bao giờ đụng khoá
// với một địa chỉ IP.
func KeyByUser(c *gin.Context) string {
	if userID := c.GetString(ContextUserIDKey); userID != "" {
		return "u:" + userID
	}
	return "anon:" + c.ClientIP()
}

// RateLimit giới hạn số request theo một hạn mức và cách khoá cho trước.
//
// scope đi vào khoá Redis nên các hạn mức khác nhau đếm độc lập: một
// người dùng bị chặn ở nhóm "login" vẫn còn nguyên hạn mức ở nhóm "user".
func RateLimit(rdb *redis.Client, scope string, rule setting.RateLimitRule, key KeyFunc) gin.HandlerFunc {
	if !rule.Enabled {
		// Trả về middleware rỗng thay vì kiểm tra cờ ở mỗi request.
		return func(c *gin.Context) { c.Next() }
	}

	capacity := strconv.Itoa(rule.Capacity)
	refill := strconv.FormatFloat(rule.RefillPerSecond, 'f', -1, 64)

	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		redisKey := fmt.Sprintf("ratelimit:%s:%s", scope, key(c))
		now := strconv.FormatInt(time.Now().UnixMilli(), 10)

		res, err := tokenBucketScript.Run(ctx, rdb, []string{redisKey}, capacity, refill, now).Slice()
		if err != nil {
			// Fail-open: Redis lỗi thì cho request đi qua thay vì chặn toàn
			// bộ hệ thống. Rate limit là lớp bảo vệ, không phải lớp bắt buộc.
			global.Logger.Warn("rate limit không kiểm tra được, cho request đi qua",
				zap.String("scope", scope),
				zap.String("key", redisKey),
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
