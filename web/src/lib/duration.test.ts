import { describe, test, it, expect } from 'vitest'
import { formatTimeSpan, hoursToDisplay, dynamicSpanImpossible } from './duration'

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

// openspec/changes/dienst-zeitmodus-strikt: die Regel entscheidet, was gar nicht erst
// gespeichert werden darf. Sie ist bewusst NOTWENDIG, nicht hinreichend — bei
// verschiedenen Ankern hängt die Dauer an der Spieldauer des Termins und ist hier nicht
// entscheidbar. Genau diese Grenze halten die Fälle unten fest.
describe('dynamicSpanImpossible', () => {
  it('erkennt den gleichen Anker mit nicht dahinter liegendem End-Versatz', () => {
    expect(dynamicSpanImpossible('dynamisch', 'start', 40, 'start', 25)).toBe(true)
    expect(dynamicSpanImpossible('dynamisch', 'start', 40, 'start', 40)).toBe(true)
    expect(dynamicSpanImpossible('dynamisch', 'end', 0, 'end', -30)).toBe(true)
  })

  it('lässt den gleichen Anker mit dahinter liegendem End-Versatz zu', () => {
    expect(dynamicSpanImpossible('dynamisch', 'start', 25, 'start', 40)).toBe(false)
    expect(dynamicSpanImpossible('dynamisch', 'start', -30, 'start', 20)).toBe(false)
  })

  it('prüft verschiedene Anker nicht — die Spieldauer entscheidet dort', () => {
    // "Start bei Anpfiff, Ende 15 min vor Spielende": bei jedem Spiel > 15 min gültig.
    expect(dynamicSpanImpossible('dynamisch', 'start', 0, 'end', -15)).toBe(false)
    // Auch die knappe Gegenrichtung bleibt erlaubt; sie scheitert ggf. erst am Termin.
    expect(dynamicSpanImpossible('dynamisch', 'end', 0, 'start', 30)).toBe(false)
  })

  it('greift im Modus absolut nie — die End-Felder sind dort bedeutungslos', () => {
    expect(dynamicSpanImpossible('absolut', 'start', 40, 'start', 25)).toBe(false)
  })
})
