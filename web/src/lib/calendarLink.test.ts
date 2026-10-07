import { describe, test, expect } from 'vitest'
import { calendarLink } from './calendarLink'

describe('calendarLink', () => {
  test('kürzt ISO-Timestamps auf das Datum', () => {
    expect(calendarLink('game', 17, '2026-10-11T00:00:00Z')).toBe('/kalender?date=2026-10-11&focus=game-17')
  })

  test('baut Trainings-Links', () => {
    expect(calendarLink('training', 42, '2026-10-14')).toBe('/kalender?date=2026-10-14&focus=training-42')
  })
})
