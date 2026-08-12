import clsx from "clsx";
import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";
import { formatPercent } from "@/lib/money";

/**
 * Một con số kèm mức thay đổi so với kỳ trước.
 *
 * Bốn con số tổng quan là bốn thẻ như thế này chứ không phải một biểu đồ
 * cột bốn cột: chúng không cùng đơn vị (ba con số tiền và một tỷ lệ phần
 * trăm), nên đặt chung một trục là vô nghĩa.
 */
export function StatTile({
  label,
  value,
  changePct,
  upIsGood,
  hint,
}: {
  label: string;
  value: string;
  /** null khi kỳ trước không có số liệu để so sánh. */
  changePct?: string | null;
  /** Thu tăng là tốt, chi tăng là xấu — cùng một mũi tên, hai ý nghĩa. */
  upIsGood?: boolean;
  hint?: string;
}) {
  return (
    <div className="rounded-xl border border-line bg-surface p-4">
      <p className="text-sm text-fg-muted">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-fg">{value}</p>
      {changePct !== undefined && (
        <Delta changePct={changePct} upIsGood={upIsGood ?? true} />
      )}
      {hint && <p className="mt-1 text-xs text-fg-faint">{hint}</p>}
    </div>
  );
}

function Delta({ changePct, upIsGood }: { changePct: string | null; upIsGood: boolean }) {
  if (changePct === null) {
    return <p className="mt-1.5 text-xs text-fg-faint">Kỳ trước chưa có số liệu</p>;
  }

  const value = Number(changePct);
  const flat = Math.abs(value) < 0.05;
  const good = flat ? null : value > 0 === upIsGood;

  // Mũi tên mang hướng, chữ mang con số — màu chỉ là lớp nhấn thêm. Nhờ
  // vậy người không phân biệt được xanh với đỏ vẫn đọc đúng.
  const Icon = flat ? Minus : value > 0 ? ArrowUpRight : ArrowDownRight;

  return (
    <p
      className={clsx(
        "mt-1.5 flex items-center gap-1 text-xs",
        good === null && "text-fg-muted",
        good === true && "text-income",
        good === false && "text-expense",
      )}
    >
      <Icon size={14} aria-hidden />
      {flat ? "Gần như không đổi" : `${formatPercent(changePct)} so với kỳ trước`}
    </p>
  );
}
