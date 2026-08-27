import { describe, test, it, expect } from 'vitest'
import { formatTimeSpan, hoursToDisplay } from './duration'

describe('formatTimeSpan', () => {
  test('Normalfall: Startzeit + volle Stunde ergibt eine Spanne mit Halbgeviertstrich', () => {
    expect(formatTimeSpan('08:00', 1)).toBe('8:00–9:00')
  })

  test('Mitternachtsüberlauf: Endzeit springt auf den Folgetag ohne Datumszusatz', () => {
    expect(formatTimeSpan('23:30', 1)).toBe('23:30–00:30')
  })

  test('fehlende Startzeit: bisheriger Platzhalter statt einer konstruierten Spanne', () => {
    expect(formatTimeSpan(null, 1)).toBe('—')
    expect(formatTimeSpan('', 1)).toBe('—')
    expect(formatTimeSpan(undefined, 1)).toBe('—')
  })

  test('krumme Dauer: 0.3333… Stunden runden sinnvoll auf 20 Minuten', () => {
    expect(formatTimeSpan('10:00', 1 / 3)).toBe('10:00–10:20')
  })
})

describe('formatTimeSpan — fehlende Dauer', () => {
  // Der Service Worker liefert /api/* network-first: unmittelbar nach dem Deploy
  // kann eine gecachte Board-Antwort noch ohne hours_value ankommen. Dann ist der
  // reine Startzeitpunkt die richtige Ausgabe — nicht "8:00–NaN:NaN".
  it('zeigt nur die Startzeit, wenn die Dauer fehlt', () => {
    expect(formatTimeSpan('08:00', undefined as unknown as number)).toBe('8:00')
    expect(formatTimeSpan('08:00', NaN)).toBe('8:00')
  })

  it('zeigt nur die Startzeit bei nicht-positiver Dauer', () => {
    expect(formatTimeSpan('08:00', 0)).toBe('8:00')
    expect(formatTimeSpan('08:00', -1)).toBe('8:00')
  })

  it('bleibt beim Platzhalter, wenn schon die Startzeit fehlt', () => {
    expect(formatTimeSpan(null, undefined as unknown as number)).toBe('—')
  })
})

describe('hoursToDisplay — fehlender Wert', () => {
  it('liefert einen leeren String statt NaN', () => {
    expect(hoursToDisplay(undefined as unknown as number)).toBe('')
    expect(hoursToDisplay(NaN)).toBe('')
  })
})
