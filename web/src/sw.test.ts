import { describe, test, expect, vi, beforeAll } from 'vitest'

// sw.ts registriert beim Import Workbox-Routen (precacheAndRoute liest
// self.__WB_MANIFEST) — im Build injiziert vite-plugin-pwa das Array, im
// Vitest-Lauf muss es hier vorab gestubbt werden, sonst wirft
// `PrecacheController.addToCacheList` ("not-an-array"). Der dynamische Import
// läuft deshalb erst NACH dem Stub (statische Imports würden vor jedem
// Testcode ausgeführt).
let resolveClickTarget: (typeof import('./sw'))['resolveClickTarget']
let applyClickTarget: (typeof import('./sw'))['applyClickTarget']

beforeAll(async () => {
  ;(globalThis as unknown as { __WB_MANIFEST: unknown[] }).__WB_MANIFEST = []
  const mod = await import('./sw')
  resolveClickTarget = mod.resolveClickTarget
  applyClickTarget = mod.applyClickTarget
})

describe('resolveClickTarget', () => {
  test('leerer String ⇒ kein Navigationsziel', () => {
    expect(resolveClickTarget({ url: '' })).toEqual({ navigate: false, url: '/' })
  })

  test('undefined ⇒ kein Navigationsziel', () => {
    expect(resolveClickTarget(undefined)).toEqual({ navigate: false, url: '/' })
  })

  test('null ⇒ kein Navigationsziel', () => {
    expect(resolveClickTarget(null)).toEqual({ navigate: false, url: '/' })
  })

  test('fehlendes url-Feld ⇒ kein Navigationsziel', () => {
    expect(resolveClickTarget({})).toEqual({ navigate: false, url: '/' })
  })

  test('gesetzte URL ⇒ Navigationsziel', () => {
    expect(resolveClickTarget({ url: '/dienste' })).toEqual({ navigate: true, url: '/dienste' })
  })

  test('data selbst eine Zahl ⇒ kein Navigationsziel, wirft nicht', () => {
    expect(() => resolveClickTarget(42)).not.toThrow()
    expect(resolveClickTarget(42)).toEqual({ navigate: false, url: '/' })
  })

  test('data selbst ein Array ⇒ kein Navigationsziel, wirft nicht', () => {
    expect(() => resolveClickTarget([1, 2, 3])).not.toThrow()
    expect(resolveClickTarget([1, 2, 3])).toEqual({ navigate: false, url: '/' })
  })

  test('data selbst ein String ⇒ kein Navigationsziel, wirft nicht', () => {
    expect(resolveClickTarget('/dienste')).toEqual({ navigate: false, url: '/' })
  })

  test('url-Feld ist eine Zahl ⇒ kein Navigationsziel, wirft nicht', () => {
    expect(resolveClickTarget({ url: 42 })).toEqual({ navigate: false, url: '/' })
  })

  test('url-Feld ist ein Array ⇒ kein Navigationsziel, wirft nicht', () => {
    expect(resolveClickTarget({ url: ['/dienste'] })).toEqual({ navigate: false, url: '/' })
  })
})

describe('applyClickTarget (Handler-Pfad mit Fake-Clients)', () => {
  function fakeClient(url: string) {
    return { url, focus: vi.fn(), navigate: vi.fn() }
  }

  test('url: "" mit offenem Fenster ⇒ focus() ja, navigate() nein', () => {
    const client = fakeClient('https://teamwerk.example/aktuell')
    const openWindow = vi.fn()

    applyClickTarget({ navigate: false, url: '/' }, [client], 'https://teamwerk.example', openWindow)

    expect(client.focus).toHaveBeenCalledTimes(1)
    expect(client.navigate).not.toHaveBeenCalled()
    expect(openWindow).not.toHaveBeenCalled()
  })

  test('url: "/dienste" mit offenem Fenster ⇒ focus() und navigate() beide', () => {
    const client = fakeClient('https://teamwerk.example/aktuell')
    const openWindow = vi.fn()

    applyClickTarget(
      { navigate: true, url: '/dienste' },
      [client],
      'https://teamwerk.example',
      openWindow
    )

    expect(client.focus).toHaveBeenCalledTimes(1)
    expect(client.navigate).toHaveBeenCalledWith('/dienste')
    expect(openWindow).not.toHaveBeenCalled()
  })

  test('kein Ziel ohne offenes Fenster ⇒ App-Wurzel öffnen, kein Client-Aufruf', () => {
    const openWindow = vi.fn()

    applyClickTarget({ navigate: false, url: '/' }, [], 'https://teamwerk.example', openWindow)

    expect(openWindow).toHaveBeenCalledWith('/')
  })

  test('gesetztes Ziel ohne offenes Fenster ⇒ neues Fenster mit Ziel-URL', () => {
    const openWindow = vi.fn()

    applyClickTarget(
      { navigate: true, url: '/dienste' },
      [],
      'https://teamwerk.example',
      openWindow
    )

    expect(openWindow).toHaveBeenCalledWith('/dienste')
  })

  test('mehrere Clients: nur der zur Origin passende wird fokussiert', () => {
    const other = fakeClient('https://andere-app.example/')
    const match = fakeClient('https://teamwerk.example/profil')
    const openWindow = vi.fn()

    applyClickTarget(
      { navigate: true, url: '/dienste' },
      [other, match],
      'https://teamwerk.example',
      openWindow
    )

    expect(other.focus).not.toHaveBeenCalled()
    expect(match.focus).toHaveBeenCalledTimes(1)
    expect(match.navigate).toHaveBeenCalledWith('/dienste')
  })
})
