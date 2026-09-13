import type { Budget, Category, Transaction } from '../types'
import { monthOf } from './format'

export interface CategoryBreakdown {
  category: Category
  total: number
  percent: number
}

export interface MonthlyPoint {
  month: string // yyyy-mm
  label: string // "T7"
  income: number
  expense: number
}

export function sumByType(txs: Transaction[], type: Transaction['type']): number {
  return txs.filter((t) => t.type === type).reduce((s, t) => s + t.amount, 0)
}

export function filterByMonth(txs: Transaction[], month: string): Transaction[] {
  return txs.filter((t) => monthOf(t.date) === month)
}

export function expenseBreakdown(
  txs: Transaction[],
  categories: Category[],
): CategoryBreakdown[] {
  const expenses = txs.filter((t) => t.type === 'expense')
  const total = expenses.reduce((s, t) => s + t.amount, 0)
  const map = new Map<string, number>()
  for (const t of expenses) {
    map.set(t.categoryId, (map.get(t.categoryId) ?? 0) + t.amount)
  }
  return [...map.entries()]
    .map(([categoryId, amount]) => {
      const category =
        categories.find((c) => c.id === categoryId) ??
        ({ id: categoryId, name: 'Khác', type: 'expense', color: '#64748b', icon: 'MoreHorizontal' } as Category)
      return { category, total: amount, percent: total ? (amount / total) * 100 : 0 }
    })
    .sort((a, b) => b.total - a.total)
}

export function monthlyTrend(txs: Transaction[], months = 6): MonthlyPoint[] {
  const points: MonthlyPoint[] = []
  const now = new Date()
  for (let i = months - 1; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const month = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
    const monthTxs = filterByMonth(txs, month)
    points.push({
      month,
      label: `T${d.getMonth() + 1}`,
      income: sumByType(monthTxs, 'income'),
      expense: sumByType(monthTxs, 'expense'),
    })
  }
  return points
}

export interface BudgetProgress {
  budget: Budget
  category: Category
  spent: number
  percent: number
  status: Budget['status']
  remaining: number
}

// budgetProgress pairs each of the current cycle's budgets with its
// category for display. spent/percent/status come straight from the
// backend (Story 2.3, GET /budgets) — never recomputed here — so the
// Budgets page and Dashboard, which both call this, can never disagree with
// each other or with the server. `month` should be cycleSummary.currentMonth
// (AD-4), not the wall-clock's own calendar month.
export function budgetProgress(budgets: Budget[], categories: Category[], month: string): BudgetProgress[] {
  return budgets
    .filter((b) => b.month === month)
    .map((budget) => {
      const category =
        categories.find((c) => c.id === budget.categoryId) ??
        ({ id: budget.categoryId, name: 'Khác', type: 'expense', color: '#64748b', icon: 'MoreHorizontal' } as Category)
      return {
        budget,
        category,
        spent: budget.spent,
        percent: budget.percent,
        status: budget.status,
        remaining: budget.limit - budget.spent,
      }
    })
    .sort((a, b) => b.percent - a.percent)
}
