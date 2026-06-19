// Labels und Reihenfolge der Beitragskategorien (3 Stück).
export const BEITRAGS_KATEGORIEN = ['aktiv_ohne', 'aktiv_mit', 'passiv'] as const

export type BeitragsKategorie = (typeof BEITRAGS_KATEGORIEN)[number]

export const KATEGORIE_LABEL: Record<string, string> = {
  aktiv_ohne: 'Aktiv (ohne Stammverein)',
  aktiv_mit: 'Aktiv (mit Stammverein)',
  passiv: 'Passiv',
}

export function kategorieLabel(k: string): string {
  return KATEGORIE_LABEL[k] ?? k
}
