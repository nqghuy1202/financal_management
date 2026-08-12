/**
 * Mọi thứ liên quan tới thời gian đều neo vào MÚI GIỜ ỨNG DỤNG, không
 * phải múi giờ của trình duyệt.
 *
 * Backend nhóm báo cáo theo tháng bằng `model.AppTimezone` và hiểu tham
 * số dạng ngày ("2026-08-01") theo múi giờ đó. Nếu frontend dựng khoảng
 * thời gian theo múi giờ máy người dùng, một người mở app ở Nhật (+09)
 * sẽ hỏi backend một kỳ lệch mất hai tiếng: các giao dịch từ 00:00 tới
 * 02:00 ngày đầu tháng rơi sang tháng trước, mà tổng vẫn trông hợp lý
 * nên không ai phát hiện ra.
 */
export const APP_TIMEZONE = "Asia/Ho_Chi_Minh";

/**
 * Việt Nam không có giờ mùa hè, nên offset cố định quanh năm. Nhờ vậy
 * ghép chuỗi RFC3339 bằng tay là an toàn — với múi giờ có DST thì không.
 */
export const APP_UTC_OFFSET = "+07:00";

/** Bộ định dạng tách ngày ra thành các phần, luôn theo múi giờ ứng dụng. */
const partsFormatter = new Intl.DateTimeFormat("en-CA", {
  timeZone: APP_TIMEZONE,
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

type DateParts = { year: string; month: string; day: string; hour: string; minute: string };

function partsOf(date: Date): DateParts {
  const parts: Record<string, string> = {};
  for (const part of partsFormatter.formatToParts(date)) {
    parts[part.type] = part.value;
  }
  // "24" là cách en-CA viết nửa đêm; đổi về "00" cho đúng chuẩn.
  if (parts.hour === "24") parts.hour = "00";
  return parts as unknown as DateParts;
}

/** Ngày hôm nay theo múi giờ ứng dụng, dạng "2026-08-12". */
export function todayISO(): string {
  const { year, month, day } = partsOf(new Date());
  return `${year}-${month}-${day}`;
}

/** Thời điểm hiện tại theo múi giờ ứng dụng, dạng "2026-08-12T14:30". */
export function nowLocalInput(): string {
  const { year, month, day, hour, minute } = partsOf(new Date());
  return `${year}-${month}-${day}T${hour}:${minute}`;
}

/**
 * Đổi giá trị của <input type="datetime-local"> sang RFC3339.
 *
 * Input trả về "2026-08-12T14:30" không kèm múi giờ. Gửi thẳng chuỗi đó
 * lên thì backend không parse được; còn dùng `new Date(value).toISOString()`
 * thì trình duyệt hiểu theo múi giờ máy, tức người dùng ở múi giờ khác sẽ
 * ghi nhầm giờ. Ta gắn tường minh offset của múi giờ ứng dụng.
 */
export function inputToRFC3339(value: string): string {
  const withSeconds = value.length === 16 ? `${value}:00` : value;
  return `${withSeconds}${APP_UTC_OFFSET}`;
}

/** Đổi RFC3339 từ API về giá trị cho <input type="datetime-local">. */
export function rfc3339ToInput(value: string): string {
  const { year, month, day, hour, minute } = partsOf(new Date(value));
  return `${year}-${month}-${day}T${hour}:${minute}`;
}

// ---------------------------------------------------------------------
// Khoảng thời gian nửa mở [from, to)
// ---------------------------------------------------------------------

export type Period = { from: string; to: string };

function iso(year: number, monthIndex: number, day: number): string {
  // Date.UTC tự xử lý tràn tháng: monthIndex 12 thành tháng 1 năm sau.
  const d = new Date(Date.UTC(year, monthIndex, day));
  return d.toISOString().slice(0, 10);
}

/** Kỳ của một tháng. `month` đếm từ 1. */
export function monthPeriod(year: number, month: number): Period {
  return { from: iso(year, month - 1, 1), to: iso(year, month, 1) };
}

/** Kỳ của cả một năm. */
export function yearPeriod(year: number): Period {
  return { from: iso(year, 0, 1), to: iso(year + 1, 0, 1) };
}

/** Tháng hiện tại theo múi giờ ứng dụng, dạng { year, month } với month từ 1. */
export function currentYearMonth(): { year: number; month: number } {
  const { year, month } = partsOf(new Date());
  return { year: Number(year), month: Number(month) };
}

/**
 * Mười hai tháng gần nhất, kết thúc ở hết tháng hiện tại.
 * Dùng cho biểu đồ dòng tiền.
 */
export function trailingMonths(count: number): Period {
  const { year, month } = currentYearMonth();
  return {
    from: iso(year, month - count, 1),
    to: iso(year, month, 1),
  };
}

// ---------------------------------------------------------------------
// Hiển thị
// ---------------------------------------------------------------------

const dayFormatter = new Intl.DateTimeFormat("vi-VN", {
  timeZone: APP_TIMEZONE,
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
});

const dayTimeFormatter = new Intl.DateTimeFormat("vi-VN", {
  timeZone: APP_TIMEZONE,
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

export function formatDate(value: string): string {
  return dayFormatter.format(new Date(value));
}

export function formatDateTime(value: string): string {
  return dayTimeFormatter.format(new Date(value));
}

/** "2026-08" → "Th08/2026". Dùng cho nhãn biểu đồ và tiêu đề kỳ. */
export function formatMonthLabel(month: string): string {
  const [year, m] = month.split("-");
  return `Th${m}/${year}`;
}

/**
 * Tên kỳ để hiển thị, suy từ khoảng [from, to).
 *
 * Lấy tháng của THỜI ĐIỂM CUỐI CÙNG NẰM TRONG kỳ, không phải tháng của
 * `to`: kỳ [2026-08-01, 2026-09-01) là tháng 8, nhưng `to` lại là tháng 9.
 */
export function periodLabel(period: Period): string {
  const from = new Date(`${period.from}T00:00:00${APP_UTC_OFFSET}`);
  const to = new Date(`${period.to}T00:00:00${APP_UTC_OFFSET}`);
  const lastInstant = new Date(to.getTime() - 1);

  const start = partsOf(from);
  const end = partsOf(lastInstant);

  if (start.year === end.year && start.month === end.month) {
    return `Tháng ${Number(start.month)}/${start.year}`;
  }
  if (start.year === end.year && start.month === "01" && end.month === "12") {
    return `Năm ${start.year}`;
  }
  return `${formatDate(from.toISOString())} – ${formatDate(lastInstant.toISOString())}`;
}
