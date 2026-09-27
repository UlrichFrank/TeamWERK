import { LINEUP_LABEL, LINEUP_SHAPE, LINEUP_SURFACE, LINEUP_TEXT, type LineupState } from '../lib/lineup'

// Text-Kennzeichen des Aufstellungsstatus (ohne Symbol, immer gleich breit).
// In der Knopfzeile (`inRow`) übernimmt es per self-stretch die Höhe der
// Zu-/Absage-Knöpfe, frei stehend (Kartenkopf) dieselbe Höhe fest.
export default function LineupBadge({ state, text, inRow = false }: {
  state: LineupState | undefined
  /** Ersetzt die Standardbezeichnung, z. B. „5 aufgestellt" im Kartenkopf. */
  text?: string
  inRow?: boolean
}) {
  if (!state) return null
  return (
    <span
      data-lineup={state}
      className={`inline-flex items-center justify-center w-32 shrink-0 text-xs font-medium ${inRow ? 'ml-auto self-stretch' : 'h-[26px]'} ${LINEUP_SHAPE} ${LINEUP_SURFACE[state]} ${LINEUP_TEXT[state]}`}
    >
      {text ?? LINEUP_LABEL[state]}
    </span>
  )
}
