import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import type { Budget, Category, Transaction } from '../types'
import { apiGet, apiSend } from '../lib/api'
import { localizedCategory } from '../lib/i18n'
import { useAuth } from './AuthContext'
import { useI18n } from './I18nContext'

interface DataContextValue {
  categories: Category[]
  transactions: Transaction[]
  budgets: Budget[]
  loading: boolean
  categoryById: (id: string) => Category | undefined

  addTransaction: (t: Omit<Transaction, 'id'>) => Promise<void>
  updateTransaction: (t: Transaction) => Promise<void>
  deleteTransaction: (id: string) => Promise<void>

  addCategory: (c: Omit<Category, 'id'>) => Promise<void>
  deleteCategory: (id: string) => Promise<void>

  upsertBudget: (b: Omit<Budget, 'id'> & { id?: string }) => Promise<void>
  deleteBudget: (id: string) => Promise<void>
}

const DataContext = createContext<DataContextValue | null>(null)

export function DataProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const { lang } = useI18n()
  const [rawCategories, setRawCategories] = useState<Category[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [budgets, setBudgets] = useState<Budget[]>([])
  const [loading, setLoading] = useState(false)

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
      return
    }
    let cancelled = false
    setLoading(true)
    Promise.all([
      apiGet<Category[]>('/categories'),
      apiGet<Transaction[]>('/transactions'),
      apiGet<Budget[]>('/budgets'),
    ])
      .then(([cats, txs, buds]) => {
        if (cancelled) return
        setRawCategories(cats)
        setTransactions(txs)
        setBudgets(buds)
      })
      .catch(() => {
        if (!cancelled) {
          setRawCategories([])
          setTransactions([])
          setBudgets([])
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

  const addTransaction = useCallback(async (t: Omit<Transaction, 'id'>) => {
    const created = await apiSend<Transaction>('POST', '/transactions', t)
    setTransactions((prev) => [created, ...prev])
  }, [])

  const updateTransaction = useCallback(async (t: Transaction) => {
    const updated = await apiSend<Transaction>('PUT', `/transactions/${t.id}`, t)
    setTransactions((prev) => prev.map((x) => (x.id === updated.id ? updated : x)))
  }, [])

  const deleteTransaction = useCallback(async (id: string) => {
    await apiSend('DELETE', `/transactions/${id}`)
    setTransactions((prev) => prev.filter((x) => x.id !== id))
  }, [])

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

  const value = useMemo<DataContextValue>(
    () => ({
      categories,
      transactions,
      budgets,
      loading,
      categoryById,
      addTransaction,
      updateTransaction,
      deleteTransaction,
      addCategory,
      deleteCategory,
      upsertBudget,
      deleteBudget,
    }),
    [
      categories,
      transactions,
      budgets,
      loading,
      categoryById,
      addTransaction,
      updateTransaction,
      deleteTransaction,
      addCategory,
      deleteCategory,
      upsertBudget,
      deleteBudget,
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
