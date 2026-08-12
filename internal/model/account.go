package model

// Loại ví. Giá trị phải khớp với CHECK constraint của bảng accounts.
const (
	AccountTypeCash       = "cash"        // tiền mặt
	AccountTypeBank       = "bank"        // tài khoản ngân hàng
	AccountTypeEWallet    = "ewallet"     // ví điện tử: Momo, ZaloPay...
	AccountTypeCreditCard = "credit_card" // thẻ tín dụng
)

// AccountTypes là tập các loại ví hợp lệ, dùng để kiểm tra dữ liệu đầu vào.
var AccountTypes = map[string]bool{
	AccountTypeCash:       true,
	AccountTypeBank:       true,
	AccountTypeEWallet:    true,
	AccountTypeCreditCard: true,
}

// DefaultCurrency là loại tiền mặc định khi người dùng không chỉ định.
const DefaultCurrency = "VND"

// Loại giao dịch. Giá trị phải khớp với CHECK constraint của bảng
// transactions.
const (
	TransactionTypeIncome   = "income"   // khoản thu
	TransactionTypeExpense  = "expense"  // khoản chi
	TransactionTypeTransfer = "transfer" // chuyển giữa hai ví của chính mình
)

// Loại danh mục.
const (
	CategoryTypeIncome  = "income"
	CategoryTypeExpense = "expense"
)
