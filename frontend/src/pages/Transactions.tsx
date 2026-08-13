import { Fragment, useEffect, useMemo, useRef, useState } from 'react'
import {
  Plus,
  Search,
  Pencil,
  Trash2,
  ArrowUpRight,
  ArrowDownRight,
  Filter,
  SlidersHorizontal,
} from 'lucide-react'
import { useData } from '../context/DataContext'
import { useToast } from '../context/ToastContext'
import { useI18n } from '../context/I18nContext'
import type { Transaction } from '../types'
import { TransactionModal } from '../components/TransactionModal'
import { formatCurrency } from '../lib/format'

type GroupBy = 'none' | 'category' | 'type'
type SearchColumn = 'all' | 'content' | 'category'
type ColKey = 'content' | 'category' | 'type'

interface Section {
  key: string
  title: string | null
  rows: Transaction[]
  sum: number // tổng số tiền (luôn dương) trong nhóm
}

const TOGGLE_COLS: { key: ColKey; labelKey: string }[] = [
  { key: 'content', labelKey: 'tx.colContent' },
  { key: 'category', labelKey: 'tx.colCategory' },
  { key: 'type', labelKey: 'tx.colTypeShort' },
]

export function Transactions() {
  const { transactions, categoryById, deleteTransaction } = useData()
  const toast = useToast()
  const { t: tr } = useI18n()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Transaction | null>(null)

  // Table options (all inside one dropdown)
  const [query, setQuery] = useState('')
  const [searchColumn, setSearchColumn] = useState<SearchColumn>('all')
  const [groupBy, setGroupBy] = useState<GroupBy>('none')
  const [visibleCols, setVisibleCols] = useState<Record<ColKey, boolean>>({
    content: true,
    category: true,
    type: true,
  })

  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!menuOpen) return
    const onDown = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false)
    }
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setMenuOpen(false)
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [menuOpen])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return transactions
      .filter((t) => {
        if (!q) return true
        const cat = categoryById(t.categoryId)?.name ?? ''
        if (searchColumn === 'content') return t.note.toLowerCase().includes(q)
        if (searchColumn === 'category') return cat.toLowerCase().includes(q)
        return `${t.note} ${cat}`.toLowerCase().includes(q)
      })
      .sort((a, b) => (a.date < b.date ? 1 : -1))
  }, [transactions, query, searchColumn, categoryById])

  const totalIncome = filtered.filter((t) => t.type === 'income').reduce((s, t) => s + t.amount, 0)
  const totalExpense = filtered.filter((t) => t.type === 'expense').reduce((s, t) => s + t.amount, 0)
  // Tổng cộng = tính theo cột Thu/Chi: Thu cộng vào, Chi trừ ra
  const net = totalIncome - totalExpense

  const sections = useMemo<Section[]>(() => {
    if (groupBy === 'type') {
      const out: Section[] = []
      const income = filtered.filter((t) => t.type === 'income')
      const expense = filtered.filter((t) => t.type === 'expense')
      if (income.length)
        out.push({ key: 'income', title: tr('type.incomeFull'), rows: income, sum: income.reduce((s, t) => s + t.amount, 0) })
      if (expense.length)
        out.push({ key: 'expense', title: tr('type.expenseFull'), rows: expense, sum: expense.reduce((s, t) => s + t.amount, 0) })
      return out
    }
    if (groupBy === 'category') {
      const map = new Map<string, Transaction[]>()
      for (const t of filtered) {
        const arr = map.get(t.categoryId) ?? []
        arr.push(t)
        map.set(t.categoryId, arr)
      }
      return [...map.entries()]
        .map(([catId, rows]) => ({
          key: catId,
          title: categoryById(catId)?.name ?? 'Khác',
          rows,
          sum: rows.reduce((s, t) => s + t.amount, 0),
        }))
        .sort((a, b) => b.sum - a.sum)
    }
    return [{ key: 'all', title: null, rows: filtered, sum: totalIncome + totalExpense }]
  }, [filtered, groupBy, categoryById, totalIncome, totalExpense, tr])

  const openAdd = () => {
    setEditing(null)
    setModalOpen(true)
  }
  const openEdit = (t: Transaction) => {
    setEditing(t)
    setModalOpen(true)
  }
  const toggleCol = (key: ColKey) => setVisibleCols((v) => ({ ...v, [key]: !v[key] }))
  const handleDelete = async (t: Transaction) => {
    try {
      await deleteTransaction(t.id)
      toast.success(tr('toast.txDeleted'))
    } catch (e) {
      toast.error(e instanceof Error ? e.message : tr('toast.txDeleteError'))
    }
  }

  // Layout helpers for group header / subtotal / footer colSpans
  const leftCount = TOGGLE_COLS.filter((c) => visibleCols[c.key]).length // cols before "Số tiền"
  const totalCols = leftCount + 2 // + Số tiền + Thao tác
  const optionsActive =
    !!query || groupBy !== 'none' || Object.values(visibleCols).some((v) => !v)

  const renderRow = (t: Transaction) => {
    const cat = categoryById(t.categoryId)
    const isIncome = t.type === 'income'
    return (
      <tr key={t.id} className="border-b border-ink-100 last:border-0 hover:bg-ink-50/60">
        {visibleCols.content && <td className="px-4 py-3 text-ink-800">{t.note || cat?.name || '—'}</td>}
        {visibleCols.category && <td className="px-4 py-3 text-ink-500">{cat?.name ?? '—'}</td>}
        {visibleCols.type && (
          <td className="px-4 py-3">
            <span className={`chip ${isIncome ? 'bg-brand-50 text-brand-700' : 'bg-rose-50 text-rose-600'}`}>
              {isIncome ? tr('type.income') : tr('type.expense')}
            </span>
          </td>
        )}
        <td className={`tnum px-4 py-3 text-right font-semibold ${isIncome ? 'text-brand-700' : 'text-rose-600'}`}>
          {formatCurrency(t.amount)}
        </td>
        <td className="px-4 py-3">
          <div className="flex items-center justify-end gap-1">
            <button className="btn-icon" onClick={() => openEdit(t)} aria-label={`${tr('common.edit')}: ${t.note || cat?.name}`}>
              <Pencil size={15} />
            </button>
            <button
              className="btn-icon-danger"
              onClick={() => handleDelete(t)}
              aria-label={`${tr('common.delete')}: ${t.note || cat?.name}`}
            >
              <Trash2 size={15} />
            </button>
          </div>
        </td>
      </tr>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink-900">{tr('tx.title')}</h1>
          <p className="mt-1 text-sm text-ink-500">{tr('tx.subtitle')}</p>
        </div>
        <button className="btn-primary" onClick={openAdd}>
          <Plus size={17} /> {tr('tx.add')}
        </button>
      </div>

      {/* Summary strip */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="card flex items-center gap-3 p-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sky-50 text-sky-600">
            <ArrowUpRight size={18} />
          </div>
          <div>
            <p className="text-xs text-ink-500">{tr('tx.totalIncome')}</p>
            <p className="tnum font-bold text-ink-900">{formatCurrency(totalIncome)}</p>
          </div>
        </div>
        <div className="card flex items-center gap-3 p-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-50 text-rose-600">
            <ArrowDownRight size={18} />
          </div>
          <div>
            <p className="text-xs text-ink-500">{tr('tx.totalExpense')}</p>
            <p className="tnum font-bold text-ink-900">{formatCurrency(totalExpense)}</p>
          </div>
        </div>
        <div className="card flex items-center gap-3 p-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
            <Filter size={18} />
          </div>
          <div>
            <p className="text-xs text-ink-500">{tr('tx.count')}</p>
            <p className="tnum font-bold text-ink-900">{filtered.length}</p>
          </div>
        </div>
      </div>

      {/* Toolbar: single action button with options dropdown */}
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold text-ink-700">{tr('tx.listTitle')}</h2>

        <div className="relative" ref={menuRef}>
          <button
            className="btn-outline relative"
            onClick={() => setMenuOpen((v) => !v)}
            aria-haspopup="true"
            aria-expanded={menuOpen}
          >
            <SlidersHorizontal size={16} /> {tr('tx.options')}
            {optionsActive && <span className="absolute -right-1 -top-1 h-2.5 w-2.5 rounded-full bg-brand-500" />}
          </button>

          {menuOpen && (
            <div className="card absolute right-0 z-40 mt-2 w-72 animate-[fadeIn_.12s_ease-out] p-0">
              {/* Search by column */}
              <div className="p-3">
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-400">{tr('tx.searchByColumn')}</p>
                <select
                  className="input"
                  value={searchColumn}
                  onChange={(e) => setSearchColumn(e.target.value as SearchColumn)}
                >
                  <option value="all">{tr('tx.allColumns')}</option>
                  <option value="content">{tr('tx.colContent')}</option>
                  <option value="category">{tr('tx.colCategory')}</option>
                </select>
                <div className="relative mt-2">
                  <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-400" />
                  <input
                    className="input !pl-9"
                    placeholder={tr('tx.searchPlaceholder')}
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                </div>
              </div>

              {/* Group by column */}
              <div className="border-t border-ink-100 p-3">
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-400">{tr('tx.groupByColumn')}</p>
                <div className="space-y-1">
                  {([
                    { k: 'none', label: tr('tx.groupNone') },
                    { k: 'category', label: tr('tx.groupCategory') },
                    { k: 'type', label: tr('tx.groupType') },
                  ] as { k: GroupBy; label: string }[]).map(({ k, label }) => (
                    <button
                      key={k}
                      onClick={() => setGroupBy(k)}
                      className={`flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm font-medium transition ${
                        groupBy === k ? 'bg-brand-50 text-brand-700' : 'text-ink-600 hover:bg-ink-100'
                      }`}
                    >
                      {label}
                      {groupBy === k && <span className="h-2 w-2 rounded-full bg-brand-500" />}
                    </button>
                  ))}
                </div>
              </div>

              {/* Show / hide columns */}
              <div className="border-t border-ink-100 p-3">
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-400">{tr('tx.toggleColumns')}</p>
                <div className="space-y-1">
                  {TOGGLE_COLS.map((c) => (
                    <label
                      key={c.key}
                      className="flex cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-ink-700 hover:bg-ink-100"
                    >
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-ink-300 text-brand-600 focus:ring-brand-500/30"
                        checked={visibleCols[c.key]}
                        onChange={() => toggleCol(c.key)}
                      />
                      {tr(c.labelKey)}
                    </label>
                  ))}
                  <label className="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-ink-400">
                    <input type="checkbox" className="h-4 w-4 rounded border-ink-300" checked disabled />
                    {tr('tx.colAmount')} <span className="text-xs">{tr('tx.amountAlwaysShown')}</span>
                  </label>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Table */}
      {filtered.length === 0 ? (
        <div className="card flex flex-col items-center justify-center gap-2 py-16 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-ink-100 text-ink-400">
            <Search size={22} />
          </div>
          <p className="font-medium text-ink-700">{tr('tx.empty')}</p>
          <p className="text-sm text-ink-400">{tr('tx.emptyHint')}</p>
        </div>
      ) : (
        <div className="card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[560px] text-sm">
              <thead>
                <tr className="border-b border-ink-200 text-left text-xs font-medium uppercase tracking-wide text-ink-400">
                  {visibleCols.content && <th className="px-4 py-3">{tr('tx.colContent')}</th>}
                  {visibleCols.category && <th className="px-4 py-3">{tr('tx.colCategory')}</th>}
                  {visibleCols.type && <th className="px-4 py-3">{tr('tx.colType')}</th>}
                  <th className="px-4 py-3 text-right">{tr('tx.colAmount')}</th>
                  <th className="px-4 py-3 text-right">{tr('tx.colActions')}</th>
                </tr>
              </thead>
              <tbody>
                {sections.map((section) => (
                  <Fragment key={section.key}>
                    {section.title && (
                      <tr className="bg-ink-50">
                        <td colSpan={totalCols} className="px-4 py-2.5 text-xs font-semibold uppercase tracking-wide text-ink-500">
                          {section.title}
                          <span className="ml-2 font-normal text-ink-400">{tr('tx.txCount', { n: section.rows.length })}</span>
                        </td>
                      </tr>
                    )}
                    {section.rows.map(renderRow)}
                    {section.title && (
                      <tr className="bg-ink-50/50">
                        {leftCount > 0 && (
                          <td colSpan={leftCount} className="px-4 py-2 text-right text-xs font-medium text-ink-500">
                            {tr('tx.subtotal')}
                          </td>
                        )}
                        <td className="tnum px-4 py-2 text-right text-sm font-semibold text-ink-700">
                          {formatCurrency(section.sum)}
                        </td>
                        <td className="px-4 py-2" />
                      </tr>
                    )}
                  </Fragment>
                ))}
              </tbody>
              <tfoot>
                <tr className="border-t-2 border-ink-200 bg-ink-50/70">
                  {leftCount > 0 && (
                    <td colSpan={leftCount} className="px-4 py-3.5 text-sm font-semibold text-ink-700">
                      {tr('tx.grandTotal')} <span className="font-normal text-ink-400">{tr('tx.grandTotalHint')}</span>
                    </td>
                  )}
                  <td
                    className={`tnum px-4 py-3.5 text-right text-base font-bold ${
                      net >= 0 ? 'text-brand-700' : 'text-rose-600'
                    }`}
                  >
                    {formatCurrency(net)}
                  </td>
                  <td className="px-4 py-3.5" />
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      )}

      <TransactionModal open={modalOpen} onClose={() => setModalOpen(false)} editing={editing} />
    </div>
  )
}
