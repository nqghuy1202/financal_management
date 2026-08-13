import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import {
  ArrowDownRight,
  ArrowUpRight,
  Wallet,
  PiggyBank,
  ArrowRight,
} from 'lucide-react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { useData } from '../context/DataContext'
import { useI18n } from '../context/I18nContext'
import { StatCard } from '../components/StatCard'
import { CategoryIcon } from '../components/CategoryIcon'
import { budgetProgress, expenseBreakdown, filterByMonth, monthlyTrend, sumByType } from '../lib/analytics'
import { currentMonth, formatCompact, formatCurrency, formatDate } from '../lib/format'

export function Dashboard() {
  const { transactions, categories, budgets, categoryById } = useData()
  const { t: tr } = useI18n()
  const month = currentMonth()

  const monthTxs = useMemo(() => filterByMonth(transactions, month), [transactions, month])
  const income = useMemo(() => sumByType(monthTxs, 'income'), [monthTxs])
  const expense = useMemo(() => sumByType(monthTxs, 'expense'), [monthTxs])
  const balance = income - expense
  const savingRate = income ? Math.round((balance / income) * 100) : 0

  const trend = useMemo(() => monthlyTrend(transactions, 6), [transactions])
  const breakdown = useMemo(() => expenseBreakdown(monthTxs, categories), [monthTxs, categories])
  const budgetRows = useMemo(
    () => budgetProgress(budgets, transactions, categories, month).slice(0, 4),
    [budgets, transactions, categories, month],
  )
  const recent = useMemo(() => transactions.slice(0, 5), [transactions])

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink-900">{tr('dash.title')}</h1>
          <p className="mt-1 text-sm text-ink-500">{tr('dash.subtitle')}</p>
        </div>
        <Link to="/transactions" className="btn-primary">
          <Wallet size={17} /> {tr('dash.addTx')}
        </Link>
      </div>

      {/* Stat cards */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label={tr('dash.balance')}
          value={formatCurrency(balance)}
          icon={<Wallet size={20} />}
          accent="brand"
          trend={{ value: tr('dash.savingRate', { x: savingRate }), positive: balance >= 0 }}
        />
        <StatCard
          label={tr('dash.income')}
          value={formatCurrency(income)}
          icon={<ArrowUpRight size={20} />}
          accent="sky"
        />
        <StatCard
          label={tr('dash.expense')}
          value={formatCurrency(expense)}
          icon={<ArrowDownRight size={20} />}
          accent="rose"
        />
        <StatCard
          label={tr('dash.budgetsTracked')}
          value={String(budgets.filter((b) => b.month === month).length)}
          icon={<PiggyBank size={20} />}
          accent="amber"
        />
      </div>

      {/* Charts row */}
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="card p-5 lg:col-span-2">
          <div className="mb-4 flex items-center justify-between">
            <div>
              <h2 className="font-semibold text-ink-900">{tr('dash.cashflow')}</h2>
              <p className="text-sm text-ink-500">{tr('dash.cashflowSub')}</p>
            </div>
            <div className="flex items-center gap-4 text-xs">
              <span className="flex items-center gap-1.5 text-ink-500">
                <span className="h-2.5 w-2.5 rounded-full bg-brand-500" /> {tr('type.income')}
              </span>
              <span className="flex items-center gap-1.5 text-ink-500">
                <span className="h-2.5 w-2.5 rounded-full bg-rose-400" /> {tr('type.expense')}
              </span>
            </div>
          </div>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={trend} margin={{ left: -18, right: 8, top: 8 }}>
                <defs>
                  <linearGradient id="gInc" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#10b981" stopOpacity={0.35} />
                    <stop offset="100%" stopColor="#10b981" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="gExp" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#fb7185" stopOpacity={0.3} />
                    <stop offset="100%" stopColor="#fb7185" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#eef2f6" vertical={false} />
                <XAxis dataKey="label" tickLine={false} axisLine={false} fontSize={12} stroke="#94a3b8" />
                <YAxis
                  tickFormatter={(v) => formatCompact(v as number)}
                  tickLine={false}
                  axisLine={false}
                  fontSize={12}
                  stroke="#94a3b8"
                  width={48}
                />
                <Tooltip
                  formatter={(v: number) => formatCurrency(v)}
                  contentStyle={{ borderRadius: 12, border: '1px solid #e2e8f0', fontSize: 13 }}
                />
                <Area type="monotone" dataKey="income" stroke="#10b981" strokeWidth={2.5} fill="url(#gInc)" name={tr('type.income')} />
                <Area type="monotone" dataKey="expense" stroke="#fb7185" strokeWidth={2.5} fill="url(#gExp)" name={tr('type.expense')} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="card p-5">
          <h2 className="font-semibold text-ink-900">{tr('dash.expenseStructure')}</h2>
          <p className="text-sm text-ink-500">{tr('dash.expenseStructureSub')}</p>
          {breakdown.length === 0 ? (
            <div className="flex h-56 items-center justify-center text-sm text-ink-400">
              {tr('dash.noExpense')}
            </div>
          ) : (
            <>
              <div className="mx-auto h-44 w-44">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={breakdown}
                      dataKey="total"
                      nameKey="category.name"
                      innerRadius={52}
                      outerRadius={72}
                      paddingAngle={2}
                      stroke="none"
                    >
                      {breakdown.map((b) => (
                        <Cell key={b.category.id} fill={b.category.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      formatter={(v: number) => formatCurrency(v)}
                      contentStyle={{ borderRadius: 12, border: '1px solid #e2e8f0', fontSize: 13 }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <ul className="mt-4 space-y-2">
                {breakdown.slice(0, 4).map((b) => (
                  <li key={b.category.id} className="flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2 text-ink-600">
                      <span className="h-2.5 w-2.5 rounded-full" style={{ background: b.category.color }} />
                      {b.category.name}
                    </span>
                    <span className="font-medium text-ink-800">{Math.round(b.percent)}%</span>
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      </div>

      {/* Bottom row */}
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="card p-5 lg:col-span-2">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="font-semibold text-ink-900">{tr('dash.recent')}</h2>
            <Link to="/transactions" className="flex items-center gap-1 text-sm font-medium text-brand-700 hover:text-brand-800">
              {tr('dash.viewAll')} <ArrowRight size={15} />
            </Link>
          </div>
          <ul className="divide-y divide-ink-100">
            {recent.map((t) => {
              const cat = categoryById(t.categoryId)
              return (
                <li key={t.id} className="flex items-center gap-3 py-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-ink-100 text-ink-500">
                    <CategoryIcon name={cat?.icon ?? 'MoreHorizontal'} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-ink-800">{t.note || cat?.name}</p>
                    <p className="text-xs text-ink-400">{cat?.name} • {formatDate(t.date)}</p>
                  </div>
                  <span className={`tnum text-sm font-semibold ${t.type === 'income' ? 'text-brand-700' : 'text-ink-800'}`}>
                    {t.type === 'income' ? '+' : '-'}{formatCurrency(t.amount)}
                  </span>
                </li>
              )
            })}
          </ul>
        </div>

        <div className="card p-5">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="font-semibold text-ink-900">{tr('nav.budgets')}</h2>
            <Link to="/budgets" className="flex items-center gap-1 text-sm font-medium text-brand-700 hover:text-brand-800">
              {tr('dash.manage')} <ArrowRight size={15} />
            </Link>
          </div>
          {budgetRows.length === 0 ? (
            <div className="flex h-40 items-center justify-center text-sm text-ink-400">
              {tr('dash.noBudget')}
            </div>
          ) : (
            <ul className="space-y-4">
              {budgetRows.map((b) => {
                const over = b.percent > 100
                return (
                  <li key={b.budget.id}>
                    <div className="mb-1.5 flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2 text-ink-700">
                        <CategoryIcon name={b.category.icon} size={16} />
                        {b.category.name}
                      </span>
                      <span className="text-xs text-ink-500">
                        {formatCurrency(b.spent)} / {formatCurrency(b.budget.limit)}
                      </span>
                    </div>
                    <div className="h-2 overflow-hidden rounded-full bg-ink-100">
                      <div
                        className={`h-full rounded-full ${over ? 'bg-rose-500' : 'bg-brand-500'}`}
                        style={{ width: `${Math.min(b.percent, 100)}%` }}
                      />
                    </div>
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
