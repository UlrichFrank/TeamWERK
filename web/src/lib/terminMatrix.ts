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
  /** Ab hier sperrt der Server Zu-/Absagen für Spieler/Eltern (RFC3339). */
  rsvp_locks_at?: string
  rsvp_require_reason: boolean
}

export interface MatrixCell {
  status: RsvpStatus | null
  is_default: boolean
  unavailable?: boolean
  /** Nur in der Trainer-Sicht gesetzt (erfasste Anwesenheit). */
  present?: boolean
  /** Antwort stammt aus einer erfassten Abwesenheit — nur dort änderbar. */
  locked?: boolean
  /** Nur in antwortbaren Zeilen (eigene, Kinder). */
  reason?: string
}

export interface MatrixMember {
  member_id: number
  name: string
  extended: boolean
  /** Mitglied des Aufrufers. */
  is_self: boolean
  /** Eigene Zeile oder Kind — hier bietet die Tabelle Zu-/Absage an. */
  can_respond: boolean
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

export interface Tally {
  count: number
  total: number
}

/**
 * Teilnahme eines Spielers über die sichtbaren Spalten, getrennt am heutigen
 * Datum (heutige Termine zählen zu „geplant", design.md §7).
 *
 * - `past` (tatsächlich): erfasste Anwesenheit, wo vorhanden, sonst Zusage.
 * - `future` (geplant): Zusagen, auch per Voreinstellung.
 *
 * Nenner beider: Termine, die weder abgesagt noch per Serie abgemeldet sind.
 */
export function participation(
  member: MatrixMember,
  events: MatrixEvent[],
  columns: number[],
  today: string,
): { past: Tally; future: Tally } {
  const past = { count: 0, total: 0 }
  const future = { count: 0, total: 0 }
  for (const i of columns) {
    const cell = member.cells[i]
    if (!cell || events[i].cancelled || cell.unavailable) continue
    const bucket = events[i].date.slice(0, 10) < today ? past : future
    bucket.total++
    const attended = cell.present !== undefined ? cell.present : cell.status === 'confirmed'
    if (attended) bucket.count++
  }
  return { past, future }
}

/** „2 (100 %)" — ohne zählbare Termine ein Gedankenstrich. */
export function formatParticipation({ count, total }: Tally): string {
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

/**
 * Nächster Status für „Zusagen" — dieselbe Regel wie die Karten der Liste:
 * in der eigenen Zeile schaltet eine aktive Zusage auf „Vielleicht" um, eine
 * Voreinstellung wird zur echten Zusage; für Kinder ist „Zusagen" immer eine Zusage.
 */
export function nextConfirmStatus(cell: MatrixCell, isSelf: boolean): RsvpStatus {
  if (!isSelf) return 'confirmed'
  if (cell.is_default) return 'confirmed'
  return cell.status === 'confirmed' ? 'maybe' : 'confirmed'
}

/** Rückmeldefrist verstrichen (ohne Override-Recht). */
export function cutoffLocked(ev: MatrixEvent, canOverride: boolean, now: number = Date.now()): boolean {
  return !canOverride && !!ev.rsvp_locks_at && now >= new Date(ev.rsvp_locks_at).getTime()
}

/**
 * Ob eine Zelle antippbar ist (Dialog öffnet sich). Nach der Rückmeldefrist —
 * also auch für jeden vergangenen Termin — nur noch mit Override-Recht
 * (Trainer/Vorstand); Spieler und Eltern sperrt der Server dort ohnehin (422).
 */
export function isCellRespondable(
  member: MatrixMember,
  ev: MatrixEvent,
  cell: MatrixCell,
  canOverride: boolean,
  now: number = Date.now(),
): boolean {
  return member.can_respond && !ev.cancelled && !cell.unavailable && !cutoffLocked(ev, canOverride, now)
}
