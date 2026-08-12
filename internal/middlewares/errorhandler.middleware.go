package middlewares

import (
	"fmt"
	"runtime/debug"

	"financal_management/global"
	"financal_management/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery bắt panic ở bất kỳ handler nào, ghi log kèm stacktrace rồi trả
// về lỗi 500 chuẩn thay vì để kết nối bị đứt giữa chừng.
//
// Dùng thay cho gin.Recovery() mặc định để log ra zap và response đúng
// envelope của hệ thống.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				global.Logger.Error("panic trong handler",
					zap.String("request_id", c.GetString(response.RequestIDKey)),
					zap.String("path", c.Request.URL.Path),
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
				)

				response.ErrorWithCode(c, response.CodeInternalError)
				c.Abort()
			}
		}()

		c.Next()
	}
}

// ErrorHandler xử lý tập trung các lỗi mà handler đẩy vào c.Error(err).
//
// Nhờ middleware này, handler chỉ cần viết:
//
//	if err != nil {
//	    _ = c.Error(err)
//	    return
//	}
//
// mà không phải tự dựng response lỗi. Lỗi gốc được ghi log ở đây, còn
// client chỉ nhận mã và thông báo đã được lọc.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Lấy lỗi cuối cùng — thường handler chỉ đẩy vào đúng một lỗi rồi return.
		err := c.Errors.Last().Err
		appErr := response.AsAppError(err)

		fields := []zap.Field{
			zap.String("request_id", c.GetString(response.RequestIDKey)),
			zap.String("path", c.Request.URL.Path),
			zap.Int("code", appErr.Code),
		}
		if cause := appErr.Cause(); cause != nil {
			fields = append(fields, zap.Error(cause))
		}

		// Lỗi 5xx là lỗi của hệ thống nên log mức error; 4xx là lỗi phía
		// client, log mức warn để không gây nhiễu cảnh báo.
		if response.HTTPStatus(appErr.Code) >= 500 {
			global.Logger.Error(fmt.Sprintf("lỗi hệ thống: %s", appErr.Message), fields...)
		} else {
			global.Logger.Warn(fmt.Sprintf("lỗi nghiệp vụ: %s", appErr.Message), fields...)
		}

		// Nếu handler đã tự ghi response rồi thì không ghi đè.
		if c.Writer.Written() {
			return
		}
		response.Error(c, appErr)
	}
}
