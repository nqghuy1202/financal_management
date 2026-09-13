export type TransactionType = 'income' | 'expense'

export interface Category {
  id: string
  name: string
  type: TransactionType
  color: string
  icon: string
}

export interface Transaction {
  id: string
  type: TransactionType
  amount: number
  categoryId: string
  note: string
  date: string // ISO yyyy-mm-dd
}

export interface Budget {
  id: string
  categoryId: string
  limit: number
  month: string // yyyy-mm
}

export interface User {
  id: string
  name: string
  email: string
}

// A user's declared income for one budget cycle.
export interface Income {
  id: string
  cycleStartDate: string // yyyy-mm-dd
  amount: number
}

// A recurring committed cost (rent, subscriptions, ...) — a live list, not
// snapshotted per cycle.
export interface FixedCost {
  id: string
  name: string
  amount: number
}

// A user's 1:1 account settings.
export interface Settings {
  savingsGoal: number
  cycleStartDay: number
}

// Placeholder shape for the always-empty `budgets` array GET /cycle/summary
// ships today (budgets[].status is Story 2.3's job) — nothing concrete to
// type yet, matching the backend's `[]any{}`.
export type BudgetStatus = Record<string, unknown>

// One undismissed budget-threshold alert (Story 2.2). Fixed contract from
// the architecture: no percent/spent figure, just enough to render banner
// copy — the frontend resolves categoryId -> display name locally.
export interface ActiveAlert {
  categoryId: string
  threshold: 70 | 90 | 100
  status: 'near' | 'over'
}

// Response shape of GET /cycle/summary.
export interface CycleSummary {
  safeToSpend: number
  daysRemaining: number
  income: number | null
  previousIncome: number | null
  budgets: BudgetStatus[]
  activeAlerts: ActiveAlert[]
}

// Request body of PUT /cycle-settings — the sheet's one Save action.
export interface CycleSettingsInput {
  income: number
  savingsGoal: number
  cycleStartDay: number
}
