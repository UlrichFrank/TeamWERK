import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

/**
 * Gate für die Kachelform „Eckfahne" (Change `kachel-eckfahne`): Kacheln und
 * Modals tragen eine 2-px-Oberkante, 2 px Eckenradius und das Dreieck oben links
 * (`eckfahne`, `web/src/index.css`). Die Klassen-Strings sind wie die Card-Strings
 * davor abgetippt, nicht importiert — ohne Gate kommt mit der nächsten neuen
 * Karte die alte Form (`rounded-xl … border-t-4`) zurück, oder eine 2-px-Kante
 * ohne Fahne. Textuell, eine Bremse, kein Beweis (wie `designTokens.gate`).
 */

const SRC = resolve(process.cwd(), 'src')
const SCAN_DIRS = ['pages', 'components']

function tsxFiles(dir: string): string[] {
  const out: string[] = []
  const walk = (d: string) => {
    for (const entry of readdirSync(d)) {
      const p = join(d, entry)
      if (statSync(p).isDirectory()) walk(p)
      else if (entry.endsWith('.tsx') && !entry.includes('.test.')) out.push(p)
    }
  }
  walk(join(SRC, dir))
  return out
}

function findViolations(entries: { rel: string; src: string }[]): string[] {
  const out: string[] = []
  for (const { rel, src } of entries) {
    src.split('\n').forEach((line, i) => {
      if (/(?<![\w-])border-t-4\b/.test(line)) {
        out.push(`${rel}:${i + 1}: alte Kachel-Oberkante border-t-4 — border-t-2 eckfahne verwenden`)
      }
      if (/(?<![\w-])border-t-2\b/.test(line) && /\brounded-x[sl]\b/.test(line) && !/\beckfahne\b/.test(line)) {
        out.push(`${rel}:${i + 1}: Kachel-Oberkante ohne eckfahne`)
      }
      if (/(?:^|[\s"'`])sm:rounded-(?:l|tl|bl)-3xl\b/.test(line)) {
        out.push(`${rel}:${i + 1}: alte Abgrenzung des Inhaltsbereichs — sm:border-l-[3px] sm:eckfahne sm:eckfahne-lg verwenden`)
      }
      // `eckfahne` setzt position: relative für sein ::before — eine anders
      // positionierte Kachel verlöre ihre Positionierung oder ihre Fahne.
      if (/(?<![\w-])eckfahne(?![\w-])/.test(line) && /(?:^|[\s"'`])(?:sticky|fixed|absolute)(?=[\s"'`]|$)/.test(line)) {
        out.push(`${rel}:${i + 1}: eckfahne auf sticky/fixed/absolute-Element — Fahne braucht position: relative`)
      }
      if (/\beckfahne\b/.test(line) && /\brounded-(?:xl|2xl|lg)\b/.test(line)) {
        out.push(`${rel}:${i + 1}: Eckfahne mit großem Radius — rounded-xs verwenden`)
      }
    })
  }
  return out
}

describe('Eckfahne-Gate', () => {
  it('alle Kacheln und Modals in pages/ und components/ tragen die Eckfahne', () => {
    const entries = SCAN_DIRS.flatMap(tsxFiles).map(p => ({
      rel: relative(SRC, p),
      src: readFileSync(p, 'utf8'),
    }))
    expect(entries.length).toBeGreaterThan(50)
    expect(findViolations(entries)).toEqual([])
  })

  it('Poison: alte Kachelform wird erkannt', () => {
    const src = '<div className="bg-brand-surface-card rounded-xl shadow-sm border-t-4 border-brand-yellow p-6" />'
    expect(findViolations([{ rel: 'x.tsx', src }])).toHaveLength(1)
  })

  it('Poison: Oberkante ohne Fahne wird erkannt', () => {
    const src = '<div className="bg-brand-surface-card rounded-xs shadow-sm border-t-2 border-brand-yellow p-6" />'
    expect(findViolations([{ rel: 'x.tsx', src }])).toHaveLength(1)
  })

  it('Poison: Fahne mit großem Radius wird erkannt', () => {
    const src = '<div className="rounded-xl border-t-2 eckfahne border-brand-yellow" />'
    expect(findViolations([{ rel: 'x.tsx', src }])).toHaveLength(1)
  })

  it('Poison: alte Abgrenzung neben der Seitenleiste wird erkannt', () => {
    const src = '<div className="flex-1 bg-brand-white sm:rounded-l-3xl sm:border-l-4 sm:border-brand-yellow" />'
    expect(findViolations([{ rel: 'x.tsx', src }])).toHaveLength(1)
  })

  it('Poison: Eckfahne auf positioniertem Element wird erkannt', () => {
    const src = '<div className="sticky top-0 rounded-xs border-t-2 eckfahne border-brand-yellow" />'
    expect(findViolations([{ rel: 'x.tsx', src }])).toHaveLength(1)
  })

  it('korrekte Kachel ist kein Befund', () => {
    const src = '<div className="bg-brand-surface-card rounded-xs shadow-sm border-t-2 eckfahne border-brand-yellow transform-gpu p-6" />'
    expect(findViolations([{ rel: 'x.tsx', src }])).toEqual([])
  })
})
