import type { ReactNode } from 'react'

export function StatCard({
  label,
  value,
  icon,
  accent = 'brand',
  trend,
}: {
  label: string
  value: string
  icon: ReactNode
  accent?: 'brand' | 'rose' | 'sky' | 'amber'
  trend?: { value: string; positive: boolean }
}) {
  const accents: Record<string, string> = {
    brand: 'bg-brand-50 text-brand-600',
    rose: 'bg-rose-50 text-rose-600',
    sky: 'bg-sky-50 text-sky-600',
    amber: 'bg-amber-50 text-amber-600',
  }
  return (
    <div className="card p-5">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-ink-500">{label}</p>
          <p className="tnum mt-2 text-2xl font-bold tracking-tight text-ink-900">{value}</p>
        </div>
        <div className={`flex h-11 w-11 items-center justify-center rounded-xl ${accents[accent]}`}>
          {icon}
        </div>
      </div>
      {trend && (
        <p
          className={`mt-3 text-xs font-medium ${
            trend.positive ? 'text-brand-700' : 'text-rose-600'
          }`}
        >
          {trend.value}
        </p>
      )}
    </div>
  )
}
