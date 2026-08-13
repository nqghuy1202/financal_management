import { Languages } from 'lucide-react'
import { useI18n } from '../context/I18nContext'
import { LANGS } from '../lib/i18n'

/** Compact VI/EN switcher. Shows the two options as a segmented control. */
export function LanguageToggle({ className }: { className?: string }) {
  const { lang, setLang } = useI18n()
  return (
    <div className={`inline-flex items-center gap-1 rounded-lg bg-ink-100 p-0.5 ${className ?? ''}`}>
      <Languages size={15} className="ml-1.5 text-ink-400" />
      {LANGS.map((l) => (
        <button
          key={l.code}
          onClick={() => setLang(l.code)}
          className={`rounded-md px-2 py-1 text-xs font-semibold transition ${
            lang === l.code ? 'bg-white text-ink-900 shadow-sm' : 'text-ink-500 hover:text-ink-700'
          }`}
          aria-pressed={lang === l.code}
        >
          {l.short}
        </button>
      ))}
    </div>
  )
}
