import { api } from "./client";
import type { ByCategoryReport, CashFlowReport, Summary, TransactionType } from "./types";

/**
 * Khoảng thời gian nửa mở [from, to): tính cả `from`, không tính `to`.
 *
 * Nhờ vậy hai kỳ liền nhau không bao giờ đếm trùng giao dịch nằm đúng
 * ranh giới. Xem lib/period.ts để biết cách dựng khoảng.
 */
export type Period = {
  /** "2026-08-01" */
  from: string;
  /** "2026-09-01" — ngày đầu của kỳ SAU, không phải ngày cuối kỳ này. */
  to: string;
};

export function getSummary(period: Period): Promise<Summary> {
  return api<Summary>("/reports/summary", { query: { ...period } });
}

export function getByCategory(period: Period, type: TransactionType): Promise<ByCategoryReport> {
  return api<ByCategoryReport>("/reports/by-category", { query: { ...period, type } });
}

export function getCashFlow(period: Period): Promise<CashFlowReport> {
  return api<CashFlowReport>("/reports/cash-flow", { query: { ...period } });
}
