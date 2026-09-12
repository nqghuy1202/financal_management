import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import type {
  Budget,
  Category,
  CycleSettingsInput,
  CycleSummary,
  FixedCost,
  Income,
  Settings,
  Transaction,
} from '../types'
import { apiGet, apiSend } from '../lib/api'
import { localizedCategory } from '../lib/i18n'
import { useAuth } from './AuthContext'
import { useI18n } from './I18nContext'

// Response shape of PUT /cycle-settings: the canonical income row for the
// (newly computed) current cycle, plus the canonical settings row.
interface CycleSettingsResult {
  income: Income
  settings: Settings
}

// The backend has no GET for a user's settings row (only PUT /cycle-settings,
// which returns it back) so this mirrors the DB column defaults
// (savings_goal 0, cycle_start_day 1) until the first successful save this
// session populates the real value.
const DEFAULT_SETTINGS: Settings = { savingsGoal: 0, cycleStartDay: 1 }

interface DataContextValue {
  categories: Category[]
  transactions: Transaction[]
  budgets: Budget[]
  cycleSummary: CycleSummary | null
  fixedCosts: FixedCost[]
  settings: Settings
  loading: boolean
  loadError: boolean
  categoryById: (id: string) => Category | undefined

  addTransaction: (t: Omit<Transaction, 'id'>) => Promise<void>
  updateTransaction: (t: Transaction) => Promise<void>
  deleteTransaction: (id: string) => Promise<void>

  addCategory: (c: Omit<Category, 'id'>) => Promise<void>
  deleteCategory: (id: string) => Promise<void>

  upsertBudget: (b: Omit<Budget, 'id'> & { id?: string }) => Promise<void>
  deleteBudget: (id: string) => Promise<void>

  refreshCycleSummary: () => Promise<void>
  saveCycleSettings: (input: CycleSettingsInput) => Promise<void>
  addFixedCost: (fc: Omit<FixedCost, 'id'>) => Promise<void>
  updateFixedCost: (fc: FixedCost) => Promise<void>
  deleteFixedCost: (id: string) => Promise<void>
}

const DataContext = createContext<DataContextValue | null>(null)

export function DataProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const { lang } = useI18n()
  const [rawCategories, setRawCategories] = useState<Category[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [budgets, setBudgets] = useState<Budget[]>([])
  const [cycleSummary, setCycleSummary] = useState<CycleSummary | null>(null)
  const [fixedCosts, setFixedCosts] = useState<FixedCost[]>([])
  const [settings, setSettings] = useState<Settings>(DEFAULT_SETTINGS)
  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState(false)

  // Localize default category names to the current language; user-created
  // names pass through unchanged. Category logic elsewhere keys off id, not name.
  const categories = useMemo(
    () => rawCategories.map((c) => ({ ...c, name: localizedCategory(c.name, lang) })),
    [rawCategories, lang],
  )

  // Load everything for the signed-in user; clear when signed out.
  useEffect(() => {
    if (!user) {
      setRawCategories([])
      setTransactions([])
      setBudgets([])
      setCycleSummary(null)
      setFixedCosts([])
      setSettings(DEFAULT_SETTINGS)
      return
    }
    let cancelled = false
    setLoading(true)
    setLoadError(false)
    Promise.all([
      apiGet<Category[]>('/categories'),
      apiGet<Transaction[]>('/transactions'),
      apiGet<Budget[]>('/budgets'),
      apiGet<CycleSummary>('/cycle/summary'),
      apiGet<FixedCost[]>('/fixed-costs'),
    ])
      .then(([cats, txs, buds, summary, fcs]) => {
        if (cancelled) return
        setRawCategories(cats)
        setTransactions(txs)
        setBudgets(buds)
        setCycleSummary(summary)
        setFixedCosts(fcs)
      })
      .catch(() => {
        if (!cancelled) {
          setRawCategories([])
          setTransactions([])
          setBudgets([])
          setCycleSummary(null)
          setFixedCosts([])
          setSettings(DEFAULT_SETTINGS)
          setLoadError(true)
        }
      })
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [user])

  const categoryById = useCallback(
    (id: string) => categories.find((c) => c.id === id),
    [categories],
  )

  // Best-effort refetch — used after any mutation that can change
  // Safe-to-spend. Keeps the previous summary on failure rather than
  // clearing the hero out from under the user.
  const refreshCycleSummary = useCallback(async () => {
    try {
      const summary = await apiGet<CycleSummary>('/cycle/summary')
      setCycleSummary(summary)
    } catch {
      // ignore — next successful load will reconcile
    }
  }, [])

  const addTransaction = useCallback(
    async (t: Omit<Transaction, 'id'>) => {
      const created = await apiSend<Transaction>('POST', '/transactions', t)
      setTransactions((prev) => [created, ...prev])
      refreshCycleSummary()
    },
    [refreshCycleSummary],
  )

  const updateTransaction = useCallback(
    async (t: Transaction) => {
      const updated = await apiSend<Transaction>('PUT', `/transactions/${t.id}`, t)
      setTransactions((prev) => prev.map((x) => (x.id === updated.id ? updated : x)))
      refreshCycleSummary()
    },
    [refreshCycleSummary],
  )

  const deleteTransaction = useCallback(
    async (id: string) => {
      await apiSend('DELETE', `/transactions/${id}`)
      setTransactions((prev) => prev.filter((x) => x.id !== id))
      refreshCycleSummary()
    },
    [refreshCycleSummary],
  )

  const addCategory = useCallback(async (c: Omit<Category, 'id'>) => {
    const created = await apiSend<Category>('POST', '/categories', c)
    setRawCategories((prev) => [...prev, created])
  }, [])

  const deleteCategory = useCallback(async (id: string) => {
    await apiSend('DELETE', `/categories/${id}`)
    // Server cascades related budgets; mirror that locally.
    setRawCategories((prev) => prev.filter((c) => c.id !== id))
    setBudgets((prev) => prev.filter((b) => b.categoryId !== id))
  }, [])

  const upsertBudget = useCallback(async (b: Omit<Budget, 'id'> & { id?: string }) => {
    const saved = await apiSend<Budget>('POST', '/budgets', b)
    setBudgets((prev) => {
      const idx = prev.findIndex(
        (x) => x.categoryId === saved.categoryId && x.month === saved.month,
      )
      if (idx >= 0) {
        const next = [...prev]
        next[idx] = saved
        return next
      }
      return [...prev, saved]
    })
  }, [])

  const deleteBudget = useCallback(async (id: string) => {
    await apiSend('DELETE', `/budgets/${id}`)
    setBudgets((prev) => prev.filter((b) => b.id !== id))
  }, [])

  const saveCycleSettings = useCallback(
    async (input: CycleSettingsInput) => {
      const result = await apiSend<CycleSettingsResult>('PUT', '/cycle-settings', input)
      setSettings(result.settings)
      refreshCycleSummary()
    },
    [refreshCycleSummary],
  )

  const addFixedCost = useCallback(
    async (fc: Omit<FixedCost, 'id'>) => {
      const created = await apiSend<FixedCost>('POST', '/fixed-costs', fc)
      setFixedCosts((prev) => [...prev, created])
      refreshCycleSummary()
    },
    [refreshCycleSummary],
  )

  const updateFixedCost = useCallback(
    async (fc: FixedCost) => {
      const updated = await apiSend<FixedCost>('PUT', `/fixed-costs/${fc.id}`, {
        name: fc.name,
        amount: fc.amount,
      })
      setFixedCosts((prev) => prev.map((x) => (x.id === updated.id ? updated : x)))
      refreshCycleSummary()
    },
    [refreshCycleSummary],
  )

  const deleteFixedCost = useCallback(
    async (id: string) => {
      await apiSend('DELETE', `/fixed-costs/${id}`)
      setFixedCosts((prev) => prev.filter((x) => x.id !== id))
      refreshCycleSummary()
    },
    [refreshCycleSummary],
  )

  const value = useMemo<DataContextValue>(
    () => ({
      categories,
      transactions,
      budgets,
      cycleSummary,
      fixedCosts,
      settings,
      loading,
      loadError,
      categoryById,
      addTransaction,
      updateTransaction,
      deleteTransaction,
      addCategory,
      deleteCategory,
      upsertBudget,
      deleteBudget,
      refreshCycleSummary,
      saveCycleSettings,
      addFixedCost,
      updateFixedCost,
      deleteFixedCost,
    }),
    [
      categories,
      transactions,
      budgets,
      cycleSummary,
      fixedCosts,
      settings,
      loading,
      loadError,
      categoryById,
      addTransaction,
      updateTransaction,
      deleteTransaction,
      addCategory,
      deleteCategory,
      upsertBudget,
      deleteBudget,
      refreshCycleSummary,
      saveCycleSettings,
      addFixedCost,
      updateFixedCost,
      deleteFixedCost,
    ],
  )

  return <DataContext.Provider value={value}>{children}</DataContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useData(): DataContextValue {
  const ctx = useContext(DataContext)
  if (!ctx) throw new Error('useData must be used within DataProvider')
  return ctx
}
