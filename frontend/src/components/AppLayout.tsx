import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  ArrowLeftRight,
  PiggyBank,
  LogOut,
  Menu,
  X,
  Search,
  Bell,
} from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { Logo } from './Logo'
import { LanguageToggle } from './LanguageToggle'

const navItems = [
  { to: '/', labelKey: 'nav.dashboard', icon: LayoutDashboard, end: true },
  { to: '/transactions', labelKey: 'nav.transactions', icon: ArrowLeftRight, end: false },
  { to: '/budgets', labelKey: 'nav.budgets', icon: PiggyBank, end: false },
]

export function AppLayout() {
  const { user, logout } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const [mobileOpen, setMobileOpen] = useState(false)

  const initials = (user?.name ?? 'U')
    .split(' ')
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const SidebarContent = (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2.5 px-6 py-6">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-600 text-white">
          <Logo size={20} />
        </div>
        <span className="text-lg font-bold tracking-tight text-ink-900">HL Company</span>
      </div>

      <nav className="flex-1 space-y-1 px-4">
        {navItems.map(({ to, labelKey, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            onClick={() => setMobileOpen(false)}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition ${
                isActive
                  ? 'bg-brand-50 text-brand-700'
                  : 'text-ink-500 hover:bg-ink-100 hover:text-ink-800'
              }`
            }
          >
            <Icon size={19} />
            {t(labelKey)}
          </NavLink>
        ))}
      </nav>

      <div className="border-t border-ink-100 p-4">
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-ink-500 transition hover:bg-rose-50 hover:text-rose-600"
        >
          <LogOut size={19} />
          {t('action.logout')}
        </button>
      </div>
    </div>
  )

  return (
    <div className="min-h-screen bg-ink-50">
      {/* Desktop sidebar */}
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r border-ink-200 bg-white lg:block">
        {SidebarContent}
      </aside>

      {/* Mobile drawer */}
      {mobileOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <div className="absolute inset-0 bg-ink-900/40" onClick={() => setMobileOpen(false)} />
          <aside className="absolute inset-y-0 left-0 w-64 bg-white shadow-xl">{SidebarContent}</aside>
        </div>
      )}

      <div className="lg:pl-64">
        {/* Topbar */}
        <header className="sticky top-0 z-30 flex h-16 items-center gap-4 border-b border-ink-200 bg-white/80 px-4 backdrop-blur lg:px-8">
          <button
            className="btn-icon -ml-2 lg:hidden"
            onClick={() => setMobileOpen((v) => !v)}
            aria-label="Menu"
          >
            {mobileOpen ? <X size={20} /> : <Menu size={20} />}
          </button>

          <div className="relative hidden max-w-xs flex-1 sm:block">
            <Search size={17} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-400" />
            <input className="input !pl-9" placeholder={t('topbar.search')} />
          </div>

          <div className="ml-auto flex items-center gap-3">
            <LanguageToggle className="hidden sm:inline-flex" />
            <button className="btn-icon relative" aria-label="Notifications">
              <Bell size={19} />
              <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-brand-500" />
            </button>
            <div className="flex items-center gap-2.5">
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-100 text-sm font-semibold text-brand-700">
                {initials}
              </div>
              <div className="hidden text-sm sm:block">
                <p className="font-semibold leading-tight text-ink-800">{user?.name}</p>
                <p className="text-xs leading-tight text-ink-400">{user?.email}</p>
              </div>
            </div>
          </div>
        </header>

        <main className="px-4 py-6 lg:px-8 lg:py-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
