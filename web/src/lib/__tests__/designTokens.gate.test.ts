import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

/**
 * Gate für die beiden Hard Rules aus CLAUDE.md, die bisher nur dokumentiert
 * waren: „Nur `brand-*`-Tokens" und „Keine Unicode-Icons in JSX".
 *
 * Ohne Gate waren über 140 Raw-Farben (`text-gray-700`, `bg-green-100`, …) und
 * mehrere Glyphen-Buttons (`×`, `✓ Annehmen`) im Bestand — jede für sich klein,
 * zusammen der Grund, warum Profil- und Mitglieder-Formulare anders aussahen als
 * der Rest. Dasselbe Prinzip wie `buttonStyles.gate.test.ts`: textuell, eine
 * Bremse, kein Beweis.
 */

const SRC = resolve(process.cwd(), 'src')
const SCAN_DIRS = ['pages', 'components']

const PALETTE =
  'gray|red|blue|green|yellow|orange|amber|slate|zinc|neutral|stone|indigo|purple|pink|' +
  'emerald|teal|sky|cyan|lime|rose|violet|fuchsia'
const UTILITIES = 'bg|text|border|ring|divide|placeholder|fill|stroke|from|to|via|outline|decoration|accent|caret'

const RULES: { name: string; pattern: RegExp; hint: string }[] = [
  {
    name: 'Raw-Tailwind-Palette',
    pattern: new RegExp(`(?<![\\w-])(?:[a-z-]+:)*(?:${UTILITIES})-(?:${PALETTE})-\\d{2,3}\\b`, 'g'),
    hint: 'brand-*-Token verwenden (@theme in index.css)',
  },
  {
    // `bg-brand-black` ist ein anderer Farbton als Tailwinds `black`; ein
    // Modal mit `bg-black/40` sieht neben einem mit `bg-brand-black/40` anders aus.
    name: 'Raw-Schwarz',
    pattern: /(?<![\w-])(?:[a-z-]+:)*(?:bg|text|border)-black\b/g,
    hint: 'brand-black verwenden',
  },
  {
    // Nur als JSX-Text (zwischen `>` und `<`) oder als String-Literal, das
    // ausschließlich aus der Glyphe besteht — Kommentare und Fließtext wie
    // „2 × Halbzeit" bleiben erlaubt.
    name: 'Unicode-Icon',
    pattern: />\s*[×✓✗✕✔☰⋮⊘]\s*<|>\s*[✓✗✔]\s+[^<{]|['"][✓✗✕✔☰⋮⊘×]['"]/g,
    hint: 'lucide-react-Icon verwenden',
  },
]

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
      for (const { name, pattern, hint } of RULES) {
        for (const m of line.matchAll(pattern)) {
          out.push(`${rel}:${i + 1}: ${name} „${m[0].trim()}" — ${hint}`)
        }
      }
    })
  }
  return out
}

describe('Design-Tokens: nur brand-Farben, nur lucide-Icons', () => {
  const files = SCAN_DIRS.flatMap(tsxFiles)

  it('findet überhaupt Dateien (Selbsttest des Scanners)', () => {
    expect(files.length).toBeGreaterThan(50)
  })

  it('keine Verstöße im Bestand', () => {
    const entries = files.map(file => ({
      rel: relative(SRC, file).split('\\').join('/'),
      src: readFileSync(file, 'utf8'),
    }))
    const violations = findViolations(entries)
    expect(violations, `\n${violations.join('\n')}\n`).toEqual([])
  })

  it.each([
    ['<p className="text-gray-700">x</p>'],
    ['<div className="hover:bg-green-200" />'],
    ['<div className="fixed inset-0 bg-black/40" />'],
    ['<button onClick={f}>×</button>'],
    ['<button className="a">✓ Annehmen</button>'],
    ["return '✗'"],
  ])('meldet eine frisch eingeschleuste Abweichung: %s', src => {
    expect(findViolations([{ rel: 'pages/ErfundeneSeite.tsx', src }])).toHaveLength(1)
  })

  it.each([
    ['<p className="text-brand-text-muted bg-brand-black/40">x</p>'],
    ['// Anker-Korrektur 0×0 im ersten Layout'],
    ['<p>Gesamt = 2 × Halbzeit + Pause</p>'],
    ['<span>{n}× unverändert</span>'],
  ])('lässt Erlaubtes durch: %s', src => {
    expect(findViolations([{ rel: 'pages/ErfundeneSeite.tsx', src }])).toEqual([])
  })
})
