import { useState } from 'react'
import { AlertTriangle, Pencil } from 'lucide-react'
import { useData } from '../context/DataContext'
import { useI18n } from '../context/I18nContext'
import { formatCurrency } from '../lib/format'
import { CycleUpdateSheet } from './CycleUpdateSheet'

/**
 * First element on the Dashboard: today's Safe-to-spend figure, or a prompt
 * to declare income when none has been set for the current cycle. A pencil
 * icon opens CycleUpdateSheet to edit income/fixed costs/savings goal/cycle
 * day. See spec-1-5-safe-to-spend-frontend.md's I/O matrix for the three
 * states rendered here.
 */
export function SafeToSpendHero() {
  const { cycleSummary, fixedCosts, settings, loading, loadError } = useData()
  const { t } = useI18n()
  const [sheetOpen, setSheetOpen] = useState(false)

  // Cold load: only while the initial fetch is in flight, so a fetch failure
  // (or a logged-out user briefly rendering this) doesn't skeleton forever.
  const cold = loading
  // A failed initial fetch is a distinct state from "income not declared
  // yet" — don't let an outage masquerade as a data-entry prompt.
  const failed = !cold && loadError
  const incomeMissing = !cold && !failed && (!cycleSummary || cycleSummary.income === null)
  const overspent = !cold && !failed && !incomeMissing && cycleSummary!.safeToSpend < 0
  // Nothing has ever been confirmed for fixed costs / savings goal — the
  // formula silently treats them as 0, so the number may read too generous.
  const maybeInaccurate =
    !cold && !failed && !incomeMissing && (fixedCosts.length === 0 || settings.savingsGoal === 0)

  return (
    <section className="card border-brand-500/20 bg-brand-50 p-6" aria-label={t('sth.label')}>
      {cold ? (
        <div className="animate-pulse space-y-3" aria-hidden="true">
          <div className="h-4 w-32 rounded bg-brand-200/70" />
          <div className="h-10 w-56 rounded bg-brand-200/70" />
          <div className="h-4 w-40 rounded bg-brand-200/70" />
        </div>
      ) : failed ? (
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-sm font-semibold text-brand-700">{t('sth.label')}</p>
            <p className="mt-1 flex items-center gap-1.5 text-lg font-semibold text-rose-600">
              <AlertTriangle size={18} /> {t('sth.loadError')}
            </p>
          </div>
          <button className="btn-outline" onClick={() => window.location.reload()}>
            {t('sth.retry')}
          </button>
        </div>
      ) : incomeMissing ? (
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-sm font-semibold text-brand-700">{t('sth.label')}</p>
            <p className="mt-1 text-lg font-semibold text-ink-800">{t('sth.updateIncomePrompt')}</p>
          </div>
          <button className="btn-primary" onClick={() => setSheetOpen(true)}>
            <Pencil size={16} /> {t('sth.updateIncomeCta')}
          </button>
        </div>
      ) : (
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-sm font-semibold text-brand-700">{t('sth.label')}</p>
            <p
              className={`display-number tnum ${overspent ? 'text-rose-600' : 'text-ink-900'}`}
            >
              {formatCurrency(cycleSummary!.safeToSpend)}
            </p>
            <p className="mt-1 text-sm text-ink-600">
              {t('sth.daysRemaining', { n: cycleSummary!.daysRemaining })}
            </p>
            {overspent && (
              <p className="mt-1 flex items-center gap-1.5 text-xs font-medium text-rose-600">
                <AlertTriangle size={13} /> {t('sth.overspentHint')}
              </p>
            )}
            {maybeInaccurate && (
              <p className="mt-2 text-xs text-ink-500">{t('sth.inaccurateHint')}</p>
            )}
          </div>
          <button
            className="btn-icon"
            aria-label={t('sth.editAria')}
            onClick={() => setSheetOpen(true)}
          >
            <Pencil size={18} />
          </button>
        </div>
      )}

      <CycleUpdateSheet open={sheetOpen} onClose={() => setSheetOpen(false)} />
    </section>
  )
}
