import { api } from './api'

// Die Typen spiegeln die Antworten von internal/gamestats. Feldnamen sind dort
// per JSON-Tag festgelegt und folgen camelCase.

export interface Staffel {
  id: number
  code: string
  name: string
  teamName: string
  kaderId: number
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
  HasReport: boolean
}

export interface PlayerStat {
  playerId: number
  memberId: number | null
  name: string
  teamName: string
  games: number
  goals: number
  sevenMAttempts: number
  sevenMGoals: number
  twoMin: number
  warnings: number
  disq: number
  fairPlayScore: number
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
  referees: string
  warnings: string[]
  hasPdf: boolean
  players: PlayerLine[]
  events: EventLine[]
}

export const fetchStaffeln = () => api.get<Staffel[]>('/staffeln').then((r) => r.data)
export const fetchTable = (id: number) => api.get<TableRow[]>(`/staffeln/${id}/tabelle`).then((r) => r.data)
export const fetchSchedule = (id: number) => api.get<ScheduleGame[]>(`/staffeln/${id}/spielplan`).then((r) => r.data)
export const fetchRanglisten = (id: number) => api.get<PlayerStat[]>(`/staffeln/${id}/ranglisten`).then((r) => r.data)
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
