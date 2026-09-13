import { AlertTriangle, X } from 'lucide-react'
import type { ActiveAlert } from '../types'
import { useData } from '../context/DataContext'
import { useI18n } from '../context/I18nContext'

interface Props {
  alert: ActiveAlert
}

/**
 * One dismissible budget-threshold alert, rendered directly under
 * SafeToSpendHero on the Dashboard (Story 2.2). "over" (threshold 100) uses
 * the rose palette, "near" (70/90) uses amber — same chip-adjacent
 * convention as Budgets.tsx's status pills.
 */
export function AlertBanner({ alert }: Props) {
  const { categoryById, cycleSummary, dismissAlert } = useData()
  const { t } = useI18n()

  const categoryName = categoryById(alert.categoryId)?.name ?? t('alert.unknownCategory')
  const daysRemaining = cycleSummary?.daysRemaining ?? 0
  const isOver = alert.status === 'over'
  const message = t(isOver ? 'alert.over' : 'alert.near', { category: categoryName, days: daysRemaining })

  return (
    <div
      role="alert"
      className={`flex items-center gap-3 rounded-xl border-l-4 px-4 py-3 ${
        isOver ? 'border-rose-500 bg-rose-50 text-rose-600' : 'border-amber-500 bg-amber-50 text-amber-600'
      }`}
    >
      <AlertTriangle size={18} className="shrink-0" aria-hidden="true" />
      <p className="flex-1 text-sm font-medium">{message}</p>
      <button
        type="button"
        className="btn-icon shrink-0"
        aria-label={t('alert.dismissAria', { category: categoryName })}
        onClick={() => dismissAlert(alert.categoryId, alert.threshold)}
      >
        <X size={16} />
      </button>
    </div>
  )
}
