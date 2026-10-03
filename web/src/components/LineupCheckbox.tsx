import type { KeyboardEvent } from 'react'
import { Check } from 'lucide-react'
import { LINEUP_LABEL, LINEUP_SHAPE, LINEUP_SURFACE, type LineupState } from '../lib/lineup'

// Checkbox der Aufstellungs-Spalte mit denselben Flächen wie Kennzeichen und
// Tabellenzellen. Bewusst kein <input type="checkbox">: @tailwindcss/forms füllt
// eine angehakte Checkbox vollfarbig — das wäre ein zweites Grün und ein
// zweiter Rahmenstil (design.md §4).
export default function LineupCheckbox({ state, memberName, onToggle }: {
  state: LineupState
  memberName: string
  /** Fehlt für Spieler und Eltern — dann nur lesend. */
  onToggle?: (checked: boolean) => void
}) {
  const checked = state === 'in'
  const label = `${memberName}: ${LINEUP_LABEL[state]}`
  const toggle = () => onToggle?.(!checked)
  const onKeyDown = (e: KeyboardEvent<HTMLSpanElement>) => {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault()
      toggle()
    }
  }
  return (
    <span
      role="checkbox"
      aria-checked={checked}
      aria-label={label}
      aria-disabled={onToggle ? undefined : true}
      title={LINEUP_LABEL[state]}
      tabIndex={onToggle ? 0 : undefined}
      onClick={onToggle ? toggle : undefined}
      onKeyDown={onToggle ? onKeyDown : undefined}
      className={`inline-flex w-5 h-5 items-center justify-center align-middle ${LINEUP_SHAPE} ${LINEUP_SURFACE[state]} ${
        onToggle ? 'cursor-pointer hover:ring-2 hover:ring-brand-yellow focus:outline-hidden focus:ring-2 focus:ring-brand-yellow' : ''
      }`}
    >
      {checked && <Check className="w-3.5 h-3.5 text-white" aria-hidden="true" />}
    </span>
  )
}
