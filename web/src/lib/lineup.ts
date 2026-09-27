/**
 * Aufstellungsstatus eines Spielers für ein Spiel — eine Quelle für Bezeichnung
 * und Darstellung in Liste, Tabelle und Detailseite von /termine.
 * Hintergrund: openspec/changes/aufstellung-status-termine/design.md §4.
 *
 * Regeln: 1 px Rahmen und rounded-md überall; gefüllte Zustände tragen einen
 * transparenten Rahmen, damit alle drei dieselben Außenmaße haben. „offen" ist
 * innen vollständig transparent, nur der gestrichelte Rahmen ist sichtbar.
 */

export type LineupState = 'in' | 'out' | 'open'

export const LINEUP_LABEL: Record<LineupState, string> = {
  in: 'aufgestellt',
  out: 'nicht aufgestellt',
  open: 'Aufstellung offen',
}

export const LINEUP_SHAPE = 'rounded-md border'

export const LINEUP_SURFACE: Record<LineupState, string> = {
  in: 'bg-brand-green border-transparent',
  out: 'bg-brand-border border-transparent',
  open: 'bg-transparent border-dashed border-brand-text-subtle',
}

/** Schrift (und Haken) auf der jeweiligen Fläche. */
export const LINEUP_TEXT: Record<LineupState, string> = {
  in: 'text-white',
  out: 'text-brand-text',
  open: 'text-brand-text',
}

/** Liest einen Wert aus der API; alles Unbekannte heißt „kein Status". */
export function toLineupState(v: unknown): LineupState | undefined {
  return v === 'in' || v === 'out' || v === 'open' ? v : undefined
}
