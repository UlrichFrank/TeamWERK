import { describe, expect, test } from 'vitest'
import { formatColumnDate, formatParticipation, matrixLoadWindow, participation, visibleColumns, type MatrixEvent, type MatrixMember } from './terminMatrix'

const ev = (id: number, event_type: MatrixEvent['event_type'], cancelled = false): MatrixEvent => ({
  kind: event_type === 'training' ? 'training' : 'game',
  id,
  date: '2026-09-13',
  time: '18:00',
  event_type,
  title: '',
  cancelled,
})

const EVENTS = [ev(1, 'training'), ev(2, 'heim'), ev(3, 'training', true), ev(4, 'auswärts'), ev(5, 'training')]

const member = (cells: MatrixMember['cells']): MatrixMember => ({ member_id: 1, name: 'A', extended: false, cells })

describe('visibleColumns', () => {
  test('filtert nach Termin-Typ', () => {
    expect(visibleColumns(EVENTS, new Set(['training']))).toEqual([0, 2, 4])
    expect(visibleColumns(EVENTS, new Set(['heim', 'auswärts']))).toEqual([1, 3])
  })
})

describe('participation', () => {
  const m = member([
    { status: 'confirmed', is_default: false },
    { status: 'confirmed', is_default: true },
    { status: 'confirmed', is_default: false }, // abgesagter Termin
    { status: 'declined', is_default: false },
    { status: null, is_default: false, unavailable: true },
  ])

  test('zählt Zusagen inkl. Voreinstellung, ohne abgesagte und abgemeldete Termine', () => {
    expect(participation(m, EVENTS, [0, 1, 2, 3, 4])).toEqual({ count: 2, total: 3 })
  })

  test('folgt dem Typ-Filter', () => {
    expect(participation(m, EVENTS, visibleColumns(EVENTS, new Set(['training'])))).toEqual({ count: 1, total: 1 })
  })

  test('erfasste Anwesenheit geht vor der Zusage', () => {
    const t = member([
      { status: 'confirmed', is_default: false, present: false },
      { status: 'declined', is_default: false, present: true },
    ])
    expect(participation(t, EVENTS, [0, 1])).toEqual({ count: 1, total: 2 })
  })
})

describe('Formatierung', () => {
  test('Quote', () => {
    expect(formatParticipation({ count: 1, total: 2 })).toBe('1 (50 %)')
    expect(formatParticipation({ count: 0, total: 0 })).toBe('–')
  })
  test('Spaltendatum', () => {
    expect(formatColumnDate('2026-09-07T00:00:00Z')).toBe('07.09.')
  })
})

describe('matrixLoadWindow', () => {
  const season = { start_date: '2026-08-01', end_date: '2027-06-30' }
  test('ohne Vergangene ab heute bis Saisonende', () => {
    expect(matrixLoadWindow(season, false, '2026-09-23')).toEqual({ from: '2026-09-23', to: '2027-06-30' })
  })
  test('mit Vergangene ab Saisonstart', () => {
    expect(matrixLoadWindow(season, true, '2026-09-23')).toEqual({ from: '2026-08-01', to: '2027-06-30' })
  })
  test('Saison schon vorbei: 90 Tage ab heute', () => {
    expect(matrixLoadWindow({ start_date: '2025-08-01', end_date: '2026-06-30' }, false, '2026-09-23'))
      .toEqual({ from: '2026-09-23', to: '2026-12-22' })
  })
  test('deckelt auf 400 Tage', () => {
    expect(matrixLoadWindow({ start_date: '2025-01-01', end_date: '2027-06-30' }, true, '2026-09-23'))
      .toEqual({ from: '2025-01-01', to: '2026-02-05' })
  })
})
