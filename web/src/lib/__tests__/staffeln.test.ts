import { describe, test, expect } from 'vitest'
import {
  goalRatio, pointsLabel, sevenMeterRate, gameClock,
  type TableRow, type PlayerStat,
} from '../staffeln'

const row = (over: Partial<TableRow> = {}): TableRow => ({
  Position: 1, TeamName: 'A', Games: 3, Won: 2, Drawn: 0, Lost: 1,
  GoalsFor: 112, GoalsAgainst: 98, PointsPlus: 4, PointsMinus: 2, ...over,
})

const stat = (over: Partial<PlayerStat> = {}): PlayerStat => ({
  playerId: 1, memberId: null, name: 'P', teamName: 'A', games: 1, goals: 0,
  sevenMAttempts: 0, sevenMGoals: 0, twoMin: 0, warnings: 0, disq: 0,
  fairPlayScore: 0, ...over,
})

describe('Tabellen-Formatierung', () => {
  test('Torverhältnis als Doppelpunkt-Form', () => {
    expect(goalRatio(row())).toBe('112:98')
  })
  test('Punkte in der Handball-üblichen Form', () => {
    expect(pointsLabel(row())).toBe('4:2')
  })
})

describe('Siebenmeter-Quote', () => {
  // Ohne Versuch gibt es keine Quote: 0 % läse sich wie "immer verworfen".
  test('ohne Versuch keine Quote', () => {
    expect(sevenMeterRate(stat())).toBeNull()
  })
  test('gerundete Prozent', () => {
    expect(sevenMeterRate(stat({ sevenMAttempts: 3, sevenMGoals: 2 }))).toBe(67)
  })
  test('alle getroffen', () => {
    expect(sevenMeterRate(stat({ sevenMAttempts: 2, sevenMGoals: 2 }))).toBe(100)
  })
})

describe('Spielzeit', () => {
  test('Sekunden zu mm:ss mit führender Null', () => {
    expect(gameClock(99)).toBe('01:39')
    expect(gameClock(747)).toBe('12:27')
    expect(gameClock(0)).toBe('00:00')
  })
  test('über eine Stunde Spielzeit bleibt minutenbasiert', () => {
    expect(gameClock(3661)).toBe('61:01')
  })
})
