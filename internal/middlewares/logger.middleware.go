package middlewares

import (
	"time"

	"financal_management/global"
	"financal_management/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestIDHeader là header client có thể gửi lên để nối request của mình
// với log phía server. Nếu không có, server tự sinh.
const RequestIDHeader = "X-Request-ID"

// RequestID gán một id duy nhất cho mỗi request, đưa vào context và trả
// lại qua response header.
//
// Đây là nền tảng để lần vết: cùng một id sẽ xuất hiện trong log của
// API, trong event gửi sang Kafka và trong log của worker.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}

		c.Set(response.RequestIDKey, id)
		c.Writer.Header().Set(RequestIDHeader, id)

		c.Next()
	}
}

// Logger ghi lại một dòng log có cấu trúc cho mỗi request đã hoàn tất.
//
// Thay cho gin.Logger() mặc định (ghi text ra stdout), middleware này ghi
// JSON qua zap để log có thể được truy vấn theo trường.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		status := c.Writer.Status()
		fields := []zap.Field{
			zap.String("request_id", c.GetString(response.RequestIDKey)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Int("bytes", c.Writer.Size()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Lỗi do middleware xử lý lỗi ghi vào c.Errors được đính kèm ở đây
		// để không phải log hai lần.
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		switch {
		case status >= 500:
			global.Logger.Error("request thất bại", fields...)
		case status >= 400:
			global.Logger.Warn("request bị từ chối", fields...)
		default:
			global.Logger.Info("request thành công", fields...)
		}
	}
}
