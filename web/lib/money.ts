import type { Money } from "./api/types";

/**
 * Định dạng số tiền mà không đi qua kiểu number.
 *
 * Backend trả số tiền dưới dạng chuỗi vì JavaScript đọc số JSON thành
 * float64 và làm tròn sai với số lớn. Nếu ở đây ta gọi Number(amount) thì
 * công sức đó đổ sông đổ biển ngay tại bước cuối cùng.
 *
 * Intl.NumberFormat nhận thẳng chuỗi thập phân (Intl NumberFormat v3) và
 * định dạng từng chữ số như nó vốn có, nên đường đi của một số tiền từ
 * PostgreSQL tới màn hình không có chỗ nào chạm vào số thực.
 */

const VND = "VND";

function formatterFor(currency: string): Intl.NumberFormat {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency,
    // Đồng Việt Nam không tiêu đơn vị lẻ, hiện phần thập phân chỉ gây rối.
    maximumFractionDigits: currency === VND ? 0 : 2,
  });
}

// Dựng bộ định dạng khá tốn, mà bảng giao dịch gọi nó cho từng dòng.
const cache = new Map<string, Intl.NumberFormat>();

function cachedFormatter(currency: string): Intl.NumberFormat {
  let formatter = cache.get(currency);
  if (!formatter) {
    formatter = formatterFor(currency);
    cache.set(currency, formatter);
  }
  return formatter;
}

/**
 * "125000.5" + "VND" → "125.001 ₫"
 *
 * Ép kiểu vì TypeScript chỉ chấp nhận chuỗi số ở dạng hằng (`${number}`),
 * mà số tiền của ta tới từ mạng nên không kiểm tra tĩnh được.
 */
export function formatAmount(amount: string, currency = VND): string {
  return cachedFormatter(currency).format(amount as Intl.StringNumericLiteral);
}

export function formatMoney(money: Money): string {
  return formatAmount(money.amount, money.currency);
}

/** Thêm dấu + hoặc − ở đầu, dùng cho số dư ròng và mức thay đổi. */
export function formatSigned(amount: string, currency = VND): string {
  const text = formatAmount(amount, currency);
  return isNegative(amount) ? text : `+${text}`;
}

export function isNegative(amount: string): boolean {
  return amount.trimStart().startsWith("-");
}

export function isZero(amount: string): boolean {
  // "0", "0.0000", "-0.00" đều là không. Bỏ dấu và mọi số 0 rồi xem còn
  // lại gì ngoài dấu chấm.
  return /^-?0*\.?0*$/.test(amount.trim());
}

/**
 * Dạng rút gọn cho trục biểu đồ: "12,5 Tr", "1,2 Tỷ".
 *
 * Ở đây có đổi sang number, và điều đó chấp nhận được: nhãn trục vốn đã
 * là số làm tròn, sai lệch ở chữ số thứ mười sáu không ai nhìn thấy. Với
 * số tiền hiển thị chính xác thì vẫn phải dùng formatAmount.
 */
export function formatCompact(amount: string): string {
  const value = Number(amount);
  const abs = Math.abs(value);
  const sign = value < 0 ? "-" : "";

  if (abs >= 1e9) return `${sign}${trim(abs / 1e9)} Tỷ`;
  if (abs >= 1e6) return `${sign}${trim(abs / 1e6)} Tr`;
  if (abs >= 1e3) return `${sign}${trim(abs / 1e3)} N`;
  return `${sign}${abs}`;
}

function trim(value: number): string {
  return value.toFixed(1).replace(/\.0$/, "").replace(".", ",");
}

/** "23.456" → "23,5%" */
export function formatPercent(value: string, digits = 1): string {
  return `${Number(value).toFixed(digits).replace(".", ",")}%`;
}
