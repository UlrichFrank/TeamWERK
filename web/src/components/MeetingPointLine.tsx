import { Flag } from 'lucide-react'
import { meetingIsPreviousDay } from '../lib/meeting'

type MeetingPointLineProps = {
  gameDate: string
  meetTime: string | null | undefined
  meetDate: string | null | undefined
  meetPlace?: string
  /** Detail/Modal schreiben „13:30 Uhr", die kompakte Terminliste „13:30". */
  withUhr?: boolean
  className?: string
}

/**
 * Treffzeit eines Spiels (spiel-treffpunkt): „Treffen 13:30 · Parkplatz".
 * Rendert nichts ohne Treffzeit. Liegt das Treffen vor dem Spieltag (Anwurf
 * wurde verlegt), steht „(Vortag)" dabei.
 */
export default function MeetingPointLine({ gameDate, meetTime, meetDate, meetPlace, withUhr, className = '' }: MeetingPointLineProps) {
  if (!meetTime) return null
  const time = `${meetTime}${withUhr ? ' Uhr' : ''}${meetingIsPreviousDay(gameDate, meetDate) ? ' (Vortag)' : ''}`
  return (
    <div className={`flex items-start gap-2 text-sm text-brand-text-muted ${className}`}>
      <Flag className="w-4 h-4 mt-0.5 shrink-0" />
      <span className="min-w-0 break-words">
        Treffen {time}
        {meetPlace ? ` · ${meetPlace}` : ''}
      </span>
    </div>
  )
}
