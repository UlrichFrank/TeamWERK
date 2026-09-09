import { describe, test, expect } from 'vitest'
import {
  buildTeamOptions,
  effectiveTeamIds,
  matchesTeamFilter,
  parseTeamIds,
  serializeTeamIds,
  toggleTeamId,
} from './teamFilter'

// Teil von openspec/changes/team-mehrfachfilter-alle-listen: die vier Listen
// (Termine, Kalender, Dienste, Mitfahrten) teilen sich diese Semantik. Die
// Kernregel steht in den letzten beiden Blöcken: leere Menge = vollständige
// Menge = „kein Filter".

const OPTIONS = [
  { id: 1, label: 'mA2' },
  { id: 2, label: 'mC2' },
  { id: 3, label: 'wB' },
]

describe('parseTeamIds', () => {
  test('leerer/fehlender Wert ergibt kein Filter', () => {
    expect(parseTeamIds(null).size).toBe(0)
    expect(parseTeamIds('').size).toBe(0)
  })

  test('einzelne ID bleibt gültig (Bestandslinks)', () => {
    expect([...parseTeamIds('3')]).toEqual([3])
  })

  test('kommaseparierte Liste, Leerzeichen tolerant', () => {
    expect([...parseTeamIds('3, 7')]).toEqual([3, 7])
  })

  test('unbrauchbare Teile werden still verworfen', () => {
    expect([...parseTeamIds('abc,4,-1,0')]).toEqual([4])
    expect(parseTeamIds('abc').size).toBe(0)
  })
})

describe('effectiveTeamIds', () => {
  test('kein Filter zeigt alle Kästchen angehakt', () => {
    expect([...effectiveTeamIds(new Set(), OPTIONS)]).toEqual([1, 2, 3])
  })

  test('echte Auswahl bleibt unverändert', () => {
    expect([...effectiveTeamIds(new Set([2]), OPTIONS)]).toEqual([2])
  })
})

describe('toggleTeamId', () => {
  test('aus „alle" wird die Menge ohne die abgewählte Mannschaft', () => {
    expect([...toggleTeamId(effectiveTeamIds(new Set(), OPTIONS), 2)]).toEqual([1, 3])
  })

  test('fügt eine fehlende Mannschaft hinzu', () => {
    expect([...toggleTeamId(new Set([1]), 3)]).toEqual([1, 3])
  })
})

describe('serializeTeamIds', () => {
  test('leere Auswahl ist kein Filter', () => {
    expect(serializeTeamIds(new Set(), 3)).toBeNull()
  })

  test('vollständige Auswahl ist ebenfalls kein Filter', () => {
    expect(serializeTeamIds(new Set([1, 2, 3]), 3)).toBeNull()
  })

  test('echte Teilmenge wird zur ID-Liste', () => {
    expect(serializeTeamIds(new Set([1, 3]), 3)).toBe('1,3')
  })

  test('ohne geladene Mannschaften zählt nur die leere Menge als Default', () => {
    expect(serializeTeamIds(new Set([1]), 0)).toBe('1')
  })
})

describe('matchesTeamFilter', () => {
  test('ohne Filter passt alles — auch ein Termin ohne Mannschaft', () => {
    expect(matchesTeamFilter(new Set(), [])).toBe(true)
    expect(matchesTeamFilter(new Set(), undefined)).toBe(true)
  })

  test('eine Überschneidung genügt', () => {
    expect(matchesTeamFilter(new Set([2]), [1, 2])).toBe(true)
  })

  test('ohne Überschneidung fällt der Termin heraus', () => {
    expect(matchesTeamFilter(new Set([2]), [1, 3])).toBe(false)
  })

  test('bei aktivem Filter fällt ein Termin ohne Mannschaft heraus', () => {
    expect(matchesTeamFilter(new Set([2]), [])).toBe(false)
    expect(matchesTeamFilter(new Set([2]), undefined)).toBe(false)
  })
})

describe('buildTeamOptions', () => {
  test('nutzt den Kurznamen, sonst den vollen Namen', () => {
    const options = buildTeamOptions([
      { id: 1, name: 'A-Jugend männlich 2', age_class: 'A-Jugend', gender: 'm', team_number: 2, group_count: 2 },
      { id: 3, name: 'B-Jugend weiblich', age_class: 'B-Jugend', gender: 'f', team_number: 1, group_count: 1 },
    ])
    expect(options).toEqual([
      { id: 1, label: 'mA2' },
      { id: 3, label: 'wB' },
    ])
  })
})
