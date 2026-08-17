/**
 * App-Icon-Badge (`navigator.setAppBadge`) im AppShell.
 *
 * Der Service Worker setzt die Zahl beim Push (`web/src/sw.ts`), der AppShell
 * hält sie danach live nach. Beide schreiben auf dieselbe, geräteweite Zahl —
 * deshalb ist hier nicht nur wichtig, WANN gesetzt, sondern vor allem, wann
 * NICHT gelöscht wird: ein `clearAppBadge()` aus dem Frontend überschreibt
 * stillschweigend, was der Push gerade korrekt gesetzt hat, und es gibt keinen
 * zweiten Weg, die Zahl zurückzuholen (bis zur nächsten Push).
 *
 * Die beiden Regressionsfälle stammen aus einem realen Bericht (iOS-PWA:
 * „Nachrichten kommen an, Badge bleibt weg"):
 *   1. Start der App — `chatUnread` steht auf dem Initialwert 0, die echte Zahl
 *      ist noch unterwegs.
 *   2. Der Ladeversuch scheitert (offline/Funkloch beim App-Start) — der Fehler
 *      wird bewusst geschluckt, die Zahl bleibt unbekannt.
 * In beiden Fällen ist „unbekannt" nicht dasselbe wie „null".
 */
import { describe, test, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest'
import AppShell from '../AppShell'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { api } from '../../lib/api'

vi.mock('../../hooks/usePushSubscription', () => ({ usePushSubscription: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../contexts/VersionContext', () => ({
  useVersion: () => ({ version: null, updateAvailable: false, latestVersion: null }),
  VersionProvider: ({ children }: { children: React.ReactNode }) => children,
}))

const setAppBadge = vi.fn(() => Promise.resolve())
const clearAppBadge = vi.fn(() => Promise.resolve())

beforeAll(() => {
  // jsdom kennt die Badging-API nicht; der Produktionscode gated auf
  // `'setAppBadge' in navigator`, deshalb echte Properties statt Stubs am Objekt.
  Object.defineProperty(navigator, 'setAppBadge', { value: setAppBadge, configurable: true })
  Object.defineProperty(navigator, 'clearAppBadge', { value: clearAppBadge, configurable: true })
  vi.spyOn(window.localStorage.__proto__, 'getItem').mockReturnValue(null)
  vi.spyOn(window.localStorage.__proto__, 'setItem').mockImplementation(() => {})
})

beforeEach(() => {
  setAppBadge.mockClear()
  clearAppBadge.mockClear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

function conv(id: number, unreadCount: number) {
  return {
    id,
    type: 'direct' as const,
    name: `Konv ${id}`,
    createdBy: 1,
    unreadCount,
    lastMessage: { body: 'Hallo', sentAt: '2026-08-16T10:00:00Z' },
    members: [{ id: 1, name: 'Ich' }],
  }
}

async function renderShell(conversations: unknown[], broadcasts: unknown[] = []) {
  renderAsPersona(<AppShell />, 'spieler', {
    route: '/',
    mocks: [
      { url: '/chat/conversations', data: conversations },
      { url: '/chat/broadcasts', data: broadcasts },
    ],
  })
  await flushAsync()
  await flushAsync()
}

/**
 * Rendert den Shell mit einem fehlschlagenden `/chat/conversations`.
 * Über `api.get` statt über den Mock-Adapter, weil dessen Catch-all-Handler
 * (`onAny().reply(200, [])`) in `setupApiMock` vor jedem später registrierten
 * Fehlerfall greift — die Reihenfolge ist dort nicht steuerbar.
 */
async function renderShellWithFailingLoad() {
  // Vor dem Render: die Ladeeffekte laufen beim ersten Render an, ein später
  // gesetzter Spy käme zu spät.
  vi.spyOn(api, 'get').mockImplementation(((url: string) => {
    if (url === '/chat/conversations') return Promise.reject(new Error('offline'))
    return Promise.resolve({ data: url === '/profile/me' ? { children: [] } : [] })
  }) as typeof api.get)
  renderAsPersona(<AppShell />, 'spieler', { route: '/' })
  await flushAsync()
  await flushAsync()
}

describe('AppShell – App-Icon-Badge', () => {
  test('setzt die Zahl, sobald die Ungelesen-Zählung geladen ist', async () => {
    await renderShell([conv(1, 2), conv(2, 1)])

    expect(setAppBadge).toHaveBeenCalledWith(3)
  })

  test('löscht das Badge, wenn nichts ungelesen ist', async () => {
    await renderShell([conv(1, 0)])

    expect(clearAppBadge).toHaveBeenCalled()
    expect(setAppBadge).not.toHaveBeenCalled()
  })

  // Regression: der Effekt hing allein an `chatUnread`, das mit 0 initialisiert
  // wird. Beim Mount lief deshalb IMMER erst ein clearAppBadge() — auf iOS
  // sichtbar als „Badge verschwindet beim Öffnen der App, obwohl noch etwas
  // ungelesen ist". Die vom Push gesetzte Zahl war damit weg, bevor die echte
  // Zählung überhaupt eintraf.
  test('löscht das Badge nicht, bevor die erste Zählung da ist', async () => {
    await renderShell([conv(1, 4)])

    expect(clearAppBadge).not.toHaveBeenCalled()
    expect(setAppBadge).toHaveBeenCalledWith(4)
  })

  // Regression: schlägt der Ladeversuch fehl, schluckt `catch {}` den Fehler und
  // `chatUnread` bleibt auf 0 stehen — nicht, weil nichts ungelesen wäre,
  // sondern weil es unbekannt ist. Ein clearAppBadge() macht daraus eine
  // Aussage, die der Code nicht treffen kann, und die Zahl kommt erst mit der
  // nächsten Push zurück.
  test('lässt das Badge unangetastet, wenn die Zählung nicht geladen werden kann', async () => {
    await renderShellWithFailingLoad()

    expect(clearAppBadge).not.toHaveBeenCalled()
    expect(setAppBadge).not.toHaveBeenCalled()
  })
})
