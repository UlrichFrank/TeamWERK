import { useRef, useState } from 'react'
import { ChevronDown, Users } from 'lucide-react'
import { useDismissOnOutside } from '../hooks/useDismissOnOutside'
import { type TeamFilterOption } from '../lib/teamFilter'
import { HEADER_CTRL, HEADER_CTRL_ICON, HEADER_NEUTRAL, HEADER_PRIMARY } from '../lib/buttonStyles'

interface Props {
  teams: TeamFilterOption[]
  /**
   * Die **effektive** Auswahl, nicht der URL-Zustand: „kein Filter" kommt hier
   * als vollständige Menge an, damit alle Kästchen angehakt sind. Genau so
   * verhält sich der Typ-Filter, dessen Default-Menge ebenfalls „alle" ist.
   */
  active: Set<number>
  onToggle: (teamId: number) => void
  /** Compact-Modus (schmale Kopfzeile): nur Icon + Zähler, kein Wort-Label. */
  compact: boolean
  ariaLabel?: string
}

/**
 * Mehrfachauswahl der Mannschaften als Dropdown mit Checkboxen — dieselbe Form
 * wie der Typ-Filter (`EventTypeFilter`) im Compact-Modus.
 *
 * Anders als dort ist das Dropdown auch auf breiten Bildschirmen ein Dropdown:
 * Mannschaften sind dynamisch und je nach Nutzer bis zu einem Dutzend, als
 * Chip-Reihe wäre die Kopfzeile nicht mehr lesbar.
 *
 * Die Semantik „alles angehakt = kein Filter" lebt beim Aufrufer (er entscheidet,
 * was in die URL geht); diese Komponente zeigt nur an und meldet Umschaltungen.
 */
export default function TeamFilter({ teams, active, onToggle, compact, ariaLabel = 'Mannschafts-Filter' }: Props) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useDismissOnOutside(open, ref, () => setOpen(false))

  // Ohne Mannschaften gibt es nichts zu filtern — ein toter Knopf wäre eine
  // Behauptung, es gäbe eine Auswahl. (Seiten, die den Filter erst ab zwei
  // Mannschaften zeigen, entscheiden das zusätzlich selbst.)
  if (teams.length === 0) return null

  const activeCount = teams.filter(t => active.has(t.id)).length
  const allActive = activeCount === teams.length

  const label = allActive
    ? 'Teams'
    : activeCount === 1
      ? (teams.find(t => active.has(t.id))?.label ?? 'Teams')
      : `${activeCount} Teams`

  return (
    <div className="relative shrink-0" ref={ref}>
      <button
        onClick={() => setOpen(o => !o)}
        aria-label={ariaLabel}
        aria-expanded={open}
        className={`${compact ? HEADER_CTRL_ICON : HEADER_CTRL} ${allActive ? HEADER_NEUTRAL : HEADER_PRIMARY}`}
      >
        <Users className="w-3.5 h-3.5" />
        {compact
          ? !allActive && <span>{activeCount}/{teams.length}</span>
          : <span>{label}</span>}
        <ChevronDown className="w-3.5 h-3.5" />
      </button>
      {open && (
        <div className="absolute left-0 top-full mt-1 z-20 bg-white border border-brand-border rounded-md shadow-lg py-1 min-w-[160px] max-h-72 overflow-y-auto">
          {teams.map(t => (
            <label
              key={t.id}
              className="flex items-center gap-2 px-3 py-2 text-sm text-brand-text hover:bg-brand-table-select cursor-pointer"
            >
              <input
                type="checkbox"
                checked={active.has(t.id)}
                onChange={() => onToggle(t.id)}
                className="w-4 h-4 accent-brand-yellow"
              />
              <span>{t.label}</span>
            </label>
          ))}
        </div>
      )}
    </div>
  )
}
