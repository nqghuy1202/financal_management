package model

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func mustMoney(t *testing.T, amount, currency string) Money {
	t.Helper()
	m, err := ParseMoney(amount, currency)
	if err != nil {
		t.Fatalf("ParseMoney(%q, %q) lỗi: %v", amount, currency, err)
	}
	return m
}

func TestParseMoney(t *testing.T) {
	m := mustMoney(t, "125000.50", "VND")

	if got := m.Amount().String(); got != "125000.5" {
		t.Errorf("Amount() = %s, mong đợi 125000.5", got)
	}
	if m.Currency() != "VND" {
		t.Errorf("Currency() = %s, mong đợi VND", m.Currency())
	}
}

func TestNewMoney_MaTienTeSai(t *testing.T) {
	for _, currency := range []string{"", "VN", "VNDD", "vn1", "123"} {
		if _, err := NewMoney(decimal.NewFromInt(1), currency); !errors.Is(err, ErrInvalidCurrency) {
			t.Errorf("mã tiền tệ %q phải bị từ chối, nhận được lỗi: %v", currency, err)
		}
	}
}

// Mã tiền tệ viết thường vẫn nhận, nhưng được chuẩn hoá về viết hoa.
func TestNewMoney_ChuanHoaMaTienTe(t *testing.T) {
	m, err := ParseMoney("100", " vnd ")
	if err != nil {
		t.Fatalf("ParseMoney lỗi: %v", err)
	}
	if m.Currency() != "VND" {
		t.Errorf("Currency() = %q, mong đợi VND", m.Currency())
	}
}

func TestAdd(t *testing.T) {
	a := mustMoney(t, "100000", "VND")
	b := mustMoney(t, "50000", "VND")

	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add lỗi: %v", err)
	}
	if got := sum.Amount().String(); got != "150000" {
		t.Errorf("tổng = %s, mong đợi 150000", got)
	}
}

// Đây là lý do chính để có kiểu Money: cộng hai loại tiền khác nhau phải
// báo lỗi chứ không được lặng lẽ cho ra số vô nghĩa.
func TestAdd_KhacLoaiTien(t *testing.T) {
	vnd := mustMoney(t, "100000", "VND")
	usd := mustMoney(t, "10", "USD")

	if _, err := vnd.Add(usd); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("cộng VND với USD phải báo ErrCurrencyMismatch, nhận được: %v", err)
	}
	if _, err := vnd.Sub(usd); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("trừ VND cho USD phải báo ErrCurrencyMismatch, nhận được: %v", err)
	}
	if _, err := vnd.LessThan(usd); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("so sánh VND với USD phải báo ErrCurrencyMismatch, nhận được: %v", err)
	}
}

// Phép tính hay sai nhất với số thực: 0.1 + 0.2 trong float64 cho ra
// 0.30000000000000004. Với decimal thì phải đúng bằng 0.3.
func TestAdd_KhongCoSaiSoSoThuc(t *testing.T) {
	a := mustMoney(t, "0.1", "USD")
	b := mustMoney(t, "0.2", "USD")

	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add lỗi: %v", err)
	}
	if got := sum.Amount().String(); got != "0.3" {
		t.Errorf("0.1 + 0.2 = %s, mong đợi đúng 0.3", got)
	}
}

func TestSub(t *testing.T) {
	a := mustMoney(t, "100000", "VND")
	b := mustMoney(t, "30000", "VND")

	diff, err := a.Sub(b)
	if err != nil {
		t.Fatalf("Sub lỗi: %v", err)
	}
	if got := diff.Amount().String(); got != "70000" {
		t.Errorf("hiệu = %s, mong đợi 70000", got)
	}
	if !diff.IsPositive() {
		t.Error("70000 phải là số dương")
	}
}

func TestSub_RaSoAm(t *testing.T) {
	a := mustMoney(t, "10000", "VND")
	b := mustMoney(t, "30000", "VND")

	diff, err := a.Sub(b)
	if err != nil {
		t.Fatalf("Sub lỗi: %v", err)
	}
	if !diff.IsNegative() {
		t.Error("10000 - 30000 phải là số âm")
	}
}

// JSON phải xuất số tiền dưới dạng chuỗi. Nếu xuất dạng số, JavaScript
// sẽ đọc thành float64 và làm tròn sai với số lớn.
func TestMarshalJSON_XuatDangChuoi(t *testing.T) {
	m := mustMoney(t, "9007199254740993", "VND")

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal lỗi: %v", err)
	}

	want := `{"amount":"9007199254740993","currency":"VND"}`
	if string(data) != want {
		t.Errorf("JSON = %s\nmong đợi  = %s", data, want)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	var m Money
	if err := json.Unmarshal([]byte(`{"amount":"125000.50","currency":"VND"}`), &m); err != nil {
		t.Fatalf("Unmarshal lỗi: %v", err)
	}

	if got := m.Amount().String(); got != "125000.5" {
		t.Errorf("Amount() = %s, mong đợi 125000.5", got)
	}
	if m.Currency() != "VND" {
		t.Errorf("Currency() = %s, mong đợi VND", m.Currency())
	}
}

// Số tiền lớn phải đi qua JSON mà không mất một chữ số nào.
func TestJSON_KhongMatDoChinhXac(t *testing.T) {
	original := mustMoney(t, "123456789012345.6789", "VND")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal lỗi: %v", err)
	}

	var decoded Money
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal lỗi: %v", err)
	}

	if !decoded.Amount().Equal(original.Amount()) {
		t.Errorf("sau khi qua JSON: %s, ban đầu: %s", decoded.Amount(), original.Amount())
	}
}

func TestString(t *testing.T) {
	m := mustMoney(t, "125000", "VND")
	if got, want := m.String(), "125000 VND"; got != want {
		t.Errorf("String() = %q, mong đợi %q", got, want)
	}
}

func TestZero(t *testing.T) {
	m, err := Zero("VND")
	if err != nil {
		t.Fatalf("Zero lỗi: %v", err)
	}
	if !m.IsZero() {
		t.Error("Zero() phải trả về số tiền bằng 0")
	}
}

func TestParseMoney_ChuoiKhongPhaiSo(t *testing.T) {
	if _, err := ParseMoney("mot tram nghin", "VND"); err == nil {
		t.Error("chuỗi không phải số phải báo lỗi")
	}
}
