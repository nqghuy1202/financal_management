"use client";

import type { CategoryTotal } from "@/lib/api/types";
import { formatAmount, formatPercent } from "@/lib/money";

/**
 * Chi tiêu theo danh mục, dạng thanh ngang xếp từ lớn tới nhỏ.
 *
 * Vì sao không phải biểu đồ tròn: ứng dụng có sẵn 16 danh mục mặc định.
 * Quá bảy mảng màu thì các màu cạnh nhau bắt đầu lẫn vào nhau, và mắt
 * người vốn so sánh chiều dài chính xác hơn nhiều so với so sánh diện
 * tích hình quạt — mà so sánh chính xác đúng là việc cần làm ở đây
 * ("tiền đi đâu nhiều nhất?").
 *
 * Cũng vì thế chỉ dùng MỘT màu cho mọi thanh. Tô mỗi danh mục một màu là
 * mã hoá lại đúng cái mà chiều dài thanh đã nói, tiêu tốn kênh màu mà
 * không thêm thông tin nào.
 */

/** Quá số này thì phần đuôi gộp lại, để danh sách không dài lê thê. */
const MAX_ROWS = 8;

export function CategoryBreakdown({ items }: { items: CategoryTotal[] }) {
  const rows = fold(items);
  // Thanh dài nhất lấp đầy khung, các thanh khác so theo nó — chênh lệch
  // giữa những danh mục nhỏ nhờ vậy vẫn nhìn thấy được.
  const max = Math.max(...rows.map((r) => Number(r.total_amount)), 0);

  return (
    <ul className="space-y-3">
      {rows.map((item) => {
        const width = max > 0 ? (Number(item.total_amount) / max) * 100 : 0;
        return (
          <li key={item.category_id ?? item.category_name} className="space-y-1.5">
            <div className="flex items-baseline justify-between gap-3 text-sm">
              <span className="truncate text-fg">
                {item.category_icon && <span className="mr-1.5">{item.category_icon}</span>}
                {item.category_name}
              </span>
              <span className="shrink-0 tabular text-fg-muted">
                {formatAmount(item.total_amount)}
                <span className="ml-2 text-fg-faint">{formatPercent(item.percentage)}</span>
              </span>
            </div>
            <div className="h-2 w-full rounded-full bg-surface-2">
              <div
                className="h-2 rounded-r-full"
                style={{ width: `${width}%`, background: "var(--chart-1)" }}
              />
            </div>
          </li>
        );
      })}
    </ul>
  );
}

/**
 * Gộp phần đuôi thành một dòng "Khác".
 *
 * Số tiền của dòng gộp là con số người dùng đọc, nên nó được cộng bằng
 * số nguyên lớn chứ không qua số thực. Chiều dài thanh thì dùng Number
 * cũng được — sai lệch ở chữ số thứ mười sáu không đổi một pixel nào.
 */
function fold(items: CategoryTotal[]): CategoryTotal[] {
  if (items.length <= MAX_ROWS) return items;

  const head = items.slice(0, MAX_ROWS - 1);
  const tail = items.slice(MAX_ROWS - 1);

  return [
    ...head,
    {
      category_id: null,
      category_name: `Khác (${tail.length} danh mục)`,
      category_icon: "",
      category_color: "",
      total_amount: sumDecimals(tail.map((t) => t.total_amount)),
      percentage: sumDecimals(tail.map((t) => t.percentage)),
      transaction_count: tail.reduce((sum, t) => sum + t.transaction_count, 0),
    },
  ];
}

/**
 * Cộng các số thập phân dạng chuỗi mà không đi qua float.
 *
 * Cách làm: dời dấu chấm sang phải cho mọi số về cùng số chữ số thập
 * phân, cộng như số nguyên bằng BigInt, rồi trả dấu chấm về chỗ cũ.
 * BigInt không có giới hạn 53 bit của Number, nên tổng luôn đúng từng
 * chữ số dù số tiền lớn tới đâu.
 */
function sumDecimals(values: string[]): string {
  const scale = Math.max(...values.map((v) => (v.split(".")[1] ?? "").length), 0);

  let total = 0n;
  for (const value of values) {
    total += scaled(value, scale);
  }

  return unscale(total, scale);
}

function scaled(value: string, scale: number): bigint {
  const negative = value.startsWith("-");
  const [whole, fraction = ""] = value.replace(/^[-+]/, "").split(".");
  const digits = BigInt(whole + fraction.padEnd(scale, "0").slice(0, scale));
  return negative ? -digits : digits;
}

function unscale(total: bigint, scale: number): string {
  if (scale === 0) return total.toString();

  const sign = total < 0n ? "-" : "";
  const digits = (total < 0n ? -total : total).toString().padStart(scale + 1, "0");
  return `${sign}${digits.slice(0, -scale)}.${digits.slice(-scale)}`;
}
