"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { getByCategory, getCashFlow, getSummary } from "@/lib/api/reports";
import { listTransactions } from "@/lib/api/transactions";
import { formatAmount, formatPercent, isNegative } from "@/lib/money";
import { trailingMonths } from "@/lib/time";
import { PeriodPicker, usePeriodSelection } from "@/components/period-picker";
import { StatTile } from "@/components/stat-tile";
import { CashFlowChart } from "@/components/charts/cash-flow-chart";
import { CategoryBreakdown } from "@/components/charts/category-breakdown";
import { TransactionRow } from "@/components/transactions/transaction-row";
import { Card, CardTitle } from "@/components/ui/card";
import { EmptyState, ErrorState, Skeleton } from "@/components/ui/states";

/** Biểu đồ dòng tiền luôn nhìn 12 tháng gần nhất, không phụ thuộc kỳ đang chọn. */
const CASH_FLOW_MONTHS = 12;

export default function DashboardPage() {
  const selection = usePeriodSelection();
  const { period } = selection;

  const summary = useQuery({
    queryKey: ["summary", period],
    queryFn: () => getSummary(period),
  });

  const byCategory = useQuery({
    queryKey: ["by-category", period, "expense"] as const,
    queryFn: () => getByCategory(period, "expense"),
  });

  const cashFlowPeriod = trailingMonths(CASH_FLOW_MONTHS);
  const cashFlow = useQuery({
    queryKey: ["cash-flow", cashFlowPeriod],
    queryFn: () => getCashFlow(cashFlowPeriod),
  });

  const recent = useQuery({
    queryKey: ["transactions", "recent"],
    queryFn: () => listTransactions({ page_size: 5 }),
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">Tổng quan</h1>
        <PeriodPicker {...selection} />
      </div>

      {/* Bốn con số quan trọng nhất, đọc được ngay không cần diễn giải. */}
      {summary.isPending ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} className="h-28" />
          ))}
        </div>
      ) : summary.isError ? (
        <Card>
          <ErrorState error={summary.error} onRetry={() => summary.refetch()} />
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatTile
            label="Tổng thu"
            value={formatAmount(summary.data.total_income, summary.data.currency)}
            changePct={summary.data.previous_period.income_change_pct}
            upIsGood
          />
          <StatTile
            label="Tổng chi"
            value={formatAmount(summary.data.total_expense, summary.data.currency)}
            changePct={summary.data.previous_period.expense_change_pct}
            upIsGood={false}
          />
          <StatTile
            label="Số dư ròng"
            value={formatAmount(summary.data.net_balance, summary.data.currency)}
            hint={
              isNegative(summary.data.net_balance)
                ? "Kỳ này chi nhiều hơn thu"
                : `${summary.data.transaction_count} giao dịch`
            }
          />
          <StatTile
            label="Tỷ lệ tiết kiệm"
            value={formatPercent(summary.data.savings_rate_pct)}
            hint="Phần thu nhập chưa tiêu"
          />
        </div>
      )}

      <Card>
        <CardTitle>Dòng tiền {CASH_FLOW_MONTHS} tháng gần nhất</CardTitle>
        {cashFlow.isPending ? (
          <Skeleton className="h-64" />
        ) : cashFlow.isError ? (
          <ErrorState error={cashFlow.error} onRetry={() => cashFlow.refetch()} />
        ) : (
          <CashFlowChart data={cashFlow.data.items} />
        )}
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardTitle>Chi tiêu theo danh mục</CardTitle>
          {byCategory.isPending ? (
            <Skeleton className="h-48" />
          ) : byCategory.isError ? (
            <ErrorState error={byCategory.error} onRetry={() => byCategory.refetch()} />
          ) : byCategory.data.items.length === 0 ? (
            <EmptyState
              title="Kỳ này chưa có khoản chi nào"
              description="Ghi lại một khoản chi để thấy tiền của bạn đi đâu."
            />
          ) : (
            <CategoryBreakdown items={byCategory.data.items} />
          )}
        </Card>

        <Card>
          <CardTitle
            action={
              <Link
                href="/transactions"
                className="text-sm font-medium text-brand underline underline-offset-4"
              >
                Xem tất cả
              </Link>
            }
          >
            Giao dịch gần đây
          </CardTitle>
          {recent.isPending ? (
            <Skeleton className="h-48" />
          ) : recent.isError ? (
            <ErrorState error={recent.error} onRetry={() => recent.refetch()} />
          ) : recent.data.items.length === 0 ? (
            <EmptyState
              title="Chưa có giao dịch nào"
              description="Mọi con số ở trên sẽ xuất hiện khi bạn ghi khoản đầu tiên."
            />
          ) : (
            <ul className="divide-y divide-line">
              {recent.data.items.map((tx) => (
                <li key={tx.id}>
                  <TransactionRow transaction={tx} />
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </div>
  );
}
