import { ShieldCheck, TrendingUp, Wallet } from 'lucide-react'
import { Logo } from '../components/Logo'
import { useI18n } from '../context/I18nContext'

export function AuthHero() {
  const { t } = useI18n()
  return (
    <div className="relative hidden overflow-hidden bg-brand-700 lg:flex lg:flex-col lg:justify-between lg:p-12">
      {/* decorative blobs */}
      <div className="pointer-events-none absolute -right-24 -top-24 h-72 w-72 rounded-full bg-brand-500/40 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-32 -left-16 h-80 w-80 rounded-full bg-brand-400/30 blur-3xl" />

      <div className="relative flex items-center gap-2.5 text-white">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-white/15 backdrop-blur">
          <Logo size={20} />
        </div>
        <span className="text-lg font-bold">HL Company</span>
      </div>

      <div className="relative text-white">
        <h2 className="whitespace-pre-line text-3xl font-bold leading-tight">{t('hero.title')}</h2>
        <p className="mt-4 max-w-md text-brand-100">{t('hero.subtitle')}</p>

        <div className="mt-10 space-y-4">
          {[
            { icon: Wallet, text: t('hero.f1') },
            { icon: TrendingUp, text: t('hero.f2') },
            { icon: ShieldCheck, text: t('hero.f3') },
          ].map(({ icon: Icon, text }) => (
            <div key={text} className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/15 text-white backdrop-blur">
                <Icon size={18} />
              </div>
              <span className="text-sm text-brand-50">{text}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="relative text-xs text-brand-200">{t('hero.footer')}</div>
    </div>
  )
}
