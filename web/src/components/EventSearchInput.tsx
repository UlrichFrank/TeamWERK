import { Search, X } from 'lucide-react'

interface Props {
  value: string
  onChange: (value: string) => void
  /** Benennt die auf DIESER Seite durchsuchten Felder — die Feldmengen sind
   *  pro Seite verschieden (design.md §6), ein Einheitstext würde Felder
   *  versprechen, die es dort nicht gibt. */
  placeholder: string
  /** Im Compact-Modus (schmale Header-Leiste) wird das Feld schmaler. */
  compact?: boolean
  ariaLabel?: string
}

/**
 * Textfilter für die Termin-Ansichten. Reines Eingabefeld ohne eigene
 * Filterlogik — die steckt in lib/eventFilter.ts, die URL-Persistenz im Hook
 * useDebouncedQueryParam.
 *
 * Der iOS-Auto-Zoom ist bereits global in index.css entschärft (input/textarea/
 * select bekommen unter 640px font-size: 16px), hier also nichts zu tun.
 *
 * Höhe: fixe Pixelhöhe statt Padding-getriebener Höhe, damit das Feld exakt so
 * hoch ist wie die Filter-Buttons daneben. Deren Höhe hängt am Modus (im Browser
 * nachgemessen): 30px normal (`py-1.5` + `text-xs`-Zeilenhöhe 1rem + Rahmen),
 * 28px im Compact-Modus, wo die Buttons nur noch das 14px-Icon ohne Label
 * enthalten. Eine Padding-Lösung würde zusätzlich auf Mobile auseinanderlaufen,
 * weil index.css die Schriftgröße dort auf 16px zwingt; bei fixer Höhe zentriert
 * der Browser den Text unabhängig davon. Das unterschreitet bewusst die
 * 44px-Touch-Target-Regel — die Filter-Buttons in derselben Leiste tun das schon,
 * und eine Leiste mit zwei Höhen sah kaputt aus.
 *
 * `::-webkit-search-cancel-button` wird ausgeblendet: Chrome/Safari zeichnen bei
 * `type="search"` ein eigenes Lösch-Kreuz, das direkt neben unserem stand.
 */
export default function EventSearchInput({ value, onChange, placeholder, compact = false, ariaLabel = 'Termine filtern' }: Props) {
  return (
    <div className={`relative ${compact ? 'w-full sm:w-48' : 'w-full sm:w-64'}`}>
      <Search className="w-4 h-4 text-brand-text-subtle absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
      <input
        type="search"
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        aria-label={ariaLabel}
        className={`w-full ${compact ? 'h-7' : 'h-[30px]'} border border-brand-border rounded-md pl-9 pr-9 py-0 text-sm [&::-webkit-search-cancel-button]:appearance-none text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow`}
      />
      {value !== '' && (
        <button
          type="button"
          onClick={() => onChange('')}
          aria-label="Filter leeren"
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-brand-text-subtle hover:text-brand-text transition-colors"
        >
          <X className="w-4 h-4" />
        </button>
      )}
    </div>
  )
}
