import { buildTeamShortNames, type TeamForName } from './teamName'

/**
 * Gemeinsame Semantik des Mannschafts-Filters für Termine, Kalender, Dienste
 * und Mitfahrten — die vier Listen, die dieselbe Frage stellen („welche
 * Mannschaften will ich sehen?") und sie vorher vier Mal verschieden
 * beantwortet haben.
 *
 * Ein Filter ist eine **Menge** von Team-IDs. Die **leere Menge heißt „kein
 * Filter"** und wird in der Oberfläche als „alle angehakt" gezeigt — wie beim
 * Typ-Filter (`EventTypeFilter`), dessen Default ebenfalls „alles aktiv" ist.
 * Daraus folgt die Kernregel: wer die letzte Mannschaft abwählt oder alle
 * anhakt, landet wieder bei „kein Filter" statt bei einer leeren Liste.
 */

export interface TeamFilterOption {
  id: number
  label: string
}

/**
 * `team=3` oder `team=3,7` → Menge. Ungültige Teile werden still verworfen
 * (Spec: ungültige Query-Parameter verhalten sich wie kein Filter, ohne
 * Fehlermeldung).
 */
export function parseTeamIds(raw: string | null | undefined): Set<number> {
  if (!raw) return new Set()
  return new Set(
    raw
      .split(',')
      .map(s => parseInt(s.trim()))
      .filter(n => Number.isFinite(n) && n > 0),
  )
}

/**
 * Die Auswahl, wie die Kästchen sie zeigen: „kein Filter" wird zur
 * vollständigen Menge, damit das Abwählen der ersten Mannschaft aus „alle"
 * eine echte Teilmenge macht.
 */
export function effectiveTeamIds(selected: Set<number>, options: TeamFilterOption[]): Set<number> {
  return selected.size > 0 ? selected : new Set(options.map(o => o.id))
}

/** Ein Kästchen umschalten — auf der effektiven Menge, nicht auf der URL-Menge. */
export function toggleTeamId(effective: Set<number>, teamId: number): Set<number> {
  const next = new Set(effective)
  if (next.has(teamId)) next.delete(teamId)
  else next.add(teamId)
  return next
}

/**
 * Menge → Query-Wert. `null` heißt „Parameter weglassen": sowohl die leere als
 * auch die vollständige Auswahl sind derselbe Zustand „kein Filter". Der
 * Vollständigkeits-Vergleich braucht die Zahl der bekannten Mannschaften;
 * solange die noch nicht geladen sind (`optionCount === 0`), wird nur die leere
 * Menge als Default gewertet.
 */
export function serializeTeamIds(selected: Set<number>, optionCount: number): string | null {
  if (selected.size === 0) return null
  if (optionCount > 0 && selected.size >= optionCount) return null
  return [...selected].join(',')
}

/**
 * Filterprädikat: die leere Menge lässt alles durch, sonst genügt **eine**
 * Überschneidung — ein Termin zweier Mannschaften erscheint also, sobald eine
 * davon gewählt ist.
 */
export function matchesTeamFilter(selected: Set<number>, teamIds: readonly number[] | null | undefined): boolean {
  if (selected.size === 0) return true
  if (!teamIds) return false
  return teamIds.some(id => selected.has(id))
}

/**
 * Optionen für `<TeamFilter>` aus der `/teams`-Antwort. Beschriftet wird mit dem
 * Kurznamen („mA2"); fehlen die Felder dafür, greift der volle Name und zuletzt
 * die ID — eine Mannschaft ohne Beschriftung wäre nicht auswählbar.
 */
export function buildTeamOptions<T extends TeamForName & { name?: string }>(teams: T[]): TeamFilterOption[] {
  const shortNames = buildTeamShortNames(teams)
  return teams.map(t => ({ id: t.id, label: shortNames.get(t.id) ?? t.name ?? `Team ${t.id}` }))
}
