export function formatCurrency(value: number, opts: { sign?: boolean } = {}): string {
  const formatted = new Intl.NumberFormat('vi-VN', {
    style: 'currency',
    currency: 'VND',
    maximumFractionDigits: 0,
  }).format(Math.abs(value))
  if (opts.sign && value !== 0) {
    return `${value > 0 ? '+' : '-'}${formatted}`
  }
  return value < 0 ? `-${formatted}` : formatted
}

export function formatCompact(value: number): string {
  return new Intl.NumberFormat('vi-VN', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value)
}

export function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

export function formatMonth(month: string): string {
  const [y, m] = month.split('-')
  return `Tháng ${Number(m)}/${y}`
}

export function currentMonth(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

export function monthOf(iso: string): string {
  return iso.slice(0, 7)
}

export function uid(): string {
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36).slice(-4)
}

/**
 * Format a numeric text input with thousand separators as the user types.
 * Keeps only digits, then groups them (vi-VN → dots, matching how amounts are
 * shown elsewhere). "1000" → "1.000".
 */
export function formatNumberInput(value: string | number): string {
  const digits = String(value).replace(/\D/g, '')
  if (!digits) return ''
  return Number(digits).toLocaleString('vi-VN')
}

/** Parse a grouped/formatted number string back to a plain integer. */
export function parseNumberInput(value: string): number {
  const digits = value.replace(/\D/g, '')
  return digits ? Number(digits) : 0
}
