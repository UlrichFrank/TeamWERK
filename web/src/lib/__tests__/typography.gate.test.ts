import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

/**
 * Gate gegen handgesetzte Überschriften und Menüeinträge — Gegenstück zu
 * `buttonStyles.gate.test.ts` für `lib/typography.ts` (Change
 * `typografie-rollen`).
 *
 * Anders als das Button-Gate sucht dieses keine wörtlichen Kopien: eine
 * handgesetzte Überschrift ist eine freie Kombination (`text-lg font-semibold`,
 * `font-bold text-base` …), keine Kopie. Erkannt wird deshalb am Tag:
 *
 * - Jede `<h1>`/`<h2>`/`<h3>` MUSS eine Rollen-Konstante referenzieren und DARF
 *   daneben keine Größen- oder Schnittklasse setzen. Farb- und Zustandsklassen
 *   (gedämpft, durchgestrichen) bleiben erlaubt — das ist Zustand, nicht Rolle.
 * - Jedes Element mit `role="menuitem"` MUSS `MENU_ITEM` oder
 *   `MENU_ITEM_DANGER` referenzieren.
 *
 * Bewusst textuell (wie das Button-Gate): der Tag wird mit Klammer- und
 * Quote-Zählung bis zu seinem `>` gelesen. Eine Überschrift, die als `<div>`
 * gebaut ist, sieht das Gate nicht — die bekannte Grenze.
 */

const SRC = resolve(process.cwd(), 'src')
const SCAN_DIRS = ['pages', 'components']

const HEADING_ROLES = ['PAGE_TITLE', 'ENTRY_TITLE', 'MODAL_TITLE', 'SECTION_TITLE', 'SUBSECTION_TITLE', 'OVERLINE']
const MENU_ROLES = ['MENU_ITEM', 'MENU_ITEM_DANGER']

type Rule = 'heading' | 'menuitem'

/**
 * Begründete Ausnahmen. Ein Eintrag, dessen Fundstelle verschwunden ist, lässt
 * den Test ebenfalls fehlschlagen.
 */
const ALLOWLIST: { file: string; rule: Rule; count: number; reason: string }[] = []

/**
 * Übergangsliste während der Migration: Dateien, die heute noch verstoßen.
 * Schrumpft je Task-Gruppe; eine Datei, die nicht mehr verstößt, muss hier
 * raus (sonst rot) — die Liste kann also nur kleiner werden.
 */
const MIGRATION_PENDING: string[] = [
  'components/admin/MemberAdminTab.tsx',
  'components/admin/MemberDatenschutzTab.tsx',
  'components/admin/MemberFamilieTab.tsx',
  'components/admin/MemberKontaktTab.tsx',
  'components/admin/MemberStammdatenTab.tsx',
  'components/AttendanceStatsView.tsx',
  'components/FileViewer.tsx',
  'components/H4AImportModal.tsx',
  'components/MarkdownRenderer.tsx',
  'components/profile/ProfileAccountTab.tsx',
  'components/profile/ProfileBankTab.tsx',
  'components/profile/ProfileDatenschutzTab.tsx',
  'components/profile/ProfileKalenderTab.tsx',
  'components/profile/ProfileMemberTab.tsx',
  'components/profile/ProfileMiscTab.tsx',
  'components/profile/ProfileProfilTab.tsx',
  'components/SaisonbilanzPanel.tsx',
  'components/SpielberichtPanel.tsx',
  'components/SpieltagDetailModal.tsx',
  'components/staffeln/RefereeView.tsx',
  'components/staffeln/Spielmatrix.tsx',
  'components/staffeln/TeamStatsViews.tsx',
  'pages/admin/BeitragslaufPage.tsx',
  'pages/admin/TresorPage.tsx',
  'pages/admin/WartungsmodusPage.tsx',
  'pages/AdminDutyTemplatesPage.tsx',
  'pages/AdminDutyTypesPage.tsx',
  'pages/AdminKaderPage.tsx',
  'pages/AdminSettingsPage.tsx',
  'pages/AdminTrainingsPage.tsx',
  'pages/AdminUsersPage.tsx',
  'pages/AdminVenuesPage.tsx',
  'pages/ChatPage.tsx',
  'pages/ChildProfilePage.tsx',
  'pages/DashboardPage.tsx',
  'pages/DatenschutzPage.tsx',
  'pages/DienstRanglistePage.tsx',
  'pages/DocumentsPage.tsx',
  'pages/DutyInstructionPage.tsx',
  'pages/DutyPage.tsx',
  'pages/ForgotPasswordPage.tsx',
  'pages/KalenderPage.tsx',
  'pages/LoginPage.tsx',
  'pages/MatchReportFormPage.tsx',
  'pages/MatchReportListPage.tsx',
  'pages/MatchReportPendingListPage.tsx',
  'pages/MeinTeamPage.tsx',
  'pages/MemberDetailPage.tsx',
  'pages/MembersPage.tsx',
  'pages/MitfahrgelegenheitenPage.tsx',
  'pages/ProfilAnwesenheitPage.tsx',
  'pages/ProfilePage.tsx',
  'pages/ProfilTrainingstagebuchPage.tsx',
  'pages/RegisterPage.tsx',
  'pages/RequestMembershipPage.tsx',
  'pages/ResetPasswordPage.tsx',
  'pages/SepaMandatViewerPage.tsx',
  'pages/StaffelnPage.tsx',
  'pages/TeamAnwesenheitPage.tsx',
  'pages/TeamTrainingstagebuchPage.tsx',
  'pages/TermineDetailPage.tsx',
  'pages/TerminePage.tsx',
  'pages/UebungsgruppenPage.tsx',
  'pages/VideoDetailPage.tsx',
  'pages/VideosPage.tsx',
]

interface Tag {
  name: string
  text: string
  line: number
}

/**
 * Alle JSX-Öffnungstags einer Datei, jeweils bis zu ihrem schließenden `>`.
 * Zählt `{}`-Tiefe und Quotes; `${…}` in einem Template-Literal öffnet eine
 * Klammerebene, nach deren `}` das Literal weiterläuft.
 */
function openingTags(src: string): Tag[] {
  const tags: Tag[] = []
  const re = /<([a-zA-Z][\w.]*)/g
  let m: RegExpExecArray | null
  while ((m = re.exec(src))) {
    const start = m.index
    const templateDepths: number[] = []
    let depth = 0
    let quote: string | null = null
    let end = -1
    for (let i = start + m[0].length; i < src.length; i++) {
      const c = src[i]
      if (quote) {
        if (c === '\\') i++
        else if (c === quote) quote = null
        else if (quote === '`' && c === '$' && src[i + 1] === '{') {
          i++
          depth++
          templateDepths.push(depth)
          quote = null
        }
        continue
      }
      if (c === '"' || c === "'" || c === '`') quote = c
      else if (c === '{') depth++
      else if (c === '}') {
        if (templateDepths[templateDepths.length - 1] === depth) {
          templateDepths.pop()
          quote = '`'
        }
        depth--
      } else if (c === '>' && depth === 0) {
        end = i
        break
      }
    }
    if (end === -1) continue
    tags.push({ name: m[1], text: src.slice(start, end + 1), line: src.slice(0, start).split('\n').length })
  }
  return tags
}

/**
 * Block- und Zeilenkommentare durch Leerraum ersetzen (Zeilenumbrüche bleiben,
 * damit die Zeilennummern stimmen) — ein `<h1>` im Kommentartext ist kein Tag.
 * Zeilenkommentare nur am Zeilenanfang, damit `https://` in Strings überlebt.
 */
function stripComments(src: string): string {
  const blank = (s: string) => s.replace(/[^\n]/g, ' ')
  return src.replace(/\/\*[\s\S]*?\*\//g, blank).replace(/^[ \t]*\/\/.*$/gm, blank)
}

/** Der `className`-Ausdruck eines Tags, oder `null`, wenn es keinen gibt. */
function classNameExpr(tagText: string): string | null {
  const i = tagText.search(/\bclassName=/)
  if (i === -1) return null
  const rest = tagText.slice(i + 'className='.length)
  if (rest[0] === '"' || rest[0] === "'") {
    const close = rest.indexOf(rest[0], 1)
    return rest.slice(0, close + 1)
  }
  if (rest[0] === '{') {
    let depth = 0
    for (let j = 0; j < rest.length; j++) {
      if (rest[j] === '{') depth++
      else if (rest[j] === '}') { depth--; if (depth === 0) return rest.slice(0, j + 1) }
    }
  }
  return rest
}

const SIZE_OR_WEIGHT =
  /(^|[\s"'`}])(?:[a-z]+:)?(text-(?:xs|sm|base|lg|xl|[2-9]xl|\[[^\]\s]+\])|font-(?:medium|semibold|bold|extrabold|black))(?=[\s"'`$]|$)/

function refs(expr: string, names: string[]): boolean {
  return names.some(n => new RegExp(`\\b${n}\\b`).test(expr))
}

interface Violation { rel: string; rule: Rule; line: number; msg: string }

/** Die eigentliche Regel — als Funktion, damit die Poison-Tests sie durchlaufen. */
function findViolations(entries: { rel: string; src: string }[]): Violation[] {
  const out: Violation[] = []
  for (const { rel, src } of entries) {
    for (const tag of openingTags(stripComments(src))) {
      const expr = classNameExpr(tag.text)
      if (/^h[1-3]$/.test(tag.name)) {
        if (expr === null) {
          out.push({ rel, rule: 'heading', line: tag.line, msg: `<${tag.name}> ohne Rollen-Konstante` })
          continue
        }
        const literal = SIZE_OR_WEIGHT.exec(expr)
        if (literal) {
          out.push({ rel, rule: 'heading', line: tag.line, msg: `<${tag.name}> setzt „${literal[2]}“ von Hand` })
        } else if (!refs(expr, HEADING_ROLES)) {
          out.push({ rel, rule: 'heading', line: tag.line, msg: `<${tag.name}> ohne Rollen-Konstante` })
        }
      }
      if (/\brole=["']menuitem["']/.test(tag.text) && (expr === null || !refs(expr, MENU_ROLES))) {
        out.push({ rel, rule: 'menuitem', line: tag.line, msg: 'role="menuitem" ohne MENU_ITEM/MENU_ITEM_DANGER' })
      }
    }
  }
  return out
}

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

function countBy(vs: Violation[], rel: string, rule: Rule): number {
  return vs.filter(v => v.rel === rel && v.rule === rule).length
}

describe('Überschriften und Menüeinträge nutzen lib/typography', () => {
  const entries = SCAN_DIRS.flatMap(tsxFiles).map(file => ({
    rel: relative(SRC, file).split('\\').join('/'),
    src: readFileSync(file, 'utf8'),
  }))
  const all = findViolations(entries)

  it('findet überhaupt Überschriften (Selbsttest des Scanners)', () => {
    const headings = entries.flatMap(e => openingTags(stripComments(e.src))).filter(t => /^h[1-3]$/.test(t.name))
    expect(headings.length).toBeGreaterThan(100)
  })

  it('keine handgesetzte Überschrift und kein Menüeintrag ohne Konstante', () => {
    const offending = all.filter(v => {
      if (MIGRATION_PENDING.includes(v.rel)) return false
      const allowed = ALLOWLIST.find(a => a.file === v.rel && a.rule === v.rule)?.count ?? 0
      return countBy(all, v.rel, v.rule) > allowed
    })
    expect(
      offending.map(v => `${v.rel}:${v.line} ${v.msg} — Konstante aus '../lib/typography' importieren.`),
      '\n',
    ).toEqual([])
  })

  it('MIGRATION_PENDING enthält nur Dateien, die noch verstoßen', () => {
    const clean = MIGRATION_PENDING.filter(f => !all.some(v => v.rel === f))
    expect(clean, '\nDatei verstößt nicht mehr — aus MIGRATION_PENDING entfernen.\n').toEqual([])
  })

  it('keine verwaisten Allowlist-Einträge', () => {
    const orphans = ALLOWLIST.filter(a => countBy(all, a.file, a.rule) !== a.count)
    expect(
      orphans.map(o => `${o.file} / ${o.rule} (erwartet ${o.count}×)`),
      '\nAllowlist-Eintrag passt nicht mehr zum Code — Eintrag anpassen oder entfernen.\n',
    ).toEqual([])
  })

  // Poison-Sanity: beweist, dass die Regel scharf ist.
  const poison = (src: string) => findViolations([{ rel: 'pages/ErfundeneSeite.tsx', src }])

  it('meldet eine handgesetzte <h2>', () => {
    expect(poison('<h2 className="text-lg font-semibold text-brand-text mb-4">X</h2>')).toHaveLength(1)
  })

  it('meldet eine Größenklasse neben der Konstante', () => {
    expect(poison('<h2 className={`${SECTION_TITLE} text-sm`}>X</h2>')).toHaveLength(1)
  })

  it('meldet eine Überschrift ohne Klasse', () => {
    expect(poison('<h3>X</h3>')).toHaveLength(1)
  })

  it('meldet eine handgesetzte Überschrift in einem mehrzeiligen Template-Literal mit Ternary', () => {
    expect(poison('<h1\n  className={`font-bold ${x ? "a" : "b"}`}\n>X</h1>')).toHaveLength(1)
  })

  it('ignoriert Überschriften-Tags in Kommentaren', () => {
    expect(poison('// die <h1> trägt keinen Badge\n{/* <h2 className="text-lg">X</h2> */}')).toEqual([])
  })

  it('lässt Konstante plus Layout- und Zustandsklassen durch', () => {
    expect(poison('<h2 className={`${SECTION_TITLE} mb-4 truncate`}>X</h2>')).toEqual([])
    expect(poison('<h1 className={`${PAGE_TITLE} ${c ? "line-through opacity-60" : ""}`}>X</h1>')).toEqual([])
    expect(poison('<h2 id={id} className={MODAL_TITLE}>X</h2>')).toEqual([])
  })

  it('verwechselt Farbklassen nicht mit Größenklassen', () => {
    expect(poison('<h2 className={`${SECTION_TITLE} text-brand-text-muted`}>X</h2>')).toEqual([])
  })

  it('meldet role="menuitem" ohne Konstante und lässt die Konstante durch', () => {
    expect(poison('<button role="menuitem" onClick={() => go()} className="w-full px-4 text-xs">X</button>')).toHaveLength(1)
    expect(poison('<button role="menuitem" onClick={() => go()} className={MENU_ITEM}>X</button>')).toEqual([])
  })
})
