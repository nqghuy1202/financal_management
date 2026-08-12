// Package model chứa các struct mô tả thực thể nghiệp vụ, dùng chung
// giữa các tầng controller, service và repo.
//
// Từ Phase 1, các struct ánh xạ trực tiếp với bảng database sẽ do sqlc
// sinh ra. Package này giữ những kiểu do ta tự định nghĩa: kiểu tiền tệ,
// các enum nghiệp vụ, và các struct tổng hợp không tương ứng 1-1 với một
// bảng nào.
package model
