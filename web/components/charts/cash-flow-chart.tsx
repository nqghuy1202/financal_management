"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { MonthlyFlow } from "@/lib/api/types";
import { formatAmount, formatCompact } from "@/lib/money";
import { formatMonthLabel } from "@/lib/time";

/**
 * Thu và chi từng tháng, hai cột đứng cạnh nhau.
 *
 * Vì sao cột nhóm chứ không phải cột chồng: câu hỏi người dùng đặt ra là
 * "tháng này tôi chi nhiều hơn hay ít hơn thu?", tức là so sánh hai giá
 * trị với nhau. Cột chồng trả lời câu hỏi khác — tổng của hai thứ — mà
 * tổng của thu và chi thì không có ý nghĩa gì.
 *
 * Vì sao không có trục thứ hai cho số dư ròng: hai thang đo trên một
 * biểu đồ khiến chỗ cắt nhau của hai đường trông như một mối liên hệ,
 * trong khi nó chỉ là hệ quả của việc ta chọn thang. Số dư ròng đã có ở
 * hàng thẻ KPI phía trên.
 */
export function CashFlowChart({ data }: { data: MonthlyFlow[] }) {
  const rows = data.map((flow) => ({
    month: formatMonthLabel(flow.month),
    // Recharts cần number để tính chiều cao cột. Chấp nhận được: đây là
    // toạ độ vẽ, còn con số người dùng đọc trong tooltip vẫn lấy từ
    // chuỗi gốc bên dưới.
    income: Number(flow.income),
    expense: Number(flow.expense),
    raw: flow,
  }));

  return (
    <div>
      <Legend />
      <div className="h-64 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={rows} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barGap={2}>
            {/* Lưới chỉ kẻ ngang và là nét liền mảnh: nét đứt đọc như một
                ngưỡng cảnh báo, trong khi đây chỉ là vạch chia. */}
            <CartesianGrid
              vertical={false}
              stroke="var(--chart-grid)"
              strokeDasharray="0"
            />
            <XAxis
              dataKey="month"
              tickLine={false}
              axisLine={{ stroke: "var(--chart-axis)" }}
              tick={{ fill: "var(--fg-faint)", fontSize: 11 }}
              interval="preserveStartEnd"
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              width={56}
              tick={{ fill: "var(--fg-faint)", fontSize: 11 }}
              tickFormatter={(value: number) => formatCompact(String(value))}
            />
            <Tooltip
              cursor={{ fill: "var(--surface-2)" }}
              content={<FlowTooltip />}
            />
            {/* maxBarSize giữ cột mảnh: cột lấp đầy ô nhìn nặng và ồn.
                radius bo 4px ở đầu cột, vuông ở chân để cột vẫn mọc lên
                từ một đường nền chung. */}
            <Bar dataKey="income" fill="var(--chart-1)" maxBarSize={20} radius={[4, 4, 0, 0]} />
            <Bar dataKey="expense" fill="var(--chart-2)" maxBarSize={20} radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

/**
 * Chú giải tự viết thay vì <Legend> của Recharts.
 *
 * Chữ trong chú giải mang màu chữ bình thường, chỉ ô vuông nhỏ bên cạnh
 * mang màu của cột. Tô màu chính chữ sẽ khiến nó khó đọc trên nền, mà
 * danh tính thì ô vuông đã nói rồi.
 */
function Legend() {
  return (
    <div className="mb-3 flex items-center gap-4 text-xs text-fg-muted">
      <LegendItem color="var(--chart-1)" label="Thu" />
      <LegendItem color="var(--chart-2)" label="Chi" />
    </div>
  );
}

function LegendItem({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span
        aria-hidden
        className="inline-block h-2.5 w-2.5 rounded-sm"
        style={{ background: color }}
      />
      {label}
    </span>
  );
}

type TooltipPayload = { payload: { raw: MonthlyFlow } };

function FlowTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: TooltipPayload[];
}) {
  if (!active || !payload?.length) return null;
  const flow = payload[0].payload.raw;

  return (
    <div className="rounded-lg border border-line bg-surface px-3 py-2 text-xs shadow-lg">
      <p className="mb-1.5 font-medium text-fg">{formatMonthLabel(flow.month)}</p>
      <dl className="space-y-1 tabular">
        <Row color="var(--chart-1)" label="Thu" value={flow.income} />
        <Row color="var(--chart-2)" label="Chi" value={flow.expense} />
        <div className="flex items-center justify-between gap-6 border-t border-line pt-1 text-fg">
          <dt>Còn lại</dt>
          <dd>{formatAmount(flow.net)}</dd>
        </div>
      </dl>
    </div>
  );
}

function Row({ color, label, value }: { color: string; label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-6 text-fg-muted">
      <dt className="flex items-center gap-1.5">
        <span
          aria-hidden
          className="inline-block h-2 w-2 rounded-sm"
          style={{ background: color }}
        />
        {label}
      </dt>
      <dd className="text-fg">{formatAmount(value)}</dd>
    </div>
  );
}
