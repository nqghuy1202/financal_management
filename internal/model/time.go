package model

import (
	"sync"
	"time"
)

// AppTimezone là múi giờ dùng để hiểu và cắt các mốc thời gian nghiệp vụ.
//
// Vì sao cần: cột occurred_at lưu theo UTC, nhưng người dùng nghĩ theo
// giờ địa phương. Khi họ hỏi "chi tiêu tháng 8", ranh giới phải là
// 00:00 ngày 01/08 GIỜ VIỆT NAM, tương ứng 17:00 ngày 31/07 UTC.
//
// Nếu lấy nhầm 00:00 UTC làm ranh giới thì một khoản chi lúc 00:30 ngày
// 01/08 giờ Việt Nam sẽ bị tính vào tháng 7 — người dùng thấy con số sai
// mà không hiểu vì sao.
//
// Hiện đặt cứng vì ứng dụng phục vụ người dùng Việt Nam. Khi cần hỗ trợ
// nhiều múi giờ thì thêm cột timezone vào bảng users; các câu SQL đã
// nhận múi giờ làm tham số nên phần database không phải sửa.
const AppTimezone = "Asia/Ho_Chi_Minh"

var (
	appLocationOnce sync.Once
	appLocation     *time.Location
)

// AppLocation trả về múi giờ ứng dụng.
//
// Kết quả được nạp một lần rồi dùng lại, vì time.LoadLocation phải đọc
// file cơ sở dữ liệu múi giờ của hệ điều hành.
//
// Lùi về UTC nếu máy chủ không có cơ sở dữ liệu múi giờ — hay gặp trên
// image Docker tối giản. Báo cáo lệch vài giờ vẫn tốt hơn là hỏng hẳn.
func AppLocation() *time.Location {
	appLocationOnce.Do(func() {
		loc, err := time.LoadLocation(AppTimezone)
		if err != nil {
			loc = time.UTC
		}
		appLocation = loc
	})
	return appLocation
}
