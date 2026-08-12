// Package global chứa các đối tượng xuyên suốt toàn ứng dụng, được khởi
// tạo một lần duy nhất ở tầng initialize và chỉ đọc sau đó.
//
// Chỉ hai thứ được đặt ở đây: cấu hình và logger. Chúng là mối quan tâm
// cắt ngang mọi tầng, và truyền chúng qua từng constructor sẽ làm nhiễu
// chữ ký hàm ở khắp nơi mà không đổi lại được gì.
//
// Connection pool của PostgreSQL và client Redis cố tình KHÔNG nằm ở đây.
// Chúng được tạo trong initialize rồi truyền tường minh xuống repo qua
// constructor, để tầng repo có thể được test với database tạm mà không
// phải ghi đè biến toàn cục.
package global

import (
	"financal_management/internal/pkg/setting"

	"go.uber.org/zap"
)

var (
	// Config là cấu hình đã được nạp và validate.
	Config setting.Config

	// Logger là logger dùng chung. Trước khi InitLogger chạy xong, biến
	// này là nil — không log gì trước thời điểm đó.
	Logger *zap.Logger
)
