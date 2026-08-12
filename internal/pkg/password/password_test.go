package password

import (
	"errors"
	"strings"
	"testing"
)

func TestHashVaVerify(t *testing.T) {
	plain := "matkhau-cua-toi-123"

	hashed, err := Hash(plain)
	if err != nil {
		t.Fatalf("Hash lỗi: %v", err)
	}

	// Chuỗi băm không được chứa mật khẩu gốc.
	if strings.Contains(hashed, plain) {
		t.Error("chuỗi băm chứa mật khẩu gốc")
	}

	if err := Verify(hashed, plain); err != nil {
		t.Errorf("Verify với mật khẩu đúng phải thành công, nhưng lỗi: %v", err)
	}
}

func TestVerify_SaiMatKhau(t *testing.T) {
	hashed, err := Hash("dung")
	if err != nil {
		t.Fatalf("Hash lỗi: %v", err)
	}

	err = Verify(hashed, "sai")
	if !errors.Is(err, ErrMismatch) {
		t.Errorf("mong đợi ErrMismatch, nhận được: %v", err)
	}
}

// bcrypt sinh salt ngẫu nhiên mỗi lần, nên cùng một mật khẩu phải cho ra
// hai chuỗi băm khác nhau. Nếu giống nhau nghĩa là salt không hoạt động,
// và kẻ tấn công có thể dùng bảng tra sẵn để phá hàng loạt mật khẩu.
func TestHash_SinhSaltKhacNhauMoiLan(t *testing.T) {
	plain := "cung-mot-mat-khau"

	first, err := Hash(plain)
	if err != nil {
		t.Fatalf("Hash lần 1 lỗi: %v", err)
	}
	second, err := Hash(plain)
	if err != nil {
		t.Fatalf("Hash lần 2 lỗi: %v", err)
	}

	if first == second {
		t.Error("hai lần băm cùng mật khẩu cho ra kết quả giống nhau, salt không hoạt động")
	}

	// Dù khác nhau, cả hai đều phải kiểm tra được.
	if err := Verify(first, plain); err != nil {
		t.Errorf("chuỗi băm thứ nhất không kiểm tra được: %v", err)
	}
	if err := Verify(second, plain); err != nil {
		t.Errorf("chuỗi băm thứ hai không kiểm tra được: %v", err)
	}
}

func TestHash_MatKhauQuaDai(t *testing.T) {
	// bcrypt chỉ xử lý 72 byte đầu và báo lỗi nếu dài hơn.
	_, err := Hash(strings.Repeat("a", 73))
	if err == nil {
		t.Error("mật khẩu dài quá 72 ký tự phải báo lỗi")
	}
}

func TestVerify_ChuoiBamHong(t *testing.T) {
	err := Verify("day-khong-phai-chuoi-bcrypt", "matkhau")
	if err == nil {
		t.Fatal("chuỗi băm hỏng phải báo lỗi")
	}
	// Chuỗi băm hỏng là lỗi hệ thống, không phải "sai mật khẩu" —
	// hai trường hợp này phải phân biệt được.
	if errors.Is(err, ErrMismatch) {
		t.Error("chuỗi băm hỏng không được báo là ErrMismatch")
	}
}
