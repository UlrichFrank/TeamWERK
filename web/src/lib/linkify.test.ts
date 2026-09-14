import { describe, test, expect } from 'vitest'
import { inAppPath, isDocumentLink, splitLinks } from './linkify'

const ORIGIN = 'https://teamwerk.team-stuttgart.org'

describe('splitLinks', () => {
  test('Text ohne URL bleibt ein Textteil', () => {
    expect(splitLinks('Halle gesperrt')).toEqual([{ kind: 'text', value: 'Halle gesperrt' }])
  })

  test('URL mitten im Text wird Link, Umgebung bleibt Text', () => {
    expect(splitLinks(`Turnierplan: ${ORIGIN}/dokumente/datei/12 bitte lesen`)).toEqual([
      { kind: 'text', value: 'Turnierplan: ' },
      { kind: 'link', href: `${ORIGIN}/dokumente/datei/12` },
      { kind: 'text', value: ' bitte lesen' },
    ])
  })

  test('Satzzeichen am Ende gehören zum Satz, nicht zur URL', () => {
    expect(splitLinks(`Plan: ${ORIGIN}/dokumente/datei/12.`)).toEqual([
      { kind: 'text', value: 'Plan: ' },
      { kind: 'link', href: `${ORIGIN}/dokumente/datei/12` },
      { kind: 'text', value: '.' },
    ])
  })

  test('umschließende Klammer wird abgeschnitten, URL-eigene Klammer bleibt', () => {
    expect(splitLinks('(siehe https://a.de/x)')[1]).toEqual({ kind: 'link', href: 'https://a.de/x' })
    expect(splitLinks('https://de.wikipedia.org/wiki/Handball_(Sport)')[0]).toEqual({
      kind: 'link',
      href: 'https://de.wikipedia.org/wiki/Handball_(Sport)',
    })
  })

  test('mehrere URLs und Zeilenumbrüche', () => {
    const parts = splitLinks('A https://a.de\nB https://b.de')
    expect(parts.filter((p) => p.kind === 'link')).toHaveLength(2)
    expect(parts.map((p) => (p.kind === 'text' ? p.value : p.href)).join('')).toBe('A https://a.de\nB https://b.de')
  })

  test('nacktes Schema ohne Host ist kein Link', () => {
    expect(splitLinks('https://')).toEqual([{ kind: 'text', value: 'https://' }])
  })
})

describe('inAppPath', () => {
  test('App-Route derselben Origin → Pfad inkl. Query und Hash', () => {
    expect(inAppPath(`${ORIGIN}/termine?focus=game-5#x`, ORIGIN)).toBe('/termine?focus=game-5#x')
  })

  test('fremde Origin → null', () => {
    expect(inAppPath('https://team-stuttgart.org/info', ORIGIN)).toBeNull()
  })

  test('API-Endpunkte und statische Dateien kennt der Router nicht → null', () => {
    expect(inAppPath(`${ORIGIN}/api/files/3/download`, ORIGIN)).toBeNull()
    expect(inAppPath(`${ORIGIN}/benutzerhandbuch.html`, ORIGIN)).toBeNull()
  })
})

describe('isDocumentLink', () => {
  test('nur der teilbare Dokument-Link', () => {
    expect(isDocumentLink('/dokumente/datei/12')).toBe(true)
    expect(isDocumentLink('/dokumente/12')).toBe(false)
    expect(isDocumentLink('/dokumente/anzeigen/12')).toBe(false)
  })
})
