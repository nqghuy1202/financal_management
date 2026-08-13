import { useEffect, useState } from 'react'
import type { Transaction, TransactionType } from '../types'
import { useData } from '../context/DataContext'
import { useToast } from '../context/ToastContext'
import { useI18n } from '../context/I18nContext'
import { Modal } from './Modal'
import { CategoryIcon } from './CategoryIcon'
import { formatNumberInput, parseNumberInput } from '../lib/format'

interface Props {
  open: boolean
  onClose: () => void
  editing?: Transaction | null
}

const todayISO = () => new Date().toISOString().slice(0, 10)

export function TransactionModal({ open, onClose, editing }: Props) {
  const { categories, addTransaction, updateTransaction } = useData()
  const toast = useToast()
  const { t } = useI18n()
  const [type, setType] = useState<TransactionType>('expense')
  const [amount, setAmount] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [note, setNote] = useState('')
  const [date, setDate] = useState(todayISO())
  const [error, setError] = useState('')

  const typeCategories = categories.filter((c) => c.type === type)

  useEffect(() => {
    if (!open) return
    if (editing) {
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
  }, [open, editing])

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
      if (editing) await updateTransaction({ ...payload, id: editing.id })
      else await addTransaction(payload)
      toast.success(editing ? t('toast.txUpdated') : t('toast.txAdded'))
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
      title={editing ? t('txm.editTitle') : t('txm.addTitle')}
      footer={
        <>
          <button className="btn-outline" onClick={onClose}>{t('common.cancel')}</button>
          <button className="btn-primary" onClick={submit}>{editing ? t('common.save') : t('common.add')}</button>
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
            <div>
              <label className="label">{t('txm.note')}</label>
              <input className="input" placeholder={t('txm.notePlaceholder')} value={note} onChange={(e) => setNote(e.target.value)} />
            </div>
          </div>

          {/* Right: category picker */}
          <div>
            <label className="label">{t('txm.category')}</label>
            <div className="grid grid-cols-2 gap-2">
              {typeCategories.map((c) => (
                <button
                  key={c.id}
                  onClick={() => setCategoryId(c.id)}
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

        {error && <p role="alert" className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-600">{error}</p>}
      </div>
    </Modal>
  )
}
