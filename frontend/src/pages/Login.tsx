import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Mail, Lock, Eye, EyeOff, Loader2, Sparkles } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { AuthHero } from './AuthHero'
import { Logo } from '../components/Logo'
import { LanguageToggle } from '../components/LanguageToggle'

export function Login() {
  const { login, demo } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPw, setShowPw] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [demoLoading, setDemoLoading] = useState(false)

  const onDemo = async () => {
    setError('')
    setDemoLoading(true)
    try {
      await demo()
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('err.demoFail'))
    } finally {
      setDemoLoading(false)
    }
  }

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(email, password)
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('err.loginFail'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      <AuthHero />
      <div className="relative flex items-center justify-center px-6 py-12">
        <div className="absolute right-4 top-4">
          <LanguageToggle />
        </div>
        <div className="w-full max-w-sm">
          <div className="mb-8 flex items-center gap-2.5 lg:hidden">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-600 text-white">
              <Logo size={20} />
            </div>
            <span className="text-lg font-bold text-ink-900">HL Company</span>
          </div>

          <h1 className="text-2xl font-bold tracking-tight text-ink-900">{t('login.welcome')}</h1>
          <p className="mt-1.5 text-sm text-ink-500">{t('login.subtitle')}</p>

          <form onSubmit={onSubmit} className="mt-8 space-y-4">
            <div>
              <label className="label" htmlFor="email">{t('field.email')}</label>
              <div className="relative">
                <Mail size={17} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-400" />
                <input
                  id="email"
                  type="email"
                  className="input !pl-9"
                  placeholder={t('placeholder.email')}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  autoComplete="email"
                />
              </div>
            </div>

            <div>
              <label className="label" htmlFor="password">{t('field.password')}</label>
              <div className="relative">
                <Lock size={17} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-400" />
                <input
                  id="password"
                  type={showPw ? 'text' : 'password'}
                  className="input !pl-9 !pr-10"
                  placeholder={t('placeholder.password')}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPw((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-ink-400 hover:text-ink-600"
                  aria-label={t('field.password')}
                >
                  {showPw ? <EyeOff size={17} /> : <Eye size={17} />}
                </button>
              </div>
            </div>

            <div className="flex items-center justify-between text-sm">
              <label className="flex items-center gap-2 text-ink-500">
                <input type="checkbox" className="rounded border-ink-300 text-brand-600 focus:ring-brand-500/30" />
                {t('login.remember')}
              </label>
              <a href="#" className="font-medium text-brand-700 hover:text-brand-800">{t('login.forgot')}</a>
            </div>

            {error && (
              <p role="alert" className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-600">{error}</p>
            )}

            <button type="submit" className="btn-primary w-full" disabled={loading}>
              {loading && <Loader2 size={17} className="animate-spin" />}
              {t('login.submit')}
            </button>
          </form>

          <div className="my-5 flex items-center gap-3 text-xs text-ink-400">
            <span className="h-px flex-1 bg-ink-200" />
            {t('login.or')}
            <span className="h-px flex-1 bg-ink-200" />
          </div>

          <button type="button" className="btn-outline w-full" onClick={onDemo} disabled={demoLoading}>
            {demoLoading ? <Loader2 size={17} className="animate-spin" /> : <Sparkles size={17} />}
            {t('login.demo')}
          </button>

          <p className="mt-6 text-center text-sm text-ink-500">
            {t('login.noAccount')}{' '}
            <Link to="/register" className="font-semibold text-brand-700 hover:text-brand-800">
              {t('login.registerNow')}
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
