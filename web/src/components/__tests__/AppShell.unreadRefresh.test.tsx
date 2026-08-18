/**
 * Aktualität des Ungelesen-Zählers. Quelle:
 * openspec/changes/chat-badge-live-kanal-robust/specs/chat-unread-badge-pfad/spec.md
 *
 * Der Kern: `chatUnread` hatte genau zwei Schreiber — Mount und SSE-Callback.
 * Eine Homescreen-PWA wird beim Wechsel in den Hintergrund eingefroren und
 * später FORTGESETZT, nicht neu gemountet; fiel in der Zwischenzeit der
 * Live-Kanal aus, gab es keinen Weg zurück zur Wahrheit und die Zahl blieb
 * dauerhaft stehen (meist auf 0). Diese Tests halten den zweiten Weg offen.
 *
 * Bewusst NICHT über `renderAsPersona`: dessen `mocks`-Option kann nur
 * 200-Antworten, und `setupApiMock` läuft im Render — ein vorher gesetzter
 * Fehler-Mock würde überschrieben und der Test wäre aus dem falschen Grund
 * grün. Hier wird der Mock deshalb selbst aufgesetzt, VOR dem Render.
 */
import { describe, test, expect, vi, beforeAll, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { act } from 'react'
import MockAdapter from 'axios-mock-adapter'
import AppShell from '../AppShell'
import { api } from '../../lib/api'
import { AuthContext, type AuthCtx, type User } from '../../contexts/AuthContext'
import { PersonContactProvider } from '../../contexts/PersonContactContext'
import { flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/usePushSubscription', () => ({ usePushSubscription: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../contexts/VersionContext', () => ({
  useVersion: () => ({ version: null, updateAvailable: false, latestVersion: null }),
  VersionProvider: ({ children }: { children: React.ReactNode }) => children,
}))

beforeAll(() => {
  vi.spyOn(window.localStorage.__proto__, 'getItem').mockReturnValue(null)
  vi.spyOn(window.localStorage.__proto__, 'setItem').mockImplementation(() => {})
})

let mock: MockAdapter

const ME = {
  id: 1, email: 'test@test.local', name: 'Test User',
  club_functions: [], is_parent: false, children: [],
}

function conv(unreadCount: number) {
  return {
    id: 1,
    type: 'direct' as const,
    name: 'Konv',
    createdBy: 1,
    unreadCount,
    lastMessage: { body: 'Hallo', sentAt: '2026-08-16T10:00:00Z' },
    members: [{ id: 1, name: 'Ich' }],
  }
}

/** Antwort des Servers für alle folgenden Ladeversuche. */
function serverReturns(unread: number) {
  mock.reset()
  mock.onGet('/chat/conversations').reply(200, [conv(unread)])
  mock.onGet('/chat/broadcasts').reply(200, [])
  mock.onGet('/profile/me').reply(200, ME)
  mock.onAny().reply(200, [])
}

/** Die Chat-Abrufe scheitern (offline, Serverfehler). */
function serverFails() {
  mock.reset()
  mock.onGet('/chat/conversations').networkError()
  mock.onGet('/chat/broadcasts').networkError()
  mock.onGet('/profile/me').reply(200, ME)
  mock.onAny().reply(200, [])
}

const user: User = {
  id: 1,
  email: 'spieler@test.local',
  role: 'standard',
  clubFunctions: ['spieler'],
  isParent: false,
}

const ctx: AuthCtx = {
  user,
  loading: false,
  impersonating: null,
  mapsProvider: 'auto',
  setMapsProvider: () => {},
  capabilities: [],
  hasCapability: () => false,
  navRoutes: ['/', '/profil', '/kalender', '/mein-team', '/dokumente', '/dienste', '/chat'],
  passwordChangeRecommended: false,
  dismissPasswordChangeHint: () => {},
  login: async () => {},
  logout: async () => {},
  startImpersonation: async () => {},
  stopImpersonation: async () => {},
}

async function renderShell() {
  const result = render(
    <AuthContext.Provider value={ctx}>
      <PersonContactProvider>
        <MemoryRouter initialEntries={['/']}>
          <AppShell />
        </MemoryRouter>
      </PersonContactProvider>
    </AuthContext.Provider>,
  )
  await flushAsync()
  await flushAsync()
  return result
}

const vereinHeader = () => screen.getByRole('button', { name: /^Verein/ })

async function becomeVisible() {
  await act(async () => {
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    document.dispatchEvent(new Event('visibilitychange'))
    await new Promise(resolve => setTimeout(resolve, 0))
  })
}

async function goOnline() {
  await act(async () => {
    window.dispatchEvent(new Event('online'))
    await new Promise(resolve => setTimeout(resolve, 0))
  })
}

describe('AppShell — Aktualität des Ungelesen-Zählers', () => {
  beforeEach(() => {
    mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
    serverReturns(0)
  })

  test('Sichtbarwerden lädt die Zahl neu', async () => {
    await renderShell()
    expect(vereinHeader().textContent?.trim()).toBe('Verein')

    // Während die App im Hintergrund war, sind Nachrichten eingetroffen.
    serverReturns(3)
    await becomeVisible()

    expect(vereinHeader()).toHaveTextContent('3')
  })

  test('Rückkehr ins Netz lädt die Zahl neu', async () => {
    await renderShell()

    serverReturns(2)
    await goOnline()

    expect(vereinHeader()).toHaveTextContent('2')
  })

  test('gescheiterter Start-Load zeigt keinen Badge und wird nachgeholt', async () => {
    serverFails()
    await renderShell()

    // Kein Badge, keine Fehlermeldung — aber auch keine gesicherte „0".
    expect(vereinHeader().textContent?.trim()).toBe('Verein')

    serverReturns(4)
    await becomeVisible()

    expect(vereinHeader()).toHaveTextContent('4')
  })

  test('gescheiterter Refetch lässt den zuletzt bekannten Wert stehen', async () => {
    serverReturns(5)
    await renderShell()
    expect(vereinHeader()).toHaveTextContent('5')

    // Netz weg: die Zahl darf nicht auf 0 zurückfallen — „unbekannt" ist keine
    // Aussage über den Posteingang.
    serverFails()
    await becomeVisible()

    expect(vereinHeader()).toHaveTextContent('5')
  })

  test('erfolgreich geladene Null zeigt keinen Badge', async () => {
    serverReturns(0)
    await renderShell()

    expect(vereinHeader().textContent?.trim()).toBe('Verein')
  })

  test('nach Unmount lösen die Ereignisse keinen Request mehr aus', async () => {
    const { unmount } = await renderShell()

    unmount()
    const before = mock.history.get.length
    await becomeVisible()
    await goOnline()

    expect(mock.history.get.length).toBe(before)
  })
})
