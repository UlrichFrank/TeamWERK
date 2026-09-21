// Hervorhebung der eigenen Mannschaft und der eigenen Spieler in den Tabellen
// der Staffel-Ansicht.
//
// Eine gemeinsame Fundstelle, weil die Auszeichnung in neun Reitern gleich
// aussehen muss und aus drei Teilen besteht, von denen keiner allein genügt:
//
//   font-semibold      — fällt auf, reicht aber nicht: in der Torschützen-,
//                        Angriffs- und Fair-Play-Tabelle ist die Wertspalte
//                        ohnehin fett, eine nur fette Zeile wäre dort kaum vom
//                        Normalfall zu unterscheiden.
//   brand-table-select — die Zeilenmarkierung, die den Unterschied trägt.
//   aria-current       — weil Schriftschnitt und Hintergrund an einem
//                        Screenreader vorbeigehen. Deshalb lautet die
//                        Anforderung "nicht ausschließlich visuell".

/** Klassen der hervorgehobenen Zeile; leer, wenn sie es nicht ist. */
export const highlightClass = (own: boolean): string =>
  own ? 'bg-brand-table-select font-semibold' : ''

/**
 * Attribute der hervorgehobenen Zeile — als Objekt zum Spreaden, damit
 * aria-current nicht an einzelnen Stellen vergessen wird.
 */
export const highlightProps = (own: boolean) =>
  own ? ({ className: highlightClass(true), 'aria-current': true as const }) : {}

/**
 * Prüft, ob eine Mannschaft die eigene ist. Der Vergleich läuft über die
 * Schreibweise des Verbands, die der Server in Affiliation.teamNames liefert —
 * sie stammt aus der Verknüpfung mit dem eigenen Spieltermin, nicht aus einem
 * Namensabgleich (design.md §10).
 */
export const isOwnTeam = (teamNames: string[], team: string): boolean =>
  teamNames.includes(team)

/** Prüft, ob eine Spielerzeile dem Nutzer oder einem seiner Kinder gehört. */
export const isOwnPlayer = (playerIds: number[], playerId: number): boolean =>
  playerIds.includes(playerId)
