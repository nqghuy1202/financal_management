"use client";

import { useState } from "react";
import clsx from "clsx";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { currentYearMonth, monthPeriod, periodLabel, yearPeriod, type Period } from "@/lib/time";

type Mode = "month" | "year";

export type PeriodSelection = {
  period: Period;
  mode: Mode;
  label: string;
};

/**
 * Chọn kỳ xem báo cáo: một tháng hoặc cả năm, kèm nút lùi/tiến.
 *
 * Kỳ luôn là khoảng nửa mở [from, to) do lib/time dựng ra, nên hai kỳ
 * liền nhau không bao giờ đếm trùng giao dịch nằm đúng lúc giao thời.
 */
export function usePeriodSelection(): PeriodSelection & {
  mode: Mode;
  setMode: (mode: Mode) => void;
  shift: (delta: number) => void;
} {
  const now = currentYearMonth();
  const [mode, setMode] = useState<Mode>("month");
  const [year, setYear] = useState(now.year);
  const [month, setMonth] = useState(now.month);

  const period = mode === "month" ? monthPeriod(year, month) : yearPeriod(year);

  function shift(delta: number) {
    if (mode === "year") {
      setYear((y) => y + delta);
      return;
    }
    // Dồn tháng về khoảng 1–12 và mượn/trả sang năm. Dùng Date để không
    // phải tự viết phép chia lấy dư cho số âm.
    const shifted = new Date(Date.UTC(year, month - 1 + delta, 1));
    setYear(shifted.getUTCFullYear());
    setMonth(shifted.getUTCMonth() + 1);
  }

  return { period, mode, setMode, shift, label: periodLabel(period) };
}

export function PeriodPicker({
  mode,
  setMode,
  shift,
  label,
}: {
  mode: Mode;
  setMode: (mode: Mode) => void;
  shift: (delta: number) => void;
  label: string;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="flex items-center gap-1 rounded-lg border border-line bg-surface p-0.5">
        <ModeButton active={mode === "month"} onClick={() => setMode("month")}>
          Tháng
        </ModeButton>
        <ModeButton active={mode === "year"} onClick={() => setMode("year")}>
          Năm
        </ModeButton>
      </div>

      <div className="flex items-center gap-1 rounded-lg border border-line bg-surface px-1">
        <IconButton onClick={() => shift(-1)} label="Kỳ trước">
          <ChevronLeft size={16} />
        </IconButton>
        <span className="min-w-32 text-center text-sm font-medium">{label}</span>
        <IconButton onClick={() => shift(1)} label="Kỳ sau">
          <ChevronRight size={16} />
        </IconButton>
      </div>
    </div>
  );
}

function ModeButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={clsx(
        "rounded-md px-3 py-1 text-sm transition-colors",
        active ? "bg-brand-soft font-medium text-brand-on-soft" : "text-fg-muted hover:text-fg",
      )}
    >
      {children}
    </button>
  );
}

function IconButton({
  onClick,
  label,
  children,
}: {
  onClick: () => void;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      className="rounded-md p-1.5 text-fg-muted hover:bg-surface-2 hover:text-fg"
    >
      {children}
    </button>
  );
}
