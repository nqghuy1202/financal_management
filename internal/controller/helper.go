package controller

import (
	"financal_management/internal/middlewares"
	"financal_management/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// keyMessage là khoá JSON cho các response chỉ trả về một lời nhắn.
const keyMessage = "message"

// currentUserID lấy id người dùng đã đăng nhập từ context.
//
// Chỉ dùng được trong handler nằm sau middleware RequireAuth. Trả về
// false khi không lấy được, và đã ghi lỗi vào context — nơi gọi chỉ cần
// return ngay.
func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetString(middlewares.ContextUserIDKey)
	if raw == "" {
		// Tới đây mà không có user id nghĩa là route bị quên gắn
		// RequireAuth — lỗi lập trình chứ không phải lỗi người dùng.
		_ = c.Error(response.New(response.CodeUnauthorized))
		return uuid.Nil, false
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		_ = c.Error(response.Wrap(response.CodeTokenInvalid, err))
		return uuid.Nil, false
	}

	return id, true
}

// pathUUID đọc một tham số UUID trên đường dẫn, ví dụ /accounts/:id.
func pathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		_ = c.Error(response.Newf(response.CodeInvalidParam,
			"Tham số %s phải là UUID hợp lệ", name))
		return uuid.Nil, false
	}
	return id, true
}
