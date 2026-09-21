import { describe, test, expect } from 'vitest'
import { sharedRanks } from '../ranking'

describe('sharedRanks', () => {
  test('Gleichstand teilt den Platz und überspringt den nächsten', () => {
    expect(sharedRanks([30, 30, 25, 20, 20, 20, 10], (n) => n)).toEqual([1, 1, 3, 4, 4, 4, 7])
  })
  test('ohne Gleichstand zählt es durch', () => {
    expect(sharedRanks([3, 2, 1], (n) => n)).toEqual([1, 2, 3])
  })
  test('leere Liste', () => {
    expect(sharedRanks([], (n: number) => n)).toEqual([])
  })
  test('der Schlüssel darf die angezeigte, gerundete Größe sein', () => {
    const rows = [28.52, 28.48, 27.9]
    expect(sharedRanks(rows, (v) => v.toFixed(1))).toEqual([1, 1, 3])
  })
})
