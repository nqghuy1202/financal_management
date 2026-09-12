import { useEffect, useState } from 'react'
import { ChevronDown, ChevronRight, Check, Plus, Trash2 } from 'lucide-react'
import type { FixedCost } from '../types'
import { useData } from '../context/DataContext'
import { useToast } from '../context/ToastContext'
import { useI18n } from '../context/I18nContext'
import { Modal } from './Modal'
import { formatNumberInput, parseNumberInput } from '../lib/format'

interface Props {
  open: boolean
  onClose: () => void
}

interface Draft {
  name: string
  amount: string
}

function draftFrom(fc: FixedCost): Draft {
  return { name: fc.name, amount: formatNumberInput(fc.amount) }
}

/**
 * The one surface behind the Safe-to-spend hero's pencil icon: income +
 * fixed-costs list + savings goal, plus a collapsed "Advanced" section for
 * the cycle start day. Mirrors TransactionModal.tsx's structure and
 * error-string convention. Income/savings-goal/cycle-day save together via
 * one PUT /cycle-settings on the footer's Save button; each fixed-cost
 * add/edit/delete calls its own endpoint immediately (per spec-1-5's
 * Boundaries & Constraints).
 */
export function CycleUpdateSheet({ open, onClose }: Props) {
  const { cycleSummary, fixedCosts, settings, saveCycleSettings, addFixedCost, updateFixedCost, deleteFixedCost } =
    useData()
  const toast = useToast()
  const { t } = useI18n()

  const [income, setIncome] = useState('')
  const [savingsGoal, setSavingsGoal] = useState('')
  const [cycleStartDay, setCycleStartDay] = useState('1')
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const [drafts, setDrafts] = useState<Record<string, Draft>>({})
  const [newName, setNewName] = useState('')
  const [newAmount, setNewAmount] = useState('')
  // In-flight row ids for the per-row add/update/delete actions, keyed by
  // fixed-cost id (or 'new' for the add row), so a fast double-click can't
  // fire duplicate requests.
  const [busyIds, setBusyIds] = useState<Set<string>>(new Set())

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

  // Reset every field from the current context values each time the sheet
  // opens — it has no independent lifetime beyond one open/close cycle.
  useEffect(() => {
    if (!open) return
    setIncome(cycleSummary?.income != null ? formatNumberInput(cycleSummary.income) : '')
    setSavingsGoal(formatNumberInput(settings.savingsGoal))
    setCycleStartDay(String(settings.cycleStartDay))
    setAdvancedOpen(false)
    setError('')
    setDrafts(Object.fromEntries(fixedCosts.map((fc) => [fc.id, draftFrom(fc)])))
    setNewName('')
    setNewAmount('')
    // Only re-sync when the sheet transitions to open, not on every
    // background refetch, so in-progress edits aren't clobbered.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const setDraft = (id: string, patch: Partial<Draft>) =>
    setDrafts((prev) => ({ ...prev, [id]: { ...prev[id], ...patch } }))

  const isDirty = (fc: FixedCost) => {
    const d = drafts[fc.id]
    if (!d) return false
    return d.name.trim() !== fc.name || parseNumberInput(d.amount) !== fc.amount
  }

  const handleUpdateFixedCost = async (fc: FixedCost) => {
    const d = drafts[fc.id]
    if (!d) return
    const name = d.name.trim()
    const amount = parseNumberInput(d.amount)
    if (!name || amount <= 0) return setError(t('err.fixedCostInvalid'))
    setError('')
    await withBusy(fc.id, async () => {
      try {
        await updateFixedCost({ id: fc.id, name, amount })
        toast.success(t('toast.fixedCostUpdated'))
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.fixedCostUpdateError'))
      }
    })
  }

  const handleDeleteFixedCost = async (id: string) => {
    await withBusy(id, async () => {
      try {
        await deleteFixedCost(id)
        setDrafts((prev) => {
          const next = { ...prev }
          delete next[id]
          return next
        })
        toast.success(t('toast.fixedCostDeleted'))
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.fixedCostDeleteError'))
      }
    })
  }

  const handleAddFixedCost = async () => {
    const name = newName.trim()
    const amount = parseNumberInput(newAmount)
    if (!name || amount <= 0) return setError(t('err.fixedCostInvalid'))
    setError('')
    await withBusy('new', async () => {
      try {
        await addFixedCost({ name, amount })
        setNewName('')
        setNewAmount('')
        toast.success(t('toast.fixedCostAdded'))
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t('toast.fixedCostAddError'))
      }
    })
  }

  const handleSave = async () => {
    const incomeValue = parseNumberInput(income)
    if (!incomeValue || incomeValue <= 0) return setError(t('err.incomePositive'))
    const savingsGoalValue = parseNumberInput(savingsGoal)
    if (savingsGoalValue < 0) return setError(t('err.savingsGoalNegative'))
    const cycleStartDayValue = Number(cycleStartDay)
    if (!Number.isInteger(cycleStartDayValue) || cycleStartDayValue < 1 || cycleStartDayValue > 31) {
      return setError(t('err.cycleStartDayRange'))
    }
    setError('')
    setSaving(true)
    try {
      // Flush any fixed-cost rows the user edited in-place but never
      // confirmed via their own per-row Check icon, so the main Save button
      // doesn't silently drop them.
      const dirty = fixedCosts.filter((fc) => isDirty(fc))
      await Promise.all(
        dirty.map((fc) => {
          const d = drafts[fc.id]
          return updateFixedCost({ id: fc.id, name: d.name.trim(), amount: parseNumberInput(d.amount) })
        }),
      )
      await saveCycleSettings({ income: incomeValue, savingsGoal: savingsGoalValue, cycleStartDay: cycleStartDayValue })
      toast.success(t('toast.cycleSettingsSaved'))
      onClose()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('toast.cycleSettingsSaveError'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      size="lg"
      title={t('cus.title')}
      footer={
        <>
          <button className="btn-outline" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button className="btn-primary" onClick={handleSave} disabled={saving}>
            {t('common.save')}
          </button>
        </>
      }
    >
      <div className="space-y-5">
        <div>
          <label className="label">{t('cus.income')}</label>
          <input
            type="text"
            inputMode="numeric"
            className="input text-lg font-semibold tnum"
            placeholder="0"
            value={income}
            onChange={(e) => setIncome(formatNumberInput(e.target.value))}
          />
        </div>

        <div>
          <label className="label">{t('cus.fixedCosts')}</label>
          <div className="space-y-2">
            {fixedCosts.length === 0 && (
              <p className="text-sm text-ink-400">{t('cus.noFixedCosts')}</p>
            )}
            {fixedCosts.map((fc) => {
              const d = drafts[fc.id] ?? draftFrom(fc)
              return (
                <div key={fc.id} className="flex items-center gap-2">
                  <input
                    className="input"
                    value={d.name}
                    onChange={(e) => setDraft(fc.id, { name: e.target.value })}
                    placeholder={t('cus.fixedCostNamePlaceholder')}
                  />
                  <input
                    type="text"
                    inputMode="numeric"
                    className="input w-32 tnum"
                    value={d.amount}
                    onChange={(e) => setDraft(fc.id, { amount: formatNumberInput(e.target.value) })}
                    placeholder={t('cus.fixedCostAmountPlaceholder')}
                  />
                  {isDirty(fc) && (
                    <button
                      className="btn-icon"
                      aria-label={t('common.save')}
                      disabled={busyIds.has(fc.id)}
                      onClick={() => handleUpdateFixedCost(fc)}
                    >
                      <Check size={16} />
                    </button>
                  )}
                  <button
                    className="btn-icon-danger"
                    aria-label={t('common.delete')}
                    disabled={busyIds.has(fc.id)}
                    onClick={() => handleDeleteFixedCost(fc.id)}
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              )
            })}
            <div className="flex items-center gap-2">
              <input
                className="input"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder={t('cus.fixedCostNamePlaceholder')}
              />
              <input
                type="text"
                inputMode="numeric"
                className="input w-32 tnum"
                value={newAmount}
                onChange={(e) => setNewAmount(formatNumberInput(e.target.value))}
                placeholder={t('cus.fixedCostAmountPlaceholder')}
              />
              <button
                className="btn-icon"
                aria-label={t('cus.addFixedCost')}
                disabled={busyIds.has('new')}
                onClick={handleAddFixedCost}
              >
                <Plus size={16} />
              </button>
            </div>
          </div>
        </div>

        <div>
          <label className="label">{t('cus.savingsGoal')}</label>
          <input
            type="text"
            inputMode="numeric"
            className="input tnum"
            placeholder="0"
            value={savingsGoal}
            onChange={(e) => setSavingsGoal(formatNumberInput(e.target.value))}
          />
        </div>

        <div>
          <button
            type="button"
            className="flex items-center gap-1.5 text-sm font-medium text-ink-600 hover:text-ink-800"
            onClick={() => setAdvancedOpen((v) => !v)}
          >
            {advancedOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
            {t('cus.advanced')}
          </button>
          {advancedOpen && (
            <div className="mt-3">
              <label className="label">{t('cus.cycleStartDay')}</label>
              <input
                type="number"
                min={1}
                max={31}
                className="input w-24 tnum"
                value={cycleStartDay}
                onChange={(e) => setCycleStartDay(e.target.value)}
              />
              <p className="mt-1 text-xs text-ink-400">{t('cus.cycleStartDayHint')}</p>
            </div>
          )}
        </div>

        {error && (
          <p role="alert" className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-600">
            {error}
          </p>
        )}
      </div>
    </Modal>
  )
}
