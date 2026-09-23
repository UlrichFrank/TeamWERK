/**
 * Rückmelde-Matrix einer Mannschaft (GET /api/teams/{id}/rsvp-matrix) —
 * Typen und die reinen Rechenregeln der Tabellenansicht auf `/termine`.
 * Hintergrund: openspec/changes/termin-matrix/design.md.
 */

export type RsvpStatus = 'confirmed' | 'declined' | 'maybe'

export interface MatrixEvent {
  kind: 'training' | 'game'
  id: number
  date: string
  time: string
  event_type: 'training' | 'heim' | 'auswärts' | 'generisch'
  title: string
  cancelled: boolean
}

export interface MatrixCell {
  status: RsvpStatus | null
  is_default: boolean
  unavailable?: boolean
  /** Nur in der Trainer-Sicht gesetzt (erfasste Anwesenheit). */
  present?: boolean
}

export interface MatrixMember {
  member_id: number
  name: string
  extended: boolean
  /** Parallel zu `events` — `cells[i]` gehört zu `events[i]`. */
  cells: MatrixCell[]
}

export interface RsvpMatrix {
  team_id: number
  team_name: string
  events: MatrixEvent[]
  members: MatrixMember[]
}

/** Pfad der Termin-Detailseite (wie die Karten der Liste). */
export function terminDetailPath(ev: MatrixEvent): string {
  if (ev.kind === 'training') return `/termine/training/${ev.id}`
  return `/termine/${ev.event_type === 'generisch' ? 'ereignis' : 'spiel'}/${ev.id}`
}

/** Indizes der Spalten, deren Termin-Typ im Typ-Filter aktiv ist. */
export function visibleColumns(events: MatrixEvent[], types: Set<string>): number[] {
  const out: number[] = []
  events.forEach((ev, i) => { if (types.has(ev.event_type)) out.push(i) })
  return out
}

/**
 * Teilnahme eines Spielers über die sichtbaren Spalten. Zähler: erfasste
 * Anwesenheit, wo vorhanden, sonst Zusage (auch per Voreinstellung). Nenner:
 * sichtbare Termine, die weder abgesagt noch per Serie abgemeldet sind.
 */
export function participation(
  member: MatrixMember,
  events: MatrixEvent[],
  columns: number[],
): { count: number; total: number } {
  let count = 0
  let total = 0
  for (const i of columns) {
    const cell = member.cells[i]
    if (!cell || events[i].cancelled || cell.unavailable) continue
    total++
    const attended = cell.present !== undefined ? cell.present : cell.status === 'confirmed'
    if (attended) count++
  }
  return { count, total }
}

/** „2 (100 %)" — ohne zählbare Termine ein Gedankenstrich. */
export function formatParticipation({ count, total }: { count: number; total: number }): string {
  if (total === 0) return '–'
  return `${count} (${Math.round((count / total) * 100)} %)`
}

/** „13.09." aus "2026-09-13". */
export function formatColumnDate(date: string): string {
  const [, m, d] = date.slice(0, 10).split('-')
  return `${d}.${m}.`
}

/** Serverseitige Obergrenze des Zeitraums (matrixMaxDays in internal/attendance/matrix.go). */
export const MATRIX_MAX_DAYS = 400

function addDays(iso: string, days: number): string {
  const d = new Date(iso + 'T12:00:00Z')
  d.setUTCDate(d.getUTCDate() + days)
  return d.toISOString().slice(0, 10)
}

/**
 * Ladefenster der Matrix: ab heute (mit „Vergangene" ab Saisonstart) bis
 * Saisonende — anders als die Liste ohne 365-Tage-Puffer in beide Richtungen,
 * weil die Spaltenzahl sonst unlesbar wird und der Server bei 400 Tagen deckelt.
 */
export function matrixLoadWindow(
  season: { start_date: string; end_date: string },
  showPast: boolean,
  today: string,
): { from: string; to: string } {
  const from = showPast && season.start_date < today ? season.start_date : today
  let to = season.end_date > from ? season.end_date : addDays(from, 90)
  const cap = addDays(from, MATRIX_MAX_DAYS)
  if (to > cap) to = cap
  return { from, to }
}
