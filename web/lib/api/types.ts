/**
 * Kiểu dữ liệu khớp với response của Go API.
 *
 * Mọi SỐ TIỀN đều là chuỗi, không phải number. Đây là quyết định của
 * backend và frontend phải tôn trọng: JSON.parse đọc số thành float64,
 * nên một số tiền lớn bị làm tròn sai ngay trước khi ta kịp nhìn thấy nó.
 * Muốn hiển thị thì dùng các hàm trong lib/format.ts, đừng gọi Number().
 */

/** Envelope chung của mọi response, xem internal/pkg/response/response.go */
export type Envelope<T> = {
  code: number;
  message: string;
  data?: T;
  request_id?: string;
};

export type ListOf<T> = { items: T[] };

// ---------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------

export type User = {
  id: string;
  email: string;
  full_name: string;
  base_currency: string;
  created_at: string;
};

/**
 * Cố tình không có refresh_token: nó nằm trong cookie HttpOnly nên
 * JavaScript không đọc được, và đó chính là mục đích.
 */
export type Session = {
  access_token: string;
  token_type: string;
  /** Số giây access token còn hiệu lực. */
  expires_in: number;
  user: User;
};

// ---------------------------------------------------------------------
// Nguồn tiền và danh mục
// ---------------------------------------------------------------------

export const ACCOUNT_TYPES = ["cash", "bank", "ewallet", "credit_card"] as const;
export type AccountType = (typeof ACCOUNT_TYPES)[number];

export type Account = {
  id: string;
  name: string;
  type: AccountType;
  icon: string;
  created_at: string;
};

export type TransactionType = "income" | "expense";

export type Category = {
  id: string;
  name: string;
  type: TransactionType;
  icon: string;
  color: string;
  /** Danh mục hệ thống thì không sửa hay xoá được — ẩn nút thay vì báo lỗi. */
  is_system: boolean;
};

// ---------------------------------------------------------------------
// Giao dịch
// ---------------------------------------------------------------------

/** model.Money của backend: số tiền luôn đi kèm loại tiền. */
export type Money = {
  amount: string;
  currency: string;
};

export type Transaction = {
  id: string;
  type: TransactionType;
  amount: Money;
  category_id: string | null;
  account_id: string | null;
  note: string;
  occurred_at: string;
  created_at: string;
};

export type TransactionPage = {
  items: Transaction[];
  total: number;
  /** Gửi lại nguyên văn ở tham số `cursor` để lấy trang kế. null là đã hết. */
  next_cursor: string | null;
};

export type TransactionInput = {
  type: TransactionType;
  /** Chuỗi thập phân, ví dụ "125000.50". */
  amount: string;
  category_id: string | null;
  account_id: string | null;
  note: string;
  /** RFC3339 kèm offset múi giờ. */
  occurred_at: string;
};

// ---------------------------------------------------------------------
// Báo cáo
// ---------------------------------------------------------------------

export type Summary = {
  from: string;
  to: string;
  currency: string;
  total_income: string;
  total_expense: string;
  net_balance: string;
  savings_rate_pct: string;
  transaction_count: number;
  previous_period: {
    total_income: string;
    total_expense: string;
    net_balance: string;
    /** null khi kỳ trước không có số liệu để so sánh. */
    income_change_pct: string | null;
    expense_change_pct: string | null;
  };
};

export type CategoryTotal = {
  category_id: string | null;
  category_name: string;
  category_icon: string;
  category_color: string;
  total_amount: string;
  percentage: string;
  transaction_count: number;
};

export type ByCategoryReport = {
  items: CategoryTotal[];
  type: TransactionType;
  currency: string;
};

export type MonthlyFlow = {
  /** Dạng "2026-08". */
  month: string;
  income: string;
  expense: string;
  net: string;
};

export type CashFlowReport = {
  items: MonthlyFlow[];
  currency: string;
};
