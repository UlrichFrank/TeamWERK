// Textfilter für die Termin-Ansichten (/dienste, /termine, /kalender).
//
// Bewusst seitenblind: dieses Modul kennt weder Spiele noch Dienste noch
// Abwesenheiten, sondern nur "Textfelder" und "Datumsfelder". Die Zuordnung,
// welches Objekt welche Felder beisteuert, machen die Adapter in den Seiten
// (siehe openspec/changes/termin-textfilter/design.md §6).
//
// Auswertungsmodell — Token-AND, Interpretation-OR:
//   - Der Ausdruck wird an Whitespace zerlegt; JEDES Token muss matchen.
//   - Ein Token matcht, wenn IRGENDEINE seiner Interpretationen greift:
//     Freitext, Datum (TT.MM. / TT.MM.JJJJ) oder Monatsname.
// Das OR ist Absicht: "mar" trifft März UND die Markthalle. Eine exklusive
// Auswertung (erst Datum probieren, sonst Freitext) würde Treffer still
// verschlucken — der Fehler wäre unsichtbar, weil eine leere Liste plausibel
// aussieht (design.md §2).

export interface FilterToken {
  /** Normalisiertes Token für den Freitext-Vergleich. Immer gesetzt. */
  literal: string
  /** Gesetzt, wenn das Token als TT.MM. bzw. TT.MM.JJJJ lesbar ist. */
  date?: { day: number; month: number; year?: number }
  /** 1–12, gesetzt bei eindeutigem Monatsnamen-Präfix ab drei Zeichen. */
  month?: number
}

/** Monatsnamen bereits normalisiert (März → marz). Index 0 = Januar. */
const MONTHS = [
  'januar',
  'februar',
  'marz',
  'april',
  'mai',
  'juni',
  'juli',
  'august',
  'september',
  'oktober',
  'november',
  'dezember',
]

/** Kleinste Präfixlänge, ab der ein Monatsname erkannt wird. */
const MIN_MONTH_PREFIX = 3

// Erlaubt "14.9", "14.09." und "14.09.2026" — nicht aber "14.092026":
// die Jahreszahl ist nur hinter einem Punkt zulässig.
const DATE_RE = /^(\d{1,2})\.(\d{1,2})(?:\.(\d{4})?)?$/

/**
 * Kleinschreibung plus Entfernen der Unicode-Combining-Marks. "Göppingen"
 * wird zu "goppingen".
 *
 * Bewusst KEINE deutsche Transliteration (ö → oe): das wäre eine zweite Regel
 * mit eigener Fehlerklasse, und auf Mobile ist die Umlaut-Tastatur ohnehin
 * direkt erreichbar. "goeppingen" findet "Göppingen" also nicht.
 */
export function normalize(value: string): string {
  return value
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
}

/**
 * Liefert den Monatsindex (1–12) zu einem Präfix, falls dieser mindestens
 * MIN_MONTH_PREFIX Zeichen lang ist und genau einen Monat trifft.
 *
 * Zwei Zeichen wären mehrdeutig ("ju" → Juni und Juli). Ab drei Zeichen wird
 * die Ergebnismenge beim Weitertippen monoton kleiner — die Erwartung an ein
 * Filterfeld (design.md §4).
 */
function monthFromPrefix(token: string): number | undefined {
  if (token.length < MIN_MONTH_PREFIX) return undefined
  let hit: number | undefined
  for (let i = 0; i < MONTHS.length; i++) {
    if (!MONTHS[i].startsWith(token)) continue
    if (hit !== undefined) return undefined // mehrdeutig
    hit = i + 1
  }
  return hit
}

function dateFromToken(token: string): FilterToken['date'] {
  const m = DATE_RE.exec(token)
  if (!m) return undefined
  const day = parseInt(m[1], 10)
  const month = parseInt(m[2], 10)
  if (day < 1 || day > 31 || month < 1 || month > 12) return undefined
  const year = m[3] ? parseInt(m[3], 10) : undefined
  return { day, month, year }
}

/**
 * Zerlegt den Filterausdruck in Tokens und berechnet je Token alle
 * Interpretationen vor. Ein leerer oder reiner Whitespace-Ausdruck ergibt ein
 * leeres Array — und damit einen wirkungslosen Filter.
 */
export function parseQuery(q: string): FilterToken[] {
  if (!q) return []
  return q
    .split(/\s+/)
    .filter(Boolean)
    .map(raw => {
      const literal = normalize(raw)
      return {
        literal,
        date: dateFromToken(literal),
        month: monthFromPrefix(literal),
      }
    })
}

/** Zerlegt einen API-Datumswert in seine Bestandteile.
 *
 * Die API liefert Datumsfelder als ISO-Timestamp ("2026-09-14T00:00:00Z"),
 * deshalb wird strikt auf die ersten zehn Zeichen geschnitten statt über
 * `new Date()` zu gehen — sonst verschiebt die Zeitzone den Tag (Gotcha
 * "SQLite DATE-Felder" in docs/agent/06-gotchas.md).
 */
function splitDate(value: string): { year: number; month: number; day: number } | null {
  const iso = value.slice(0, 10)
  if (iso.length < 10) return null
  const year = parseInt(iso.slice(0, 4), 10)
  const month = parseInt(iso.slice(5, 7), 10)
  const day = parseInt(iso.slice(8, 10), 10)
  if (!year || !month || !day) return null
  return { year, month, day }
}

function tokenMatchesDates(token: FilterToken, dates: readonly string[]): boolean {
  if (!token.date && !token.month) return false
  for (const raw of dates) {
    if (!raw) continue
    const d = splitDate(raw)
    if (!d) continue
    if (token.date) {
      // Ohne Jahresangabe wird das Jahr ignoriert: eine Saison läuft über den
      // Jahreswechsel, ein geratenes Jahr würde die halbe Liste still
      // verwerfen (design.md §3).
      const yearOk = token.date.year === undefined || token.date.year === d.year
      if (yearOk && token.date.day === d.day && token.date.month === d.month) return true
    }
    if (token.month !== undefined && token.month === d.month) return true
  }
  return false
}

/**
 * Prüft ein Objekt gegen den geparsten Ausdruck.
 *
 * @param tokens Ergebnis von parseQuery
 * @param text   Durchsuchbare Textfelder des Objekts (leere/undefined werden übersprungen)
 * @param dates  Datumsfelder des Objekts, roh aus der API (ISO-Timestamp erlaubt)
 */
export function matchesQuery(
  tokens: readonly FilterToken[],
  text: readonly (string | null | undefined)[],
  dates: readonly (string | null | undefined)[] = [],
): boolean {
  if (tokens.length === 0) return true

  const haystack: string[] = []
  for (const field of text) {
    if (field) haystack.push(normalize(field))
  }
  const dateFields: string[] = []
  for (const field of dates) {
    if (field) dateFields.push(field)
  }

  return tokens.every(token => {
    for (const field of haystack) {
      if (field.includes(token.literal)) return true
    }
    return tokenMatchesDates(token, dateFields)
  })
}
