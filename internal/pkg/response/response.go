package response

import (
	"github.com/gin-gonic/gin"
)

// RequestIDKey là khoá lưu request id trong gin.Context, do middleware
// logger gán vào. Đặt ở đây để cả response và middleware cùng dùng chung.
const RequestIDKey = "request_id"

// ResponseData là envelope chung cho mọi response của API.
type ResponseData struct {
	Code      int    `json:"code"`                 // mã nghiệp vụ, xem code.go
	Message   string `json:"message"`              // thông báo cho người dùng
	Data      any    `json:"data,omitempty"`       // dữ liệu trả về
	RequestID string `json:"request_id,omitempty"` // dùng để tra log khi có sự cố
}

// Success trả về 200 kèm dữ liệu.
func Success(c *gin.Context, data any) {
	WithCode(c, CodeSuccess, data)
}

// WithCode trả về response với mã nghiệp vụ chỉ định. HTTP status được
// suy ra từ catalog nên không thể lệch giữa mã và status.
func WithCode(c *gin.Context, code int, data any) {
	c.JSON(HTTPStatus(code), ResponseData{
		Code:      code,
		Message:   Message(code),
		Data:      data,
		RequestID: requestID(c),
	})
}

// Error trả về lỗi cho client dựa trên AppError.
//
// Chỉ Code và Message được gửi đi; lỗi gốc bên trong AppError do
// middleware xử lý lỗi ghi vào log.
func Error(c *gin.Context, err error) {
	appErr := AsAppError(err)
	c.JSON(HTTPStatus(appErr.Code), ResponseData{
		Code:      appErr.Code,
		Message:   appErr.Message,
		RequestID: requestID(c),
	})
}

// ErrorWithCode là lối tắt khi nơi gọi chỉ có sẵn mã lỗi.
func ErrorWithCode(c *gin.Context, code int) {
	Error(c, New(code))
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get(RequestIDKey); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}
