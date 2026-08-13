import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Mail, Lock, User as UserIcon, Eye, EyeOff, Loader2 } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { AuthHero } from './AuthHero'
import { Logo } from '../components/Logo'
import { LanguageToggle } from '../components/LanguageToggle'

export function Register() {
  const { register } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPw, setShowPw] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (password.length < 6) {
      setError(t('err.pwShort'))
      return
    }
    setLoading(true)
    try {
      await register(name, email, password)
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('err.registerFail'))
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

          <h1 className="text-2xl font-bold tracking-tight text-ink-900">{t('register.title')}</h1>
          <p className="mt-1.5 text-sm text-ink-500">{t('register.subtitle')}</p>

          <form onSubmit={onSubmit} className="mt-8 space-y-4">
            <div>
              <label className="label" htmlFor="name">{t('field.name')}</label>
              <div className="relative">
                <UserIcon size={17} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-400" />
                <input
                  id="name"
                  className="input !pl-9"
                  placeholder={t('placeholder.name')}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  autoComplete="name"
                />
              </div>
            </div>

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
                  placeholder={t('register.pwHint')}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
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

            {error && (
              <p role="alert" className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-600">{error}</p>
            )}

            <button type="submit" className="btn-primary w-full" disabled={loading}>
              {loading && <Loader2 size={17} className="animate-spin" />}
              {t('register.submit')}
            </button>
          </form>

          <p className="mt-6 text-center text-sm text-ink-500">
            {t('register.haveAccount')}{' '}
            <Link to="/login" className="font-semibold text-brand-700 hover:text-brand-800">
              {t('login.submit')}
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
