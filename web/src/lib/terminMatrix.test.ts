import { describe, expect, test } from 'vitest'
import { cutoffLocked, formatColumnDate, formatParticipation, isCellRespondable, matrixLoadWindow, nextConfirmStatus, participation, visibleColumns, type MatrixEvent, type MatrixMember } from './terminMatrix'

const ev = (id: number, event_type: MatrixEvent['event_type'], cancelled = false, date = '2026-09-13'): MatrixEvent => ({
  kind: event_type === 'training' ? 'training' : 'game',
  id,
  date,
  time: '18:00',
  event_type,
  title: '',
  cancelled,
  rsvp_require_reason: false,
})

// „Heute" liegt nach allen Standard-Terminen: sie zählen als vergangen.
const LATER = '2026-12-31'

const EVENTS = [ev(1, 'training'), ev(2, 'heim'), ev(3, 'training', true), ev(4, 'auswärts'), ev(5, 'training')]

const member = (cells: MatrixMember['cells'], self = false): MatrixMember => ({ member_id: 1, name: 'A', extended: false, is_self: self, can_respond: self, cells })

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
    expect(participation(m, EVENTS, [0, 1, 2, 3, 4], LATER).past).toEqual({ count: 2, total: 3 })
  })

  test('folgt dem Typ-Filter', () => {
    expect(participation(m, EVENTS, visibleColumns(EVENTS, new Set(['training'])), LATER).past).toEqual({ count: 1, total: 1 })
  })

  test('erfasste Anwesenheit geht vor der Zusage', () => {
    const t = member([
      { status: 'confirmed', is_default: false, present: false },
      { status: 'declined', is_default: false, present: true },
    ])
    expect(participation(t, EVENTS, [0, 1], LATER).past).toEqual({ count: 1, total: 2 })
  })
})

describe('participation nach Vergangenheit und Zukunft', () => {
  const events = [ev(1, 'training', false, '2026-09-20'), ev(2, 'training', false, '2026-09-22'), ev(3, 'training', false, '2026-09-23'), ev(4, 'heim', false, '2026-09-30')]
  const m = member([
    { status: 'confirmed', is_default: false, present: true },
    { status: 'confirmed', is_default: false, present: false },
    { status: 'confirmed', is_default: false },
    { status: 'declined', is_default: false },
  ])
  test('heute zählt zu geplant, Anwesenheit nur für bisher', () => {
    expect(participation(m, events, [0, 1, 2, 3], '2026-09-23')).toEqual({
      past: { count: 1, total: 2 },
      future: { count: 1, total: 2 },
    })
  })
})

describe('Zu-/Absage-Regeln', () => {
  test('„Zusagen" schaltet in der eigenen Zeile wie in der Liste', () => {
    expect(nextConfirmStatus({ status: 'confirmed', is_default: false }, true)).toBe('maybe')
    expect(nextConfirmStatus({ status: 'confirmed', is_default: true }, true)).toBe('confirmed')
    expect(nextConfirmStatus({ status: null, is_default: false }, true)).toBe('confirmed')
    expect(nextConfirmStatus({ status: 'confirmed', is_default: false }, false)).toBe('confirmed')
  })
  test('Frist sperrt nur ohne Override', () => {
    const e = { ...ev(1, 'training'), rsvp_locks_at: '2026-09-13T14:00:00Z' }
    const after = new Date('2026-09-13T15:00:00Z').getTime()
    expect(cutoffLocked(e, false, after)).toBe(true)
    expect(cutoffLocked(e, true, after)).toBe(false)
    expect(cutoffLocked(e, false, new Date('2026-09-13T13:00:00Z').getTime())).toBe(false)
  })
  test('nur eigene/Kind-Zeilen, nicht abgesagt, nicht abgemeldet', () => {
    const cell = { status: null, is_default: false }
    expect(isCellRespondable(member([cell], true), ev(1, 'training'), cell)).toBe(true)
    expect(isCellRespondable(member([cell], false), ev(1, 'training'), cell)).toBe(false)
    expect(isCellRespondable(member([cell], true), ev(1, 'training', true), cell)).toBe(false)
    expect(isCellRespondable(member([cell], true), ev(1, 'training'), { ...cell, unavailable: true })).toBe(false)
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
