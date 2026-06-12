import { X } from 'lucide-react'

export interface RegenSummary {
  created: Array<{ date: string; duty_type: string; count: number }>
  reduced: Array<{ date: string; from: string; to: string; count: number }>
  skipped: Array<{ date: string; duty_type: string }>
  notified_users: number[]
  conflicts: Array<{ date: string; duty_type_id: number; event_time: string; game_ids?: number[] }>
}

interface Props {
  summary: RegenSummary
  onDismiss: () => void
}

function isEmpty(s: RegenSummary): boolean {
  return s.created.length === 0 && s.reduced.length === 0 &&
    s.skipped.length === 0 && s.notified_users.length === 0 && s.conflicts.length === 0
}

function formatDate(iso: string): string {
  if (iso.length < 10) return iso
  return `${iso.slice(8, 10)}.${iso.slice(5, 7)}.${iso.slice(0, 4)}`
}

export default function RegenSummaryCard({ summary, onDismiss }: Props) {
  if (isEmpty(summary)) return null
  return (
    <div className="mb-4 p-4 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm relative">
      <button
        onClick={onDismiss}
        aria-label="Schließen"
        className="absolute top-2 right-2 text-brand-text-muted hover:text-brand-text"
      >
        <X className="w-4 h-4" />
      </button>
      <div className="font-semibold text-brand-text mb-2">Folgendes hat sich geändert</div>
      <ul className="space-y-1 text-brand-text">
        {summary.created.map((c, i) => (
          <li key={`c-${i}`}>
            <span className="text-brand-text-muted">{formatDate(c.date)}:</span> {c.count}× „{c.duty_type}" neu angelegt
          </li>
        ))}
        {summary.reduced.map((r, i) => (
          <li key={`r-${i}`}>
            <span className="text-brand-text-muted">{formatDate(r.date)}:</span> {r.count}× „{r.from}" → „{r.to}" (reduziert)
          </li>
        ))}
        {summary.skipped.map((s, i) => (
          <li key={`s-${i}`}>
            <span className="text-brand-text-muted">{formatDate(s.date)}:</span> „{s.duty_type}" übersprungen
          </li>
        ))}
        {summary.notified_users.length > 0 && (
          <li className="text-brand-text-muted italic">
            {summary.notified_users.length} {summary.notified_users.length === 1 ? 'Helfer wurde' : 'Helfer wurden'} benachrichtigt.
          </li>
        )}
        {summary.conflicts.length > 0 && (
          <li className="text-brand-danger">
            {summary.conflicts.length} Konflikt{summary.conflicts.length === 1 ? '' : 'e'} mit manuell gepflegten Slots — bitte prüfen.
          </li>
        )}
      </ul>
    </div>
  )
}
