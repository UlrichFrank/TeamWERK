import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

/**
 * Gate gegen abgetippte Klassen-Strings — das Gegenstück zu
 * `internal/arch/broadcast_test.go` auf der Frontend-Seite.
 *
 * Die verbindlichen Strings stehen in `lib/buttonStyles.ts` (Capability
 * `component-standards`). Wer sie stattdessen von Hand in eine Seite schreibt,
 * hängt sich von der Fundstelle ab: eine spätere Korrektur erreicht ihn nicht,
 * und genau so sind die vier verschiedenen Kopfzeilen-Höhen entstanden, die
 * dieser Change beseitigt hat.
 *
 * Bewusst textuell und nicht über einen AST: die Strings stehen teils in
 * Template-Literalen mit eingebetteten Ternaries, teils über mehrere Zeilen
 * umgebrochen. Ein AST-Ansatz müsste den JSX-Ausdruck auswerten, um an den
 * tatsächlichen String zu kommen — viel Maschinerie für dieselbe Aussage.
 * Der Preis ist, dass ein anders umbrochener Kopiervorgang durchrutscht; das
 * Gate ist eine Bremse gegen Copy-Paste, kein Beweis.
 */

const SRC = resolve(process.cwd(), 'src')
const SCAN_DIRS = ['pages', 'components']

/**
 * Je Konstante der Teil-String, an dem eine Kopie erkennbar ist — Farbe, Maß
 * und Schriftgröße zusammen. Der Hover- und Disabled-Teil bleibt außen vor:
 * Kopien lassen ihn oft weg, sollen aber trotzdem auffallen.
 */
const METRICS: { constant: string; metric: string }[] = [
  { constant: 'HEADER_H (und alles, was darauf aufbaut)', metric: 'h-11 sm:h-[30px]' },
  { constant: 'BTN_PRIMARY', metric: 'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium' },
  { constant: 'BTN_DANGER', metric: 'bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium' },
  { constant: 'BTN_SECONDARY', metric: 'border border-brand-border text-brand-text rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium' },
  { constant: 'BTN_SMALL', metric: 'bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium' },
]

/**
 * Begründete Ausnahmen. Ein Eintrag, dessen Fundstelle verschwunden ist, lässt
 * den Test ebenfalls fehlschlagen — die Liste soll nicht zur Müllhalde werden.
 */
const ALLOWLIST: { file: string; constant: string; count: number; reason: string }[] = [
  {
    file: 'pages/AdminTrainingsPage.tsx',
    constant: 'BTN_SECONDARY',
    count: 3,
    reason:
      'Abbrechen-Buttons mit hover:bg-brand-surface-card statt hover:bg-brand-table-select. ' +
      'Der Secondary-Button ist in component-standards nicht definiert und existiert im Bestand ' +
      'in vier Hover-Varianten; ihn zu vereinheitlichen ist eine Design-Entscheidung mit ' +
      'sichtbarer Folge, kein Dedupe. Siehe Folge-Change.',
  },
  {
    file: 'pages/MembersPage.tsx',
    constant: 'BTN_SECONDARY',
    count: 1,
    reason: 'Wie AdminTrainingsPage — abweichender Hover, eigener Folge-Change.',
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

/** Die eigentliche Regel — als Funktion, damit der Poison-Test sie durchläuft. */
function findViolations(entries: { rel: string; src: string }[]): string[] {
  const violations: string[] = []
  for (const { rel, src } of entries) {
    for (const { constant, metric } of METRICS) {
      const found = countOccurrences(src, metric)
      if (found === 0) continue
      const allowed = ALLOWLIST.find(a => a.file === rel && a.constant === constant)?.count ?? 0
      if (found > allowed) {
        violations.push(
          `${rel}: ${found}× die Metrik von ${constant} als Literal` +
            (allowed > 0 ? ` (${allowed}× erlaubt)` : '') +
            ` — stattdessen aus '../lib/buttonStyles' importieren.`,
        )
      }
    }
  }
  return violations
}

function countOccurrences(haystack: string, needle: string): number {
  let n = 0
  let i = haystack.indexOf(needle)
  while (i !== -1) {
    n++
    i = haystack.indexOf(needle, i + needle.length)
  }
  return n
}

describe('Button-Klassen-Strings kommen aus lib/buttonStyles', () => {
  const files = SCAN_DIRS.flatMap(tsxFiles)

  it('findet überhaupt Dateien (Selbsttest des Scanners)', () => {
    expect(files.length).toBeGreaterThan(50)
  })

  it('keine abgetippte Kopie außerhalb der Allowlist', () => {
    const entries = files.map(file => ({
      rel: relative(SRC, file).split('\\').join('/'),
      src: readFileSync(file, 'utf8'),
    }))
    const violations = findViolations(entries)
    expect(violations, `\n${violations.join('\n')}\n`).toEqual([])
  })

  // Poison-Sanity: beweist, dass die Regel scharf ist. Ohne das wäre ein Gate,
  // das nie etwas findet, von einem kaputten Gate nicht zu unterscheiden.
  it.each(METRICS)('meldet eine frisch eingeschleuste Kopie von $constant', ({ metric }) => {
    const src = `export default function X() {\n  return <button className="${metric} flex-1" />\n}\n`
    expect(findViolations([{ rel: 'pages/ErfundeneSeite.tsx', src }])).toHaveLength(1)
  })

  it('meldet eine Kopie auch in einer Datei, die für eine ANDERE Konstante erlaubt ist', () => {
    const metric = METRICS.find(m => m.constant === 'BTN_PRIMARY')!.metric
    const src = `<button className="${metric}" />`
    expect(findViolations([{ rel: 'pages/AdminTrainingsPage.tsx', src }])).toHaveLength(1)
  })

  it('meldet eine Kopie über die erlaubte Anzahl hinaus', () => {
    const metric = METRICS.find(m => m.constant === 'BTN_SECONDARY')!.metric
    const src = `<a className="${metric}" /><a className="${metric}" />`
    expect(findViolations([{ rel: 'pages/MembersPage.tsx', src }])).toHaveLength(1)
  })

  it('keine verwaisten Allowlist-Einträge', () => {
    const orphans = ALLOWLIST.filter(entry => {
      const metric = METRICS.find(m => m.constant === entry.constant)?.metric
      if (!metric) return true
      let src: string
      try {
        src = readFileSync(join(SRC, entry.file), 'utf8')
      } catch {
        return true
      }
      return countOccurrences(src, metric) !== entry.count
    })

    expect(
      orphans.map(o => `${o.file} / ${o.constant} (erwartet ${o.count}×)`),
      '\nAllowlist-Eintrag passt nicht mehr zum Code — Eintrag anpassen oder entfernen.\n',
    ).toEqual([])
  })
})
