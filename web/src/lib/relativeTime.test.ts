import { describe, it, expect, vi, afterEach } from 'vitest'
import { relativeTime } from './relativeTime'

const NOW = new Date('2026-08-16T12:00:00Z')

describe('relativeTime', () => {
  afterEach(() => vi.useRealTimers())

  function at(iso: string) {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
    return relativeTime(iso)
  }

  it('below 2 minutes reads "gerade eben"', () => {
    expect(at('2026-08-16T11:59:00Z')).toBe('gerade eben')
  })

  it('formats minutes', () => {
    expect(at('2026-08-16T11:45:00Z')).toBe('vor 15 Min.')
  })

  it('formats hours', () => {
    expect(at('2026-08-16T09:00:00Z')).toBe('vor 3 Std.')
  })

  it('formats exactly one day as "gestern"', () => {
    expect(at('2026-08-15T12:00:00Z')).toBe('gestern')
  })

  it('formats days below 30', () => {
    expect(at('2026-08-10T12:00:00Z')).toBe('vor 6 Tagen')
  })

  it('formats one month', () => {
    expect(at('2026-07-15T12:00:00Z')).toBe('vor 1 Monat')
  })

  it('formats multiple months', () => {
    expect(at('2026-05-01T12:00:00Z')).toBe('vor 3 Monaten')
  })
})

describe('Zeitzonen-Behandlung von Server-Zeitstempeln', () => {
  // SQLite CURRENT_TIMESTAMP: UTC, aber ohne Zonenkennung. Wird der String als
  // Ortszeit gelesen, verschiebt sich alles um den lokalen Offset — im Sommer
  // zwei Stunden. Das ist der Unterschied zwischen "gerade eben" und
  // "vor 2 Std." für denselben Eintrag.
  const sqliteFormat = (d: Date) => d.toISOString().replace('T', ' ').slice(0, 19)

  it('liest einen SQLite-Zeitstempel ohne Zonenkennung als UTC', () => {
    const jetzt = new Date('2026-08-16T15:44:12Z')
    vi.setSystemTime(jetzt)
    expect(relativeTime(sqliteFormat(jetzt))).toBe('gerade eben')
    vi.useRealTimers()
  })

  it('verschiebt sich nicht um den lokalen Offset', () => {
    const jetzt = new Date('2026-08-16T15:44:12Z')
    vi.setSystemTime(jetzt)
    const vorDreiStunden = new Date(jetzt.getTime() - 3 * 3600_000)
    expect(relativeTime(sqliteFormat(vorDreiStunden))).toBe('vor 3 Std.')
    vi.useRealTimers()
  })

  it('lässt RFC3339-Strings mit Zonenkennung unangetastet', () => {
    const jetzt = new Date('2026-08-16T15:44:12Z')
    vi.setSystemTime(jetzt)
    expect(relativeTime('2026-08-16T13:44:12Z')).toBe('vor 2 Std.')
    expect(relativeTime('2026-08-16T15:44:12+02:00')).toBe('vor 2 Std.')
    vi.useRealTimers()
  })
})
