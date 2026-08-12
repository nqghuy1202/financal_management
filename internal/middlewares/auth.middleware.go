package middlewares

import (
	"errors"
	"strings"

	"financal_management/internal/pkg/response"
	"financal_management/internal/pkg/token"

	"github.com/gin-gonic/gin"
)

// bearerPrefix là tiền tố chuẩn của header Authorization theo RFC 6750.
const bearerPrefix = "Bearer "

// RequireAuth chặn request không có access token hợp lệ.
//
// Khi token hợp lệ, id và email của người dùng được đặt vào context để
// handler phía sau dùng mà không phải giải mã lại token.
func RequireAuth(tm *token.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, ok := extractBearerToken(c)
		if !ok {
			response.ErrorWithCode(c, response.CodeTokenMissing)
			c.Abort()
			return
		}

		claims, err := tm.ParseAccess(accessToken)
		if err != nil {
			// Phân biệt "hết hạn" với "không hợp lệ" để frontend biết khi
			// nào nên gọi refresh, khi nào phải bắt đăng nhập lại.
			code := response.CodeTokenInvalid
			if errors.Is(err, token.ErrExpiredToken) {
				code = response.CodeTokenExpired
			}
			response.ErrorWithCode(c, code)
			c.Abort()
			return
		}

		c.Set(ContextUserIDKey, claims.UserID.String())
		c.Set(ContextUserEmailKey, claims.Email)

		c.Next()
	}
}

// extractBearerToken lấy token từ header `Authorization: Bearer <token>`.
func extractBearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if header == "" {
		return "", false
	}

	// So sánh không phân biệt hoa thường vì một số client gửi "bearer".
	if len(header) <= len(bearerPrefix) ||
		!strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return "", false
	}

	value := strings.TrimSpace(header[len(bearerPrefix):])
	if value == "" {
		return "", false
	}

	return value, true
}
