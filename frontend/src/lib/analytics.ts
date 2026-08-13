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
  remaining: number
}

export function budgetProgress(
  budgets: Budget[],
  txs: Transaction[],
  categories: Category[],
  month: string,
): BudgetProgress[] {
  const monthTxs = filterByMonth(txs, month).filter((t) => t.type === 'expense')
  return budgets
    .filter((b) => b.month === month)
    .map((budget) => {
      const spent = monthTxs
        .filter((t) => t.categoryId === budget.categoryId)
        .reduce((s, t) => s + t.amount, 0)
      const category =
        categories.find((c) => c.id === budget.categoryId) ??
        ({ id: budget.categoryId, name: 'Khác', type: 'expense', color: '#64748b', icon: 'MoreHorizontal' } as Category)
      return {
        budget,
        category,
        spent,
        percent: budget.limit ? (spent / budget.limit) * 100 : 0,
        remaining: budget.limit - spent,
      }
    })
    .sort((a, b) => b.percent - a.percent)
}
