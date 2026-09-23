import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Calendar, Circle, Dumbbell, HelpCircle, Home, Minus, Plane, ThumbsDown, ThumbsUp, UserCheck, UserX } from 'lucide-react'
import { getEventColors } from '../lib/eventColors'
import {
  formatColumnDate,
  formatParticipation,
  isCellRespondable,
  participation,
  terminDetailPath,
  type MatrixCell,
  type RsvpMatrix,
} from '../lib/terminMatrix'

const TYPE_ICON = {
  training: Dumbbell,
  heim: Home,
  'auswärts': Plane,
  generisch: Calendar,
} as const

const TYPE_LABEL = {
  training: 'Training',
  heim: 'Heimspiel',
  'auswärts': 'Auswärtsspiel',
  generisch: 'Sonstiges',
} as const

/** Symbol + Beschriftung einer Zelle. Anwesenheit (Trainer-Sicht) geht vor der Rückmeldung. */
function cellView(cell: MatrixCell): { icon: ReactNode; label: string } {
  const cls = 'w-4 h-4 mx-auto'
  if (cell.present === true) return { icon: <UserCheck className={`${cls} text-brand-green`} />, label: 'anwesend' }
  if (cell.present === false) return { icon: <UserX className={`${cls} text-brand-danger`} />, label: 'gefehlt' }
  if (cell.unavailable) return { icon: <Minus className={`${cls} text-brand-text-subtle`} />, label: 'für die Serie abgemeldet' }
  const faded = cell.is_default ? ' opacity-40' : ''
  const suffix = cell.is_default ? ' (Voreinstellung)' : ''
  switch (cell.status) {
    case 'confirmed':
      return { icon: <ThumbsUp className={`${cls} text-brand-green${faded}`} />, label: `zugesagt${suffix}` }
    case 'declined':
      return { icon: <ThumbsDown className={`${cls} text-brand-danger${faded}`} />, label: `abgesagt${suffix}` }
    case 'maybe':
      return { icon: <HelpCircle className={`${cls} text-brand-warning`} />, label: 'vielleicht' }
    default:
      return { icon: <Circle className={`${cls} text-brand-text-subtle`} />, label: 'keine Rückmeldung' }
  }
}

interface Props {
  matrix: RsvpMatrix
  /** Indizes der sichtbaren Spalten (Typ-Filter), siehe `visibleColumns`. */
  columns: number[]
  /** "YYYY-MM-DD" — trennt „Bisher" von „Geplant". */
  today: string
  /** Tipp auf eine antwortbare Zelle (eigene Zeile, Kind). */
  onCellClick?: (row: number, col: number) => void
}

export default function TerminMatrix({ matrix, columns, today, onCellClick }: Props) {
  const { events, members } = matrix
  const withPresence = members.some(m => m.cells.some(c => c.present !== undefined))
  // Abgesagte Termine stehen als durchgestrichene Spalte in der Tabelle, zählen
  // hier aber nicht mit — wie im Nenner der Teilnahme-Quote.
  const held = columns.filter(i => !events[i].cancelled)
  const counts = {
    training: held.filter(i => events[i].event_type === 'training').length,
    spiele: held.filter(i => events[i].event_type === 'heim' || events[i].event_type === 'auswärts').length,
    sonstige: held.filter(i => events[i].event_type === 'generisch').length,
  }

  if (members.length === 0) {
    return (
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-8 text-center">
        <p className="text-brand-text-muted">Für diese Mannschaft ist in der aktiven Saison kein Kader hinterlegt.</p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <p className="p-3 bg-brand-surface-card rounded-lg text-sm text-brand-text-muted">
        <span className="font-semibold text-brand-text">{counts.training}</span> Trainings
        {' | '}<span className="font-semibold text-brand-text">{counts.spiele}</span> Spiele
        {' | '}<span className="font-semibold text-brand-text">{counts.sonstige}</span> Sonstige
      </p>

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden">
        <div className="overflow-x-auto">
          <table className="border-separate border-spacing-0 text-sm">
            <thead>
              <tr>
                <th
                  scope="col"
                  className="sticky left-0 z-10 bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left border-b border-r border-brand-border-subtle"
                >
                  Spieler
                </th>
                <th scope="col" title="Vergangene Termine: erfasste Anwesenheit, sonst Zusage" className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-3 py-3 text-left border-b border-brand-border-subtle whitespace-nowrap">
                  Bisher
                </th>
                <th scope="col" title="Heutige und künftige Termine: Zusagen" className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-3 py-3 text-left border-b border-r border-brand-border-subtle whitespace-nowrap">
                  Geplant
                </th>
                {columns.map(i => {
                  const ev = events[i]
                  const Icon = TYPE_ICON[ev.event_type] ?? Calendar
                  const title = `${TYPE_LABEL[ev.event_type] ?? ''} ${formatColumnDate(ev.date)} ${ev.time}${ev.title ? ` – ${ev.title}` : ''}${ev.cancelled ? ' (abgesagt)' : ''}`
                  return (
                    <th
                      key={`${ev.kind}-${ev.id}`}
                      scope="col"
                      className="bg-brand-surface-card px-2 py-2 border-b border-brand-border-subtle font-normal"
                    >
                      <Link
                        to={terminDetailPath(ev)}
                        title={title}
                        aria-label={title}
                        className={`flex flex-col items-center gap-0.5 text-xs text-brand-text-muted hover:text-brand-text ${ev.cancelled ? 'opacity-50 line-through' : ''}`}
                      >
                        <Icon className={`w-4 h-4 ${getEventColors(ev.event_type).card.icon}`} />
                        <span className="whitespace-nowrap">{formatColumnDate(ev.date)}</span>
                      </Link>
                    </th>
                  )
                })}
              </tr>
            </thead>
            <tbody>
              {members.map((m, rowIdx) => {
                const prev = members[rowIdx - 1]
                const groupStart = m.extended && (!prev || !prev.extended)
                const part = participation(m, events, columns, today)
                const groupBorder = groupStart ? 'border-t-2 border-t-brand-border' : ''
                return (
                  <tr key={m.member_id} className="group">
                    <th
                      scope="row"
                      className={`sticky left-0 z-10 bg-brand-surface-card group-hover:bg-brand-table-select px-4 py-2.5 text-left font-medium text-brand-text border-b border-r border-brand-border-subtle max-w-[9rem] sm:max-w-[14rem] truncate ${groupBorder}`}
                      title={m.extended ? `${m.name} (erweiterter Kader)` : m.name}
                    >
                      {m.name}
                      {m.is_self && <span className="ml-1 text-xs font-normal text-brand-text-subtle">(ich)</span>}
                      {m.extended && <span className="ml-1 text-xs font-normal text-brand-text-subtle">erw.</span>}
                    </th>
                    <td className={`px-3 py-2.5 text-brand-text whitespace-nowrap border-b border-brand-border-subtle group-hover:bg-brand-table-select ${groupBorder}`}>
                      {formatParticipation(part.past)}
                    </td>
                    <td className={`px-3 py-2.5 text-brand-text whitespace-nowrap border-b border-r border-brand-border-subtle group-hover:bg-brand-table-select ${groupBorder}`}>
                      {formatParticipation(part.future)}
                    </td>
                    {columns.map(i => {
                      const ev = events[i]
                      const cell = m.cells[i]
                      const { icon, label } = ev.cancelled
                        ? { icon: null, label: 'Termin abgesagt' }
                        : cellView(cell)
                      const tdClass = `px-2 py-2.5 text-center border-b border-brand-border-subtle group-hover:bg-brand-table-select ${groupBorder}`
                      if (onCellClick && isCellRespondable(m, ev, cell)) {
                        return (
                          <td key={`${ev.kind}-${ev.id}`} className={`${tdClass} bg-brand-yellow/10`}>
                            <button
                              type="button"
                              onClick={() => onCellClick(rowIdx, i)}
                              title={`${label} – ändern`}
                              aria-label={`${m.name}, ${formatColumnDate(ev.date)}: ${label} – ändern`}
                              className="w-full min-h-[28px] flex items-center justify-center rounded-md hover:ring-2 hover:ring-brand-yellow focus:outline-none focus:ring-2 focus:ring-brand-yellow"
                            >
                              {icon}
                            </button>
                          </td>
                        )
                      }
                      return (
                        <td key={`${ev.kind}-${ev.id}`} title={label} aria-label={label} className={tdClass}>
                          {icon}
                        </td>
                      )
                    })}
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
        {columns.length === 0 && (
          <p className="px-4 py-6 text-sm text-brand-text-muted text-center">Keine Termine im gewählten Zeitraum.</p>
        )}
      </div>

      <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-brand-text-muted" aria-label="Legende">
        <li className="flex items-center gap-1"><ThumbsUp className="w-4 h-4 text-brand-green" /> zugesagt</li>
        <li className="flex items-center gap-1"><ThumbsDown className="w-4 h-4 text-brand-danger" /> abgesagt</li>
        <li className="flex items-center gap-1"><HelpCircle className="w-4 h-4 text-brand-warning" /> vielleicht</li>
        <li className="flex items-center gap-1"><Circle className="w-4 h-4 text-brand-text-subtle" /> keine Rückmeldung</li>
        <li className="flex items-center gap-1"><ThumbsUp className="w-4 h-4 text-brand-green opacity-40" /> Voreinstellung</li>
        <li className="flex items-center gap-1"><Minus className="w-4 h-4 text-brand-text-subtle" /> für die Serie abgemeldet</li>
        {onCellClick && members.some(m => m.can_respond) && (
          <li className="flex items-center gap-1"><span className="inline-block w-4 h-4 rounded bg-brand-yellow/30" /> antippen zum Zu-/Absagen</li>
        )}
        {withPresence && (
          <>
            <li className="flex items-center gap-1"><UserCheck className="w-4 h-4 text-brand-green" /> anwesend</li>
            <li className="flex items-center gap-1"><UserX className="w-4 h-4 text-brand-danger" /> gefehlt</li>
          </>
        )}
      </ul>
    </div>
  )
}
