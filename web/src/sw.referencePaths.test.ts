import { describe, test, expect, beforeAll } from 'vitest'

// Regressions-Guard (kann den SW nicht in jsdom ausführen): stellt sicher, dass
// nutzergefilterte Routen NICHT in die geteilte StaleWhileRevalidate-Menge des
// Service Workers geraten. `/api/teams` ist pro Nutzer gefiltert
// (Games.ListTeamsForUser) — ein geteilter geräteweiter SW-Cache würde sonst
// nach Login-Wechsel die Teams des Vor-Nutzers ausliefern (Cross-User-Leak).
// Quelltext Vite-nativ (?raw) eingelesen — kein node:fs.
let swSource = ''
beforeAll(async () => {
  swSource = (await import('./sw.ts?raw')).default
})

// Extrahiert das REFERENCE_PATHS-Set-Literal aus dem Quelltext.
function referencePathsLiteral(): string {
  const match = swSource.match(/const REFERENCE_PATHS = new Set\(\[([\s\S]*?)\]\)/)
  expect(match, 'REFERENCE_PATHS-Set nicht im sw.ts gefunden').toBeTruthy()
  return match![1]
}

describe('Service-Worker-Referenzrouten (StaleWhileRevalidate)', () => {
  test('nutzergefiltertes /api/teams ist NICHT im geteilten SW-Cache', () => {
    expect(referencePathsLiteral()).not.toContain('/api/teams')
  })

  test('club-weite Referenzrouten sind im geteilten SW-Cache', () => {
    const literal = referencePathsLiteral()
    for (const path of ['/api/seasons', '/api/venues', '/api/age-class-rules', '/api/duty-types']) {
      expect(literal).toContain(path)
    }
  })
})
