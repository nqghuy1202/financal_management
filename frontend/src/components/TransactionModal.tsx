import { useEffect, useMemo, useRef, useState } from 'react'
import type { RecurringFrequency, Transaction, TransactionType } from '../types'
import { useData } from '../context/DataContext'
import { useToast } from '../context/ToastContext'
import { useI18n } from '../context/I18nContext'
import { Modal } from './Modal'
import { CategoryIcon } from './CategoryIcon'
import { formatNumberInput, parseNumberInput } from '../lib/format'

// A suggested draft (due or "catch up") for a recurring template, as opened
// from Transactions.tsx's collapsible section — pre-fills the same modal
// used for adding, but submit confirms the draft (POST
// /recurring-transactions/:id/confirm) instead of creating a transaction
// directly, and the "make recurring" toggle never shows (confirming a draft
// produces a normal transaction, not a nested template).
export interface RecurringDraft {
  recurringId: string
  type: TransactionType
  amount: number
  categoryId: string
  note: string
  date: string
}

interface Props {
  open: boolean
  onClose: () => void
  editing?: Transaction | null
  draft?: RecurringDraft | null
}

const todayISO = () => new Date().toISOString().slice(0, 10)

export function TransactionModal({ open, onClose, editing, draft }: Props) {
  const { categories, transactions, addTransaction, updateTransaction, addRecurringTransaction, confirmRecurringTransaction } =
    useData()
  const toast = useToast()
  const { t } = useI18n()
  const [type, setType] = useState<TransactionType>('expense')
  const [amount, setAmount] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [note, setNote] = useState('')
  const [date, setDate] = useState(todayISO())
  const [error, setError] = useState('')
  const [categoryTouched, setCategoryTouched] = useState(false)
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [activeSuggestion, setActiveSuggestion] = useState(-1)
  const noteFieldRef = useRef<HTMLDivElement>(null)
  // Recurring toggle: only relevant when adding a brand-new transaction —
  // never while editing an existing one or confirming a suggested draft.
  const [makeRecurring, setMakeRecurring] = useState(false)
  const [frequency, setFrequency] = useState<RecurringFrequency>('monthly')
  const showRecurringToggle = !editing && !draft

  const typeCategories = categories.filter((c) => c.type === type)

  const noteSuggestions = useMemo(() => {
    const q = note.trim().toLowerCase()
    if (q.length < 2) return []
    const seen = new Set<string>()
    const out: { note: string; categoryId: string }[] = []
    for (const tx of transactions) {
      if (tx.type !== type) continue
      const n = tx.note.trim()
      if (!n) continue
      const key = n.toLowerCase()
      if (!key.includes(q) || seen.has(key)) continue
      seen.add(key)
      out.push({ note: n, categoryId: tx.categoryId })
      if (out.length >= 8) break
    }
    return out
  }, [note, transactions, type])

  const selectSuggestion = (s: { note: string; categoryId: string }) => {
    setNote(s.note)
    if (!categoryTouched && typeCategories.some((c) => c.id === s.categoryId)) {
      setCategoryId(s.categoryId)
    }
    setShowSuggestions(false)
    setActiveSuggestion(-1)
  }

  useEffect(() => {
    if (!showSuggestions) return
    const onClickOutside = (e: MouseEvent) => {
      if (noteFieldRef.current && !noteFieldRef.current.contains(e.target as Node)) {
        setShowSuggestions(false)
        setActiveSuggestion(-1)
      }
    }
    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [showSuggestions])

  useEffect(() => {
    if (!open) return
    if (draft) {
      setType(draft.type)
      setAmount(formatNumberInput(draft.amount))
      setCategoryId(draft.categoryId)
      setNote(draft.note)
      setDate(draft.date)
    } else if (editing) {
      setType(editing.type)
      setAmount(formatNumberInput(editing.amount))
      setCategoryId(editing.categoryId)
      setNote(editing.note)
      setDate(editing.date)
    } else {
      setType('expense')
      setAmount('')
      setCategoryId('')
      setNote('')
      setDate(todayISO())
    }
    setError('')
    // Editing an existing transaction (or confirming a draft) means its
    // category was already a deliberate choice — an autocomplete pick must
    // not silently overwrite it.
    setCategoryTouched(!!editing || !!draft)
    setShowSuggestions(false)
    setActiveSuggestion(-1)
    setMakeRecurring(false)
    setFrequency('monthly')
  }, [open, editing, draft])

  // Keep category valid when switching type
  useEffect(() => {
    if (categoryId && !typeCategories.some((c) => c.id === categoryId)) {
      setCategoryId(typeCategories[0]?.id ?? '')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [type])

  const submit = async () => {
    const value = parseNumberInput(amount)
    if (!value || value <= 0) return setError(t('err.amountPositive'))
    if (!categoryId) return setError(t('err.pickCategory'))
    const payload = { type, amount: value, categoryId, note: note.trim(), date }
    try {
      if (draft) {
        await confirmRecurringTransaction(draft.recurringId, payload)
        toast.success(t('toast.recurringConfirmed'))
      } else if (editing) {
        await updateTransaction({ ...payload, id: editing.id })
        toast.success(t('toast.txUpdated'))
      } else {
        await addTransaction(payload)
        if (makeRecurring) {
          try {
            await addRecurringTransaction({ ...payload, frequency })
            toast.success(t('toast.recurringCreated'))
          } catch (e) {
            toast.error(e instanceof Error ? e.message : t('toast.recurringCreateError'))
          }
        } else {
          toast.success(t('toast.txAdded'))
        }
      }
      onClose()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('toast.txSaveError'))
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      size="xl"
      title={draft ? t('txm.confirmDraftTitle') : editing ? t('txm.editTitle') : t('txm.addTitle')}
      footer={
        <>
          <button className="btn-outline" onClick={onClose}>{t('common.cancel')}</button>
          <button className="btn-primary" onClick={submit}>
            {draft ? t('common.confirm') : editing ? t('common.save') : t('common.add')}
          </button>
        </>
      }
    >
      <div className="space-y-4">
        {/* Type toggle */}
        <div className="grid grid-cols-2 gap-2 rounded-xl bg-ink-100 p-1">
          {(['expense', 'income'] as TransactionType[]).map((opt) => (
            <button
              key={opt}
              onClick={() => setType(opt)}
              className={`rounded-lg py-2 text-sm font-semibold transition ${
                type === opt
                  ? opt === 'income'
                    ? 'bg-white text-brand-600 shadow-sm'
                    : 'bg-white text-rose-600 shadow-sm'
                  : 'text-ink-500'
              }`}
            >
              {opt === 'income' ? t('type.incomeFull') : t('type.expenseFull')}
            </button>
          ))}
        </div>

        {/* Horizontal 2-column layout */}
        <div className="grid gap-5 sm:grid-cols-2">
          {/* Left: amount + date + note */}
          <div className="space-y-4">
            <div>
              <label className="label">{t('txm.amount')}</label>
              <input
                type="text"
                inputMode="numeric"
                className="input text-lg font-semibold tnum"
                placeholder="0"
                value={amount}
                onChange={(e) => setAmount(formatNumberInput(e.target.value))}
              />
            </div>
            <div>
              <label className="label">{t('txm.date')}</label>
              <input type="date" className="input" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
            <div ref={noteFieldRef} className="relative">
              <label className="label">{t('txm.note')}</label>
              <input
                className="input"
                placeholder={t('txm.notePlaceholder')}
                value={note}
                role="combobox"
                aria-autocomplete="list"
                aria-expanded={showSuggestions && noteSuggestions.length > 0}
                aria-controls="note-suggestions-listbox"
                aria-activedescendant={activeSuggestion >= 0 ? `note-suggestion-${activeSuggestion}` : undefined}
                onChange={(e) => {
                  setNote(e.target.value)
                  setShowSuggestions(true)
                  setActiveSuggestion(-1)
                }}
                onFocus={() => setShowSuggestions(true)}
                onBlur={() => {
                  setShowSuggestions(false)
                  setActiveSuggestion(-1)
                }}
                onKeyDown={(e) => {
                  if (!showSuggestions || noteSuggestions.length === 0) return
                  if (e.key === 'ArrowDown') {
                    e.preventDefault()
                    setActiveSuggestion((i) => (i + 1) % noteSuggestions.length)
                  } else if (e.key === 'ArrowUp') {
                    e.preventDefault()
                    setActiveSuggestion((i) => (i <= 0 ? noteSuggestions.length - 1 : i - 1))
                  } else if (e.key === 'Enter' && activeSuggestion >= 0) {
                    e.preventDefault()
                    selectSuggestion(noteSuggestions[activeSuggestion])
                  } else if (e.key === 'Escape') {
                    setShowSuggestions(false)
                    setActiveSuggestion(-1)
                  }
                }}
              />
              {showSuggestions && noteSuggestions.length > 0 && (
                <ul id="note-suggestions-listbox" role="listbox" className="absolute z-10 mt-1 w-full overflow-hidden rounded-xl border border-ink-200 bg-white shadow-lg">
                  {noteSuggestions.map((s, i) => (
                    <li key={s.note} id={`note-suggestion-${i}`} role="option" aria-selected={i === activeSuggestion}>
                      <button
                        type="button"
                        tabIndex={-1}
                        onMouseDown={(e) => e.preventDefault()}
                        onClick={() => selectSuggestion(s)}
                        className={`block w-full truncate px-3 py-2 text-left text-sm ${
                          i === activeSuggestion ? 'bg-brand-50 text-brand-700' : 'text-ink-600 hover:bg-ink-50'
                        }`}
                      >
                        {s.note}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>

          {/* Right: category picker */}
          <div>
            <label className="label">{t('txm.category')}</label>
            <div className="grid grid-cols-2 gap-2">
              {typeCategories.map((c) => (
                <button
                  key={c.id}
                  onClick={() => {
                    setCategoryId(c.id)
                    setCategoryTouched(true)
                  }}
                  className={`flex items-center gap-2 rounded-xl border px-3 py-2.5 text-sm font-medium transition ${
                    categoryId === c.id
                      ? 'border-brand-400 bg-brand-50 text-brand-700'
                      : 'border-ink-200 text-ink-600 hover:bg-ink-50'
                  }`}
                >
                  <CategoryIcon name={c.icon} size={16} />
                  <span className="truncate">{c.name}</span>
                </button>
              ))}
            </div>
          </div>
        </div>

        {showRecurringToggle && (
          <div className="rounded-xl border border-ink-200 p-3">
            <label className="flex cursor-pointer items-center gap-2.5 text-sm font-medium text-ink-700">
              <input
                type="checkbox"
                className="h-4 w-4 rounded border-ink-300 text-brand-600 focus:ring-brand-500/30"
                checked={makeRecurring}
                onChange={(e) => setMakeRecurring(e.target.checked)}
              />
              {t('txm.makeRecurring')}
            </label>
            {makeRecurring && (
              <div className="mt-3 grid grid-cols-2 gap-2">
                {(['weekly', 'monthly'] as RecurringFrequency[]).map((opt) => (
                  <button
                    key={opt}
                    type="button"
                    onClick={() => setFrequency(opt)}
                    className={`rounded-lg py-2 text-sm font-semibold transition ${
                      frequency === opt ? 'bg-brand-50 text-brand-700 ring-1 ring-brand-300' : 'bg-ink-100 text-ink-500'
                    }`}
                  >
                    {opt === 'weekly' ? t('recurring.weekly') : t('recurring.monthly')}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {error && <p role="alert" className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-600">{error}</p>}
      </div>
    </Modal>
  )
}
