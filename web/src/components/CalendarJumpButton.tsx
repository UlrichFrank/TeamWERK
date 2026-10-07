import { CalendarDays } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

/**
 * Icon-Aktion „Im Kalender öffnen“ (kalender-sprung). Sitzt in klickbaren
 * Karten, deshalb stoppt sie die Propagation, bevor sie navigiert.
 */
export default function CalendarJumpButton({ to }: { to: string }) {
  const navigate = useNavigate()
  return (
    <button
      type="button"
      onClick={(e) => { e.stopPropagation(); navigate(to) }}
      aria-label="Im Kalender öffnen"
      title="Im Kalender öffnen"
      className="p-2 -m-1 rounded-md text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors shrink-0"
    >
      <CalendarDays className="w-4 h-4" />
    </button>
  )
}
