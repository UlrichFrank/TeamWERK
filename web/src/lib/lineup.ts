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
 * in jeder Zelle schon das farbige Rückmelde-Symbol; eine volle grüne Fläche
 * überdeckte es. Deshalb nur Rahmen, keine Fläche:
 * - „aufgestellt": durchgezogener grüner Rahmen, 2 px — dicker als der
 *   gestrichelte, weil Grün auf Weiß schwächer trägt als Grau.
 * - „nicht aufgestellt": gestrichelt grau, ebenfalls 2 px, plus Diagonale von links unten nach
 *   rechts oben (hinter dem Symbol), damit „nicht" nicht nur an der Strichart hängt.
 * - „offen": gestrichelt grau, 1 px, ohne Diagonale.
 * Außenmaße bleiben gleich: die Box ist border-box mit fester Höhe/Breite, der
 * dickere Rahmen geht nach innen.
 */
export const LINEUP_MATRIX_SURFACE: Record<LineupState, string> = {
  in: 'border-2 border-brand-green bg-transparent',
  out: 'border-2 border-dashed border-brand-text-subtle bg-[linear-gradient(to_bottom_right,transparent_calc(50%-0.75px),theme(colors.brand.text-subtle)_calc(50%-0.75px),theme(colors.brand.text-subtle)_calc(50%+0.75px),transparent_calc(50%+0.75px))]',
  open: 'border-dashed border-brand-text-subtle bg-transparent',
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
