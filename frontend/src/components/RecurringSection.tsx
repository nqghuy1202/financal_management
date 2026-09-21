import { useState } from 'react'
import { ChevronDown, ChevronRight, Pause, Play, Trash2, Repeat, Check } from 'lucide-react'
import type { RecurringTransaction } from '../types'
import { useData } from '../context/DataContext'
import { useToast } from '../context/ToastContext'
import { useI18n } from '../context/I18nContext'
import { formatCurrency, formatNumberInput, parseNumberInput, formatDate } from '../lib/format'
import type { RecurringDraft } from './TransactionModal'

interface Draft {
  amount: string
  frequency: RecurringTransaction['frequency']
}

function draftFrom(rt: RecurringTransaction): Draft {
  return { amount: formatNumberInput(rt.amount), frequency: rt.frequency }
}

interface Props {
  onOpenDraft: (draft: RecurringDraft) => void
}

/**
 * Collapsible section above the Transactions table (spec-recurring-transactions):
 * (a) suggested-draft cards for templates that are due/overdue — tapping one
 * opens TransactionModal pre-filled via onOpenDraft, and (b) the full list
 * of templates (active + paused) with inline amount/frequency edit, pause/
 * resume, and delete. No new nav item or route — this is the only surface
 * for managing recurring templates.
 */
export function RecurringSection({ onOpenDraft }: Props) {
  const { recurringTransactions, categoryById, updateRecurringTransaction, deleteRecurringTransaction } = useData()
  const toast = useToast()
  const { t } = useI18n()
  const [open, setOpen] = useState(true)
  const [drafts, setDrafts] = useState<Record<string, Draft>>({})
  const [busyIds, setBusyIds] = useState<Set<string>>(new Set())

  if (recurringTransactions.length === 0) return null

  const dueDrafts = recurringTransactions.filter((rt) => rt.dueDraftDate)

  const withBusy = async (id: string, fn: () => Promise<void>) => {
    if (busyIds.has(id)) return
    setBusyIds((prev) => new Set(prev).add(id))
    try {
      await fn()
    } finally {
      setBusyIds((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }

  const draftOf = (rt: RecurringTransaction) => drafts[rt.id] ?? draftFrom(rt)
  const setDraft = (rt: RecurringTransaction, patch: Partial<Draft>) =>
    setDrafts((prev) => ({ ...prev, [rt.id]: { ...draftOf(rt), ...patch } }))
  const isDirty = (rt: RecurringTransaction) => {
    const d = drafts[rt.id]
    if (!d) return false
    return parseNumberInput(d.amount) !== rt.amount || d.frequency !== rt.frequency
  }

  const openDraftCard = (rt: RecurringTransaction) => {
    if (!rt.dueDraftDate) return
    onOpenDraft({
      recurringId: rt.id,
      type: rt.type,
      amount: rt.amount,
      categoryId: rt.categoryId,
      note: rt.note,
      date: rt.dueDraftDate,
    })
  }

  const handleSaveEdit = async (rt: RecurringTransaction) => {
    const d = draftOf(rt)
    const amount = parseNumberInput(d.amount)
    if (!amount || amount <= 0) return toast.error(t('err.amountPositive'))
    await withBusy(rt.id, async () => {
      try {
        await updateRecurringTransaction({ ...rt, amount, frequency: d.frequency })
        setDrafts((prev) => {
          const next = { ...prev }
          delete next[rt.id]
          return next
        })
        toast.success(t('toast.recurringUpdated'))
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.recurringUpdateError'))
      }
    })
  }

  const handleTogglePause = async (rt: RecurringTransaction) => {
    await withBusy(rt.id, async () => {
      try {
        await updateRecurringTransaction({ ...rt, active: !rt.active })
        toast.success(rt.active ? t('toast.recurringPaused') : t('toast.recurringResumed'))
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.recurringUpdateError'))
      }
    })
  }

  const handleDelete = async (rt: RecurringTransaction) => {
    await withBusy(rt.id, async () => {
      try {
        await deleteRecurringTransaction(rt.id)
        toast.success(t('toast.recurringDeleted'))
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.recurringDeleteError'))
      }
    })
  }

  return (
    <div className="card p-0">
      <button
        type="button"
        className="flex w-full items-center justify-between px-4 py-3.5"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <span className="flex items-center gap-2 text-sm font-semibold text-ink-700">
          <Repeat size={16} className="text-brand-600" />
          {t('recurring.sectionTitle')}
          {dueDrafts.length > 0 && (
            <span className="chip bg-brand-50 text-brand-700">{t('recurring.dueCount', { n: dueDrafts.length })}</span>
          )}
        </span>
        {open ? <ChevronDown size={16} className="text-ink-400" /> : <ChevronRight size={16} className="text-ink-400" />}
      </button>

      {open && (
        <div className="space-y-4 border-t border-ink-100 px-4 py-4">
          {dueDrafts.length > 0 && (
            <div>
              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-400">{t('recurring.dueTitle')}</p>
              <div className="grid gap-2 sm:grid-cols-2">
                {dueDrafts.map((rt) => {
                  const cat = categoryById(rt.categoryId)
                  const isIncome = rt.type === 'income'
                  return (
                    <button
                      key={rt.id}
                      type="button"
                      onClick={() => openDraftCard(rt)}
                      className="flex items-center justify-between gap-3 rounded-xl border border-brand-200 bg-brand-50/60 px-3 py-2.5 text-left transition hover:bg-brand-50"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-ink-800">{rt.note || cat?.name || '—'}</p>
                        <p className="text-xs text-ink-500">{formatDate(rt.dueDraftDate!)}</p>
                      </div>
                      <span className={`tnum shrink-0 text-sm font-semibold ${isIncome ? 'text-brand-700' : 'text-rose-600'}`}>
                        {formatCurrency(rt.amount)}
                      </span>
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-400">{t('recurring.templatesTitle')}</p>
            <div className="space-y-2">
              {recurringTransactions.map((rt) => {
                const cat = categoryById(rt.categoryId)
                const d = draftOf(rt)
                const busy = busyIds.has(rt.id)
                return (
                  <div
                    key={rt.id}
                    className={`flex flex-wrap items-center gap-2 rounded-xl border px-3 py-2 ${
                      rt.active ? 'border-ink-200' : 'border-ink-100 bg-ink-50/60 opacity-70'
                    }`}
                  >
                    <div className="min-w-0 flex-1">
                      {/* Falls back to '—' rather than the category name — the
                          subtitle right below already shows the category, so
                          repeating it here would just duplicate the line. */}
                      <p className="truncate text-sm font-medium text-ink-800">{rt.note || '—'}</p>
                      <p className="text-xs text-ink-500">
                        {cat?.name ?? '—'} · {t(`recurring.${rt.frequency}`)} · {t('recurring.nextDue', { date: formatDate(rt.nextDueDate) })}
                      </p>
                    </div>
                    <input
                      type="text"
                      inputMode="numeric"
                      className="input w-28 tnum"
                      value={d.amount}
                      onChange={(e) => setDraft(rt, { amount: formatNumberInput(e.target.value) })}
                    />
                    <select
                      className="input w-auto"
                      value={d.frequency}
                      onChange={(e) => setDraft(rt, { frequency: e.target.value as RecurringTransaction['frequency'] })}
                    >
                      <option value="weekly">{t('recurring.weekly')}</option>
                      <option value="monthly">{t('recurring.monthly')}</option>
                    </select>
                    {isDirty(rt) && (
                      <button
                        className="btn-icon"
                        aria-label={t('common.save')}
                        disabled={busy}
                        onClick={() => handleSaveEdit(rt)}
                      >
                        <Check size={16} />
                      </button>
                    )}
                    <button
                      className="btn-icon"
                      aria-label={rt.active ? t('recurring.pause') : t('recurring.resume')}
                      disabled={busy}
                      onClick={() => handleTogglePause(rt)}
                    >
                      {rt.active ? <Pause size={15} /> : <Play size={15} />}
                    </button>
                    <button
                      className="btn-icon-danger"
                      aria-label={t('common.delete')}
                      disabled={busy}
                      onClick={() => handleDelete(rt)}
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
