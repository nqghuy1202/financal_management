// Package services chứa logic nghiệp vụ của ứng dụng.
//
// Vị trí trong luồng xử lý:
//
//	controller  →  service  →  repo  →  database
//
// Quy tắc của tầng này:
//   - Không biết gì về HTTP. Không import gin, không đụng tới
//     *gin.Context, không trả về HTTP status. Nhờ vậy cùng một service
//     dùng được cho cả HTTP handler lẫn Kafka consumer.
//   - Nhận repo qua interface, không phụ thuộc vào cài đặt cụ thể — để
//     test được bằng mock.
//   - Trả về *response.AppError mang mã lỗi nghiệp vụ khi thất bại.
//   - Là nơi duy nhất định nghĩa ranh giới transaction của database.
package services
