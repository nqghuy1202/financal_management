import { useMemo, useState } from 'react'
import { Plus, Trash2, PiggyBank, AlertTriangle, CheckCircle2, Tag } from 'lucide-react'
import { useData } from '../context/DataContext'
import { useToast } from '../context/ToastContext'
import { useI18n } from '../context/I18nContext'
import { Modal } from '../components/Modal'
import { CategoryIcon, iconNames } from '../components/CategoryIcon'
import { budgetProgress } from '../lib/analytics'
import { currentMonth, formatCurrency, formatNumberInput, parseNumberInput } from '../lib/format'
import { localizedMonth } from '../lib/i18n'
import type { TransactionType } from '../types'

const PALETTE = ['#10b981', '#0ea5e9', '#6366f1', '#8b5cf6', '#ec4899', '#f97316', '#eab308', '#ef4444', '#64748b']

export function Budgets() {
  const { budgets, transactions, categories, upsertBudget, deleteBudget, addCategory, deleteCategory } = useData()
  const toast = useToast()
  const { t: tr, lang } = useI18n()
  const month = currentMonth()
  const monthLabel = localizedMonth(month, lang)

  const [budgetModal, setBudgetModal] = useState(false)
  const [catModal, setCatModal] = useState(false)

  // Budget form
  const expenseCats = categories.filter((c) => c.type === 'expense')
  const [bCat, setBCat] = useState(expenseCats[0]?.id ?? '')
  const [bLimit, setBLimit] = useState('')

  // Category form
  const [cName, setCName] = useState('')
  const [cType, setCType] = useState<TransactionType>('expense')
  const [cIcon, setCIcon] = useState('Tag')
  const [cColor, setCColor] = useState(PALETTE[0])

  const rows = useMemo(
    () => budgetProgress(budgets, transactions, categories, month),
    [budgets, transactions, categories, month],
  )

  const totalLimit = rows.reduce((s, r) => s + r.budget.limit, 0)
  const totalSpent = rows.reduce((s, r) => s + r.spent, 0)
  const overallPercent = totalLimit ? (totalSpent / totalLimit) * 100 : 0

  const saveBudget = async () => {
    const value = parseNumberInput(bLimit)
    if (!bCat || !value || value <= 0) return
    try {
      await upsertBudget({ categoryId: bCat, limit: value, month })
      toast.success(tr('toast.budgetSaved'))
      setBLimit('')
      setBudgetModal(false)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : tr('toast.budgetSaveError'))
    }
  }

  const saveCategory = async () => {
    if (!cName.trim()) return
    try {
      await addCategory({ name: cName.trim(), type: cType, icon: cIcon, color: cColor })
      toast.success(tr('toast.categoryAdded'))
      setCName('')
      setCIcon('Tag')
      setCColor(PALETTE[0])
      setCatModal(false)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : tr('toast.categoryAddError'))
    }
  }

  const handleDeleteBudget = async (id: string) => {
    try {
      await deleteBudget(id)
      toast.success(tr('toast.budgetDeleted'))
    } catch (e) {
      toast.error(e instanceof Error ? e.message : tr('toast.budgetDeleteError'))
    }
  }

  const handleDeleteCategory = async (id: string) => {
    try {
      await deleteCategory(id)
      toast.success(tr('toast.categoryDeleted'))
    } catch (e) {
      toast.error(e instanceof Error ? e.message : tr('toast.categoryDeleteError'))
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink-900">{tr('bud.title')}</h1>
          <p className="mt-1 text-sm text-ink-500">{tr('bud.subtitle', { month: monthLabel })}</p>
        </div>
        <div className="flex gap-2">
          <button className="btn-outline" onClick={() => setCatModal(true)}>
            <Tag size={17} /> {tr('bud.categoryBtn')}
          </button>
          <button className="btn-primary" onClick={() => setBudgetModal(true)}>
            <Plus size={17} /> {tr('bud.setBudget')}
          </button>
        </div>
      </div>

      {/* Overall */}
      <div className="card overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-4 p-5">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
              <PiggyBank size={24} />
            </div>
            <div>
              <p className="text-sm text-ink-500">{tr('bud.totalThisMonth')}</p>
              <p className="text-xl font-bold text-ink-900">
                {formatCurrency(totalSpent)} <span className="text-sm font-medium text-ink-400">/ {formatCurrency(totalLimit)}</span>
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className={`text-2xl font-bold ${overallPercent > 100 ? 'text-rose-500' : 'text-brand-600'}`}>
              {Math.round(overallPercent)}%
            </p>
            <p className="text-xs text-ink-400">{tr('bud.used')}</p>
          </div>
        </div>
        <div className="h-2.5 bg-ink-100">
          <div
            className={`h-full ${overallPercent > 100 ? 'bg-rose-500' : 'bg-brand-500'}`}
            style={{ width: `${Math.min(overallPercent, 100)}%` }}
          />
        </div>
      </div>

      {/* Budget cards */}
      {rows.length === 0 ? (
        <div className="card flex flex-col items-center justify-center gap-2 py-16 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-ink-100 text-ink-400">
            <PiggyBank size={22} />
          </div>
          <p className="font-medium text-ink-700">{tr('bud.noBudgets')}</p>
          <p className="text-sm text-ink-400">{tr('bud.noBudgetsHint')}</p>
          <button className="btn-primary mt-2" onClick={() => setBudgetModal(true)}>
            <Plus size={17} /> {tr('bud.setBudget')}
          </button>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {rows.map((r) => {
            const over = r.percent > 100
            const near = r.percent >= 80 && r.percent <= 100
            return (
              <div key={r.budget.id} className="card group p-5">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-ink-100 text-ink-500">
                      <CategoryIcon name={r.category.icon} size={20} />
                    </div>
                    <div>
                      <p className="font-semibold text-ink-800">{r.category.name}</p>
                      <p className="text-xs text-ink-400">{tr('bud.remaining', { x: formatCurrency(Math.max(r.remaining, 0)) })}</p>
                    </div>
                  </div>
                  <button
                    className="btn-icon-danger opacity-100 transition lg:opacity-0 lg:group-hover:opacity-100 lg:focus-within:opacity-100"
                    onClick={() => handleDeleteBudget(r.budget.id)}
                    aria-label={`${tr('common.delete')}: ${r.category.name}`}
                  >
                    <Trash2 size={16} />
                  </button>
                </div>

                <div className="mt-4">
                  <div className="mb-1.5 flex items-center justify-between text-sm">
                    <span className="font-medium text-ink-700">{formatCurrency(r.spent)}</span>
                    <span className="text-ink-400">/ {formatCurrency(r.budget.limit)}</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-ink-100">
                    <div
                      className={`h-full rounded-full ${over ? 'bg-rose-500' : near ? 'bg-amber-500' : 'bg-brand-500'}`}
                      style={{ width: `${Math.min(r.percent, 100)}%` }}
                    />
                  </div>
                </div>

                <div className="mt-3">
                  {over ? (
                    <span className="chip bg-rose-50 text-rose-600">
                      <AlertTriangle size={13} /> {tr('bud.over', { x: formatCurrency(-r.remaining) })}
                    </span>
                  ) : near ? (
                    <span className="chip bg-amber-50 text-amber-600">
                      <AlertTriangle size={13} /> {tr('bud.near')}
                    </span>
                  ) : (
                    <span className="chip bg-brand-50 text-brand-600">
                      <CheckCircle2 size={13} /> {tr('bud.within')}
                    </span>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Categories list */}
      <div className="card p-5">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="font-semibold text-ink-900">{tr('bud.categoriesTitle')}</h2>
            <p className="text-sm text-ink-500">{tr('bud.categoriesSub')}</p>
          </div>
          <button className="btn-outline" onClick={() => setCatModal(true)}>
            <Plus size={16} /> {tr('common.add')}
          </button>
        </div>
        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {categories.map((c) => (
            <div key={c.id} className="group flex items-center gap-3 rounded-xl border border-ink-100 px-3 py-2.5">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-ink-100 text-ink-500">
                <CategoryIcon name={c.icon} size={17} />
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-ink-800">{c.name}</p>
                <p className="text-xs text-ink-400">{c.type === 'income' ? tr('type.incomeFull') : tr('type.expenseFull')}</p>
              </div>
              <button
                className="btn-icon-danger opacity-100 transition lg:opacity-0 lg:group-hover:opacity-100 lg:focus-within:opacity-100"
                onClick={() => {
                  if (window.confirm(tr('bud.confirmDeleteCategory', { name: c.name }))) {
                    handleDeleteCategory(c.id)
                  }
                }}
                aria-label={`${tr('common.delete')}: ${c.name}`}
              >
                <Trash2 size={15} />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Budget modal */}
      <Modal
        open={budgetModal}
        onClose={() => setBudgetModal(false)}
        size="lg"
        title={tr('bud.setBudget')}
        footer={
          <>
            <button className="btn-outline" onClick={() => setBudgetModal(false)}>{tr('common.cancel')}</button>
            <button className="btn-primary" onClick={saveBudget}>{tr('common.save')}</button>
          </>
        }
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="label">{tr('bud.expenseCategory')}</label>
            <select className="input" value={bCat} onChange={(e) => setBCat(e.target.value)}>
              {expenseCats.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">{tr('bud.limitMonth', { month: monthLabel })}</label>
            <input
              type="text"
              inputMode="numeric"
              className="input text-lg font-semibold tnum"
              placeholder="0"
              value={bLimit}
              onChange={(e) => setBLimit(formatNumberInput(e.target.value))}
            />
          </div>
        </div>
      </Modal>

      {/* Category modal */}
      <Modal
        open={catModal}
        onClose={() => setCatModal(false)}
        size="xl"
        title={tr('bud.addCategoryTitle')}
        footer={
          <>
            <button className="btn-outline" onClick={() => setCatModal(false)}>{tr('common.cancel')}</button>
            <button className="btn-primary" onClick={saveCategory}>{tr('common.add')}</button>
          </>
        }
      >
        <div className="space-y-4">
          {/* Row 1: name + type */}
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="label">{tr('bud.categoryName')}</label>
              <input className="input" placeholder={tr('bud.categoryNamePlaceholder')} value={cName} onChange={(e) => setCName(e.target.value)} />
            </div>
            <div>
              <label className="label">{tr('bud.typeLabel')}</label>
              <div className="grid grid-cols-2 gap-2 rounded-xl bg-ink-100 p-1">
                {(['expense', 'income'] as TransactionType[]).map((opt) => (
                  <button
                    key={opt}
                    onClick={() => setCType(opt)}
                    className={`rounded-lg py-2 text-sm font-semibold transition ${
                      cType === opt ? 'bg-white text-ink-900 shadow-sm' : 'text-ink-500'
                    }`}
                  >
                    {opt === 'income' ? tr('type.incomeFull') : tr('type.expenseFull')}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Row 2: color + icon */}
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="label">{tr('bud.color')}</label>
              <div className="flex flex-wrap gap-2">
                {PALETTE.map((color) => (
                  <button
                    key={color}
                    onClick={() => setCColor(color)}
                    className={`h-8 w-8 rounded-full transition ${cColor === color ? 'ring-2 ring-ink-900 ring-offset-2' : ''}`}
                    style={{ background: color }}
                    aria-label={color}
                  />
                ))}
              </div>
            </div>
            <div>
              <label className="label">{tr('bud.icon')}</label>
              <div className="grid grid-cols-5 gap-2">
                {iconNames.map((name) => (
                  <button
                    key={name}
                    onClick={() => setCIcon(name)}
                    className={`flex h-10 items-center justify-center rounded-lg border transition ${
                      cIcon === name
                        ? 'border-brand-400 bg-brand-50 text-brand-700'
                        : 'border-ink-200 text-ink-500 hover:bg-ink-50'
                    }`}
                  >
                    <CategoryIcon name={name} size={18} />
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}
