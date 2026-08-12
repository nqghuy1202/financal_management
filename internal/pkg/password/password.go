// Package password lo việc băm và kiểm tra mật khẩu.
//
// Dùng bcrypt vì ba lý do:
//   - Nó tự sinh salt ngẫu nhiên và nhét luôn vào chuỗi kết quả, nên ta
//     không phải tự quản lý salt.
//   - Nó cố tình chạy chậm. Người dùng đăng nhập chờ thêm vài chục mili
//     giây thì không thấy gì, nhưng kẻ dò mật khẩu thì chậm đi hàng
//     nghìn lần.
//   - Chuỗi kết quả chứa sẵn tham số cost, nên sau này tăng cost vẫn đọc
//     được mật khẩu cũ.
//
// Tuyệt đối không dùng MD5 hay SHA-256 cho mật khẩu: chúng được thiết kế
// để chạy NHANH, đúng thứ kẻ tấn công cần.
package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// cost quyết định bcrypt chạy chậm cỡ nào. Mỗi lần tăng 1 là thời gian
// băm gấp đôi. 12 mất khoảng 200-300ms trên máy thường — đủ chậm để
// chống dò, vẫn đủ nhanh để người dùng không thấy khó chịu.
const cost = 12

// ErrMismatch trả về khi mật khẩu không khớp.
var ErrMismatch = errors.New("mật khẩu không đúng")

// bcrypt chỉ dùng 72 byte đầu của mật khẩu và báo lỗi nếu dài hơn.
// Chặn sớm ở đây để thông báo lỗi dễ hiểu hơn lỗi của thư viện.
const maxLength = 72

// Hash băm mật khẩu. Kết quả đã bao gồm salt và cost nên chỉ cần lưu
// đúng chuỗi này vào database.
func Hash(plain string) (string, error) {
	if len(plain) > maxLength {
		return "", fmt.Errorf("mật khẩu không được dài quá %d ký tự", maxLength)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("băm mật khẩu thất bại: %w", err)
	}

	return string(hashed), nil
}

// Verify kiểm tra mật khẩu có khớp với chuỗi đã băm hay không.
//
// Trả về ErrMismatch khi sai mật khẩu, và lỗi khác khi chuỗi băm hỏng
// (ví dụ dữ liệu trong DB bị sửa tay). Phân biệt hai trường hợp này để
// tầng trên biết đâu là lỗi người dùng, đâu là lỗi hệ thống.
func Verify(hashed, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	if err == nil {
		return nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrMismatch
	}
	return fmt.Errorf("kiểm tra mật khẩu thất bại: %w", err)
}
