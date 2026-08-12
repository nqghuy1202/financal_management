package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

// Money là một số tiền gắn liền với loại tiền tệ của nó.
//
// Vì sao cần kiểu riêng thay vì dùng decimal.Decimal trần:
//
//  1. Số tiền không có loại tiền là vô nghĩa. "1000" là một nghìn đồng
//     hay một nghìn đô? Gói chung lại thì không thể quên.
//  2. Cộng hai loại tiền khác nhau là lỗi. Với decimal.Decimal trần,
//     phép cộng đó biên dịch bình thường và cho ra kết quả sai lặng lẽ.
//     Với Money, nó trả về lỗi.
//  3. Kiểu này không có phương thức nào nhận float64, nên không có đường
//     nào để số thực lọt vào phép tính tiền.
//
// Hai trường đều không xuất khẩu, nên bên ngoài chỉ tạo được Money qua
// hàm khởi tạo có kiểm tra.
type Money struct {
	amount   decimal.Decimal
	currency string
}

var (
	// ErrCurrencyMismatch khi cộng trừ hai loại tiền khác nhau.
	ErrCurrencyMismatch = errors.New("không thể tính toán giữa hai loại tiền khác nhau")
	// ErrInvalidCurrency khi mã tiền tệ sai định dạng.
	ErrInvalidCurrency = errors.New("mã tiền tệ phải gồm 3 chữ cái viết hoa")
)

// currencyPattern kiểm tra mã tiền tệ theo chuẩn ISO 4217: đúng 3 chữ
// cái viết hoa, ví dụ VND, USD, EUR.
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// NewMoney tạo một số tiền.
func NewMoney(amount decimal.Decimal, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !currencyPattern.MatchString(currency) {
		return Money{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
	}
	return Money{amount: amount, currency: currency}, nil
}

// ParseMoney tạo số tiền từ chuỗi, ví dụ ParseMoney("125000.50", "VND").
//
// Nhận chuỗi chứ không nhận float64: float64 không biểu diễn chính xác
// được các số thập phân thông thường, ví dụ 0.1 + 0.2 cho ra
// 0.30000000000000004. Với tiền thì sai số đó không chấp nhận được.
func ParseMoney(amount, currency string) (Money, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(amount))
	if err != nil {
		return Money{}, fmt.Errorf("số tiền không hợp lệ %q: %w", amount, err)
	}
	return NewMoney(d, currency)
}

// Zero trả về số tiền 0 của một loại tiền.
func Zero(currency string) (Money, error) {
	return NewMoney(decimal.Zero, currency)
}

func (m Money) Amount() decimal.Decimal { return m.amount }
func (m Money) Currency() string        { return m.currency }

// Add cộng hai số tiền. Trả lỗi nếu khác loại tiền.
func (m Money) Add(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	return Money{amount: m.amount.Add(other.amount), currency: m.currency}, nil
}

// Sub trừ hai số tiền. Trả lỗi nếu khác loại tiền.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	return Money{amount: m.amount.Sub(other.amount), currency: m.currency}, nil
}

func (m Money) IsZero() bool     { return m.amount.IsZero() }
func (m Money) IsPositive() bool { return m.amount.IsPositive() }
func (m Money) IsNegative() bool { return m.amount.IsNegative() }

// LessThan so sánh hai số tiền cùng loại.
func (m Money) LessThan(other Money) (bool, error) {
	if err := m.sameCurrency(other); err != nil {
		return false, err
	}
	return m.amount.LessThan(other.amount), nil
}

// String trả về dạng "125000.5 VND", dùng cho log và thông báo lỗi.
func (m Money) String() string {
	return m.amount.String() + " " + m.currency
}

func (m Money) sameCurrency(other Money) error {
	if m.currency != other.currency {
		return fmt.Errorf("%w: %s và %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return nil
}

// moneyJSON là dạng Money khi đi qua JSON.
type moneyJSON struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// MarshalJSON xuất số tiền dưới dạng CHUỖI, không phải số.
//
// Lý do: JavaScript đọc mọi số trong JSON thành float64. Một số tiền lớn
// như 9007199254740993 sẽ bị làm tròn sai ngay khi frontend gọi
// JSON.parse. Trả về chuỗi thì frontend nhận đúng từng chữ số và tự
// quyết định cách hiển thị.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(moneyJSON{
		Amount:   m.amount.String(),
		Currency: m.currency,
	})
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var raw moneyJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parsed, err := ParseMoney(raw.Amount, raw.Currency)
	if err != nil {
		return err
	}

	*m = parsed
	return nil
}
