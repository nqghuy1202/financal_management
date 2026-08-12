// Command api khởi chạy HTTP API server của hệ thống quản lý tài chính.
package main

import (
	"fmt"
	"os"

	"financal_management/internal/initialize"
)

func main() {
	if err := initialize.Run(); err != nil {
		// Ghi ra stderr thay vì logger, vì lỗi có thể xảy ra ngay trước khi
		// logger kịp được khởi tạo.
		fmt.Fprintf(os.Stderr, "khởi động thất bại: %v\n", err)
		os.Exit(1)
	}
}
