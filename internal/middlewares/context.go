package middlewares

// Các khoá dùng để lưu thông tin vào gin.Context.
//
// Gom về một chỗ để tránh gõ nhầm chuỗi ở nhiều nơi — lỗi kiểu đó
// compiler không bắt được.
const (
	// ContextUserIDKey lưu id của người dùng đã xác thực, do middleware
	// RequireAuth gán vào (được cài đặt ở Phase 1).
	ContextUserIDKey = "user_id"

	// ContextUserEmailKey lưu email của người dùng đã xác thực.
	ContextUserEmailKey = "user_email"
)
