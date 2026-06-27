import { AlertTriangle } from 'lucide-react'

type EventNoteIndicatorProps = {
  variant: 'icon' | 'inline'
  note: string
  className?: string
}

/**
 * Markiert einen Termin mit einem Hinweis. Rendert nichts, wenn kein Hinweis
 * vorhanden ist (note leer/whitespace).
 *
 * - `icon`   → kompaktes AlertTriangle, voller Text als title-Tooltip.
 * - `inline` → AlertTriangle + voller Hinweistext in einer eigenen Zeile.
 */
export default function EventNoteIndicator({ variant, note, className = '' }: EventNoteIndicatorProps) {
  if (note.trim() === '') return null

  if (variant === 'icon') {
    return (
      <span
        aria-label="Hinweis vorhanden"
        title={`Hinweis: ${note}`}
        className={`inline-flex items-center ${className}`}
      >
        <AlertTriangle className="w-4 h-4 text-brand-danger" />
      </span>
    )
  }

  return (
    <div className={`flex items-start gap-2 text-sm text-brand-danger ${className}`}>
      <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
      <span className="whitespace-pre-wrap">{note}</span>
    </div>
  )
}
