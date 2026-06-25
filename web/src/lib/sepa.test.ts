import { describe, it, expect } from 'vitest'
import { isValidIBAN, normalizeIBAN } from './sepa'

// Vektoren gespiegelt aus internal/sepa/iban_test.go — Client- und Server-IBAN-Prüfung
// müssen identisch urteilen (sonst weicht der clientseitige Fee-Run vom Server ab).
describe('isValidIBAN (Parität zu internal/sepa/iban.go)', () => {
  const cases: [string, string, boolean][] = [
    ['DE gültig', 'DE89370400440532013000', true],
    ['DE gültig mit Leerzeichen', 'DE89 3704 0044 0532 0130 00', true],
    ['DE gültig kleingeschrieben', 'de89370400440532013000', true],
    ['AT gültig', 'AT611904300234573201', true],
    ['CH gültig', 'CH9300762011623852957', true],
    ['DE falsche Prüfsumme', 'DE88370400440532013000', false],
    ['DE zu kurz', 'DE8937040044', false],
    ['Müll', 'NICHTSGUELTIGES', false],
    ['leer', '', false],
    ['Ziffern statt Ländercode', '1289370400440532013000', false],
  ]
  for (const [name, input, want] of cases) {
    it(name, () => {
      expect(isValidIBAN(input)).toBe(want)
    })
  }
})

describe('normalizeIBAN', () => {
  it('trimmt, entfernt Leerzeichen, Großbuchstaben', () => {
    expect(normalizeIBAN('  de89 3704 0044 ')).toBe('DE8937040044')
  })
})
