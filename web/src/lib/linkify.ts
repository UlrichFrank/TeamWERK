// Zerlegt Freitext (Chat-Nachricht, Termin-Hinweis) in Text- und Link-Teile.
// Reine Funktionen ohne DOM — die Darstellung übernimmt `LinkifiedText`.

export type TextPart = { kind: 'text'; value: string } | { kind: 'link'; href: string }

const URL_RE = /https?:\/\/[^\s]+/g

// Satzzeichen am Ende gehören fast immer zum Satz, nicht zur URL
// („Turnierplan: https://…/dokumente/datei/12.").
const TRAILING_PUNCT = '.,;:!?\'"»]}'

function trimTrailing(url: string): string {
  let end = url.length
  while (end > 0) {
    const c = url[end - 1]
    if (TRAILING_PUNCT.includes(c)) { end--; continue }
    // Schließende Klammer nur abschneiden, wenn sie nicht in der URL geöffnet
    // wurde („(siehe https://…/datei/12)" vs. Wikipedia-Links mit Klammern).
    if (c === ')') {
      const head = url.slice(0, end)
      const open = head.split('(').length - 1
      const close = head.split(')').length - 1
      if (close > open) { end--; continue }
    }
    break
  }
  return url.slice(0, end)
}

export function splitLinks(text: string): TextPart[] {
  const parts: TextPart[] = []
  let last = 0
  for (const m of text.matchAll(URL_RE)) {
    const start = m.index ?? 0
    const href = trimTrailing(m[0])
    if (!/^https?:\/\/[^/]/.test(href)) continue
    if (start > last) parts.push({ kind: 'text', value: text.slice(last, start) })
    parts.push({ kind: 'link', href })
    last = start + href.length
  }
  if (last < text.length) parts.push({ kind: 'text', value: text.slice(last) })
  return parts
}

/**
 * Pfad innerhalb der SPA, wenn `href` auf eine App-Route derselben Origin zeigt —
 * sonst `null` (fremde Seite, API-Endpunkt oder statische Datei wie
 * `/benutzerhandbuch.html`, die der Router nicht kennt).
 */
export function inAppPath(href: string, origin: string): string | null {
  let url: URL
  try {
    url = new URL(href)
  } catch {
    return null
  }
  if (url.origin !== origin) return null
  if (url.pathname.startsWith('/api/')) return null
  const lastSegment = url.pathname.split('/').pop() ?? ''
  if (lastSegment.includes('.')) return null
  return url.pathname + url.search + url.hash
}

/** Teilbarer Dokument-Link aus „Link kopieren" in /dokumente. */
export function isDocumentLink(path: string): boolean {
  return /^\/dokumente\/datei\/\d+$/.test(path)
}
