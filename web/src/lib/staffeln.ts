import { api } from './api'

// Die Typen spiegeln die Antworten von internal/gamestats. Feldnamen sind dort
// per JSON-Tag festgelegt und folgen camelCase.

export interface Staffel {
  /** 0, solange zu dieser Zuordnung noch nichts abgerufen wurde. */
  id: number
  code: string
  name: string
  teamName: string
  /** Kurzname der Mannschaft ("mB1"), wie in Kalender und Terminen. */
  teamShort: string
  kaderId: number
  polled: boolean
}

export interface TableRow {
  Position: number
  TeamName: string
  Games: number
  Won: number
  Drawn: number
  Lost: number
  GoalsFor: number
  GoalsAgainst: number
  PointsPlus: number
  PointsMinus: number
}

export interface ScheduleGame {
  ID: number
  GameNo: string
  SGID: string
  GameID: number | null
  Date: string
  Time: string
  HomeTeam: string
  GuestTeam: string
  HomeGoals: number | null
  GuestGoals: number | null
  HomeGoalsHT: number | null
  GuestGoalsHT: number | null
  HallNumber: string
  /** Aufgelöste Halle aus /veranstaltungsorte; null, wenn keine die Nummer trägt. */
  Venue: { name: string; street: string; city: string; postal_code: string } | null
  HasReport: boolean
}

export interface PlayerStat {
  playerId: number
  memberId: number | null
  name: string
  teamName: string
  /** Zahl der Berichte, in deren Mannschaftsliste der Spieler steht. */
  games: number
  goals: number
  sevenMAttempts: number
  sevenMGoals: number
  /** Serverseitig gebildet, damit die Subtraktion nicht an zwei Stellen lebt. */
  sevenMMissed: number
  twoMin: number
  warnings: number
  disq: number
  fairPlayScore: number
}

// --- Kreuztabelle ---------------------------------------------------------

export interface CrossCell {
  bwhvGameId: number
  /** true = Ergebnis erfasst. Ein 0:0 ist gespielt, kein fehlendes Ergebnis. */
  played: boolean
  homeGoals: number | null
  guestGoals: number | null
  /** Nur bei noch nicht gespielten Begegnungen gesetzt. */
  date: string
}

export interface CrossRow {
  team: string
  /** Parallel zu CrossTable.teams; null = Diagonale oder keine Begegnung. */
  cells: (CrossCell | null)[]
}

export interface CrossTable {
  teams: string[]
  rows: CrossRow[]
}

// --- Tabellenverlauf ------------------------------------------------------

export interface ProgressionEntry {
  team: string
  rank: number
  points: number
  games: number
  goalsFor: number
  goalDiff: number
}

export interface ProgressionDay {
  date: string
  entries: ProgressionEntry[]
}

// --- Mannschafts-Ranglisten -----------------------------------------------

export interface GoalDistribution {
  players: number
  average: number
  median: number
  gini: number
}

export interface TeamStat {
  team: string
  /** Begegnungen mit Ergebnis — Grundlage von Toren, Angriff, Verteidigung. */
  games: number
  goalsFor: number
  goalsAgainst: number
  goalDiff: number
  /** Ausgewertete Spielberichte — Grundlage von Fair-Play und Verteilung. */
  reportGames: number
  twoMin: number
  yellow: number
  red: number
  blue: number
  /** null ohne Bericht: die Mannschaft ist nicht straffrei, sondern unbekannt. */
  fairPlayScore: number | null
  distribution: GoalDistribution | null
}

export interface FairPlayWeights {
  yellow: number
  twoMin: number
  red: number
  blue: number
}

export interface TeamStats {
  fairPlayWeights: FairPlayWeights
  teams: TeamStat[]
}

// --- Schiedsrichter -------------------------------------------------------

export interface RefereeStat {
  name: string
  games: number
  twoMin: number
  yellow: number
  red: number
  blue: number
  /** Der Name musste von dem des Gespannpartners geraten werden. */
  uncertain: boolean
}

// --- Eigene Zugehörigkeit -------------------------------------------------

/**
 * Mannschaften und Spielerzeilen, die dem Nutzer zuzurechnen sind. Leer heißt
 * "nicht belegbar" — die Zuordnung wird aus der Verknüpfung zwischen
 * BWHV-Begegnung und eigenem Spieltermin abgeleitet, nie über Namen geraten.
 */
export interface Affiliation {
  teamNames: string[]
  playerIds: number[]
}

/** Werte eines Spielers in einer Begegnung — und, in den Summen, die einer
 * ganzen Mannschaft. Ohne "Blau": der Parser liest keine blauen Karten, und
 * eine Spalte, die immer leer ist, behauptet eine Messung. */
export interface MatrixCell {
  goals: number
  sevenMAttempts: number
  sevenMGoals: number
  twoMin: number
  warnings: number
  disq: number
}

/** Eine Spalte der Spielmatrix: eine gespielte Begegnung. */
export interface MatrixGame {
  bwhvGameId: number
  date: string
  homeTeam: string
  guestTeam: string
  isHome: boolean
  homeGoals: number | null
  guestGoals: number | null
  /** Ohne ausgewerteten Bericht bleibt die Spalte leer — der Endstand steht
   * trotzdem im Kopf, damit die fehlende Auswertung sichtbar ist. */
  hasReport: boolean
}

/** Eine Zeile der Spielmatrix. `cells` läuft parallel zu `TeamMatrix.games`;
 * `null` heißt "stand in der Mannschaftsliste nicht" und ist etwas anderes als
 * eine Zelle mit lauter Nullen. */
export interface MatrixPlayer {
  playerId: number
  memberId: number | null
  name: string
  cells: (MatrixCell | null)[]
  total: MatrixCell
  games: number
}

/** Die Spielmatrix einer Mannschaft. */
export interface TeamMatrix {
  team: string
  /** Gepflegte Halbzeitdauer der Altersklasse; `null` heißt "keine Regel" —
   * die Achse des Torverlaufs wird dann aus dem Verlauf abgeleitet. */
  halfDurationMinutes: number | null
  games: MatrixGame[]
  players: MatrixPlayer[]
  gameTotals: MatrixCell[]
  total: MatrixCell
  reportGames: number
}

export interface PlayerLine {
  playerId: number
  memberId: number | null
  name: string
  side: 'home' | 'guest'
  number: number | null
  goals: number
  sevenMAttempts: number
  sevenMGoals: number
  twoMin: number
  warnings: number
  disq: number
  conflict: string
}

export type EventKind =
  | 'goal' | 'seven_m_goal' | 'seven_m_miss' | 'two_min'
  | 'warning' | 'disqualification' | 'timeout' | 'other'

export interface EventLine {
  seq: number
  clockTime: string
  gameSecond: number
  scoreHome: number | null
  scoreGuest: number | null
  kind: EventKind
  side: 'home' | 'guest' | ''
  playerId: number | null
  playerName: string | null
  number: number | null
  rawText: string
}

export interface ReportDetail {
  reportId: number
  state: string
  homeTeam: string
  guestTeam: string
  homeGoals: number | null
  guestGoals: number | null
  homeGoalsHt: number | null
  guestGoalsHt: number | null
  spectators: string
  /** Die ungetrennte Rohzeile des Dokuments — der Beleg. */
  referees: string
  /** Die daraus gewonnenen Personen; leer beim Bestand vor Migration 069. */
  refereeNames: string[]
  refereesUncertain: boolean
  warnings: string[]
  hasPdf: boolean
  players: PlayerLine[]
  events: EventLine[]
}

export const fetchStaffeln = () => api.get<Staffel[]>('/staffeln').then((r) => r.data)

/** Löst alle Staffelcodes der Saison auf und ruft die Spielpläne ab. */
export const syncStaffeln = () => api.post('/staffeln/sync').then((r) => r.data)
export const fetchTable = (id: number) => api.get<TableRow[]>(`/staffeln/${id}/tabelle`).then((r) => r.data)
export const fetchSchedule = (id: number) => api.get<ScheduleGame[]>(`/staffeln/${id}/spielplan`).then((r) => r.data)
export const fetchRanglisten = (id: number) => api.get<PlayerStat[]>(`/staffeln/${id}/ranglisten`).then((r) => r.data)
export const fetchCrossTable = (id: number) =>
  api.get<CrossTable>(`/staffeln/${id}/kreuztabelle`).then((r) => r.data)
export const fetchProgression = (id: number) =>
  api.get<ProgressionDay[]>(`/staffeln/${id}/tabellenverlauf`).then((r) => r.data)
export const fetchTeamStats = (id: number) =>
  api.get<TeamStats>(`/staffeln/${id}/teamstatistik`).then((r) => r.data)
export const fetchRefereeStats = (id: number) =>
  api.get<RefereeStat[]>(`/staffeln/${id}/schiedsrichter`).then((r) => r.data)
export const fetchAffiliation = (id: number) =>
  api.get<Affiliation>(`/staffeln/${id}/affiliation`).then((r) => r.data)
export const fetchPlayerGames = (id: number) =>
  api.get<{ teams: TeamMatrix[] }>(`/staffeln/${id}/player-games`).then((r) => r.data.teams)
export const fetchReport = (bwhvGameId: number) =>
  api.get<ReportDetail>(`/bwhv-games/${bwhvGameId}/report`).then((r) => r.data)

// Derselbe Bericht, adressiert über den eigenen Spieltermin: die
// Spieldetail-Ansicht kennt eine games.id, nicht die BWHV-Begegnung.
export const fetchReportForGame = (gameId: number) =>
  api.get<ReportDetail>(`/games/${gameId}/bwhv-report`).then((r) => r.data)
export const fetchMemberStats = (memberId: number) =>
  api.get<PlayerStat[]>(`/members/${memberId}/saisonstatistik`).then((r) => r.data)

// Torverhältnis als "112:98" — die Schnittstelle liefert beide Zahlen getrennt.
export const goalRatio = (r: TableRow) => `${r.GoalsFor}:${r.GoalsAgainst}`

// Punkte in der im Handball üblichen Form "6:2".
export const pointsLabel = (r: TableRow) => `${r.PointsPlus}:${r.PointsMinus}`

// Siebenmeter-Quote in Prozent; ohne Versuch gibt es keine Quote (null statt 0,
// sonst läse sich "nie geworfen" wie "immer verworfen").
export function sevenMeterRate(s: PlayerStat): number | null {
  if (s.sevenMAttempts === 0) return null
  return Math.round((s.sevenMGoals / s.sevenMAttempts) * 100)
}

// Spielzeit "12:27" aus Sekunden.
export function gameClock(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

// --- Freitextsuche --------------------------------------------------------

// Kleinschreibung und ohne Akzente/Umlautpunkte: "Mössing" soll auch über
// "mossing" gefunden werden, und wer auf dem Handy tippt, setzt selten Umlaute.
const normalize = (v: string) =>
  v.toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '')

/**
 * Freitext: jedes durch Leerraum getrennte Wort der Eingabe muss irgendwo in
 * den Feldern vorkommen (UND-Verknüpfung, Reihenfolge egal). Eine leere
 * Eingabe trifft alles.
 */
export function matchesSearch(fields: (string | null | undefined)[], query: string): boolean {
  const words = normalize(query).split(/\s+/).filter(Boolean)
  if (words.length === 0) return true
  const haystack = normalize(fields.filter(Boolean).join(' '))
  return words.every((w) => haystack.includes(w))
}

/** Durchsuchbare Felder einer Begegnung: Mannschaften, Halle, Datum (ISO und deutsch). */
export function gameSearchFields(g: ScheduleGame): string[] {
  const day = g.Date.slice(0, 10)
  const [y, m, d] = day.split('-')
  return [
    g.HomeTeam, g.GuestTeam, g.HallNumber,
    g.Venue?.name, g.Venue?.street, g.Venue?.city,
    day, `${d}.${m}.${y}`, g.GameNo,
  ].filter((f): f is string => !!f)
}
