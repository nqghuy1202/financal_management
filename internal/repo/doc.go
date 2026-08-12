// Package repo chứa tầng truy cập dữ liệu.
//
// Quy tắc của tầng này:
//   - Chỉ đọc/ghi dữ liệu, không chứa logic nghiệp vụ. Câu hỏi "được
//     phép làm việc này không" thuộc về tầng service.
//   - Mỗi repo được khai báo bằng một interface để tầng service phụ
//     thuộc vào interface đó thay vì vào cài đặt cụ thể.
//   - Nhận vào và trả về struct trong internal/model, không trả về kiểu
//     riêng của driver database.
//   - Từ Phase 1, phần lớn code trong package này được sqlc sinh ra từ
//     các file SQL trong thư mục migrations và queries.
package repo
