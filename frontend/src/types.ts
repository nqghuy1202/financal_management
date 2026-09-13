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

// spent/percent/status are computed fresh by the backend on every
// GET/POST /budgets (Story 2.3) — the Budgets page and Dashboard both read
// these directly instead of recomputing locally, so the two surfaces can
// never disagree.
export interface Budget {
  id: string
  categoryId: string
  limit: number
  month: string // yyyy-mm
  spent: number
  percent: number
  status: 'within' | 'near' | 'over'
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

// Response shape of GET /cycle/summary. savingsGoal/cycleStartDay are folded
// in here rather than served from a separate GET /settings (which doesn't
// exist) — see DataContext's initial load.
export interface CycleSummary {
  safeToSpend: number
  daysRemaining: number
  income: number | null
  previousIncome: number | null
  budgets: BudgetStatus[]
  activeAlerts: ActiveAlert[]
  savingsGoal: number
  cycleStartDay: number
  // currentMonth is the "yyyy-mm" budgets.month value for the current cycle
  // (AD-4) — use this instead of today's own calendar month wherever
  // "current" budgets/cycle are being matched, since cycleStartDay other
  // than 1 can make them differ.
  currentMonth: string
}

// Request body of PUT /cycle-settings — the sheet's one Save action.
export interface CycleSettingsInput {
  income: number
  savingsGoal: number
  cycleStartDay: number
}
