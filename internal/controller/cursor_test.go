package controller

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEncodeDecodeCursor(t *testing.T) {
	// Múi giờ Việt Nam, đúng trường hợp từng gây lỗi.
	occurredAt := time.Date(2026, 8, 12, 12, 30, 0, 0, time.FixedZone("ICT", 7*3600))
	id := uuid.New()

	encoded := encodeCursor(&occurredAt, &id)
	if encoded == nil {
		t.Fatal("encodeCursor trả về nil với dữ liệu hợp lệ")
	}

	gotTime, gotID, err := decodeCursor(*encoded)
	if err != nil {
		t.Fatalf("decodeCursor lỗi: %v", err)
	}
	if !gotTime.Equal(occurredAt) {
		t.Errorf("thời điểm = %v, mong đợi %v", gotTime, occurredAt)
	}
	if *gotID != id {
		t.Errorf("id = %v, mong đợi %v", gotID, id)
	}
}

// Đây là lý do tồn tại của việc mã hoá con trỏ.
//
// Chuỗi RFC3339 của múi giờ Việt Nam chứa dấu "+" (ví dụ
// "2026-08-12T12:30:00+07:00"). Trong query string, dấu "+" được giải mã
// thành DẤU CÁCH, nên gửi thẳng chuỗi đó sẽ hỏng. Base64url không sinh ra
// "+", "/" hay "=" nên đi qua URL an toàn.
func TestEncodeCursor_AnToanTrenQueryString(t *testing.T) {
	occurredAt := time.Date(2026, 8, 12, 12, 30, 0, 0, time.FixedZone("ICT", 7*3600))
	id := uuid.New()

	encoded := *encodeCursor(&occurredAt, &id)

	for _, ch := range []string{"+", "/", "=", " "} {
		if strings.Contains(encoded, ch) {
			t.Errorf("con trỏ chứa ký tự %q, sẽ hỏng khi đi qua query string: %s", ch, encoded)
		}
	}

	// Đi qua vòng mã hoá/giải mã URL vẫn phải giữ nguyên.
	values := url.Values{}
	values.Set("cursor", encoded)
	parsed, err := url.ParseQuery(values.Encode())
	if err != nil {
		t.Fatalf("ParseQuery lỗi: %v", err)
	}
	if got := parsed.Get("cursor"); got != encoded {
		t.Errorf("sau khi qua query string: %q, ban đầu: %q", got, encoded)
	}
}

// Hết dữ liệu thì không có con trỏ, để client biết dừng.
func TestEncodeCursor_NilKhiHetDuLieu(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	if encodeCursor(nil, nil) != nil {
		t.Error("cả hai nil phải trả về nil")
	}
	if encodeCursor(&now, nil) != nil {
		t.Error("thiếu id phải trả về nil")
	}
	if encodeCursor(nil, &id) != nil {
		t.Error("thiếu thời điểm phải trả về nil")
	}
}

func TestDecodeCursor_RongLaTrangDau(t *testing.T) {
	gotTime, gotID, err := decodeCursor("")
	if err != nil {
		t.Fatalf("con trỏ rỗng không được coi là lỗi: %v", err)
	}
	if gotTime != nil || gotID != nil {
		t.Error("con trỏ rỗng phải trả về nil, nghĩa là lấy trang đầu")
	}
}

func TestDecodeCursor_ChuoiHong(t *testing.T) {
	// Chuỗi do client bịa ra không được làm server hoảng loạn.
	for _, bad := range []string{
		"khong-phai-base64!!!",
		"YWJj",             // base64 hợp lệ nhưng nội dung "abc", thiếu dấu ngăn cách
		"YWJjfGRlZg",       // "abc|def" — hai phần nhưng không phải thời gian và uuid
		"MjAyNi0wOC0xMnxY", // thời gian sai định dạng
	} {
		if _, _, err := decodeCursor(bad); err == nil {
			t.Errorf("decodeCursor(%q) phải báo lỗi", bad)
		}
	}
}
