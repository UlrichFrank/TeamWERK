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

/**
 * Zustandsflächen der Tabellenansicht von /termine (TerminMatrix). Dort steht
 * in jeder Zelle schon das farbige Rückmelde-Symbol:
 * - „aufgestellt" + zugesagt: volle grüne Fläche, das Symbol darauf weiß (TerminMatrix).
 * - „aufgestellt" ohne Zusage: nur grüner Rahmen, 2 px (`LINEUP_MATRIX_IN_FRAME`), das
 *   Symbol behält seine Farbe — eine Absage auf voller grüner Fläche wäre irreführend.
 * - „nicht aufgestellt": gestrichelt grau, 1,5 px, plus gestrichelte Diagonale von links unten
 *   nach rechts oben hinter dem Symbol (`LineupDiagonal` in TerminMatrix), damit „nicht"
 *   nicht nur an der Strichart hängt.
 * - „offen": gestrichelt grau, 1 px, ohne Diagonale.
 * Außenmaße bleiben gleich: die Box ist border-box mit fester Höhe/Breite, der
 * dickere Rahmen geht nach innen.
 */
export const LINEUP_MATRIX_SURFACE: Record<LineupState, string> = {
  in: 'bg-brand-green border-transparent',
  out: 'border-[1.5px] border-dashed border-brand-text-subtle bg-transparent',
  open: 'border-dashed border-brand-text-subtle bg-transparent',
}

/** „aufgestellt" ohne Zusage in der Tabellenansicht: nur grüner Rahmen. */
export const LINEUP_MATRIX_IN_FRAME = 'border-2 border-brand-green bg-transparent'

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
