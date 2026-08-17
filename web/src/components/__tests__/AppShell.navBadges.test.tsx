/**
 * Badge-Pfad in der Navigation: Modul-Header „Verein" und Hinweis-Punkt am
 * Hamburger. Quelle: openspec/changes/nachrichten-badge-pfad/specs/
 * chat-unread-badge-pfad/spec.md
 *
 * Kern der Anforderung ist, dass die Zahl sichtbar wird, OHNE dass der Nutzer
 * vorher etwas aufklappen muss — das eingeklappte Modul rendert seine Items
 * gar nicht, und die mobile Sidebar ist bis zum Antippen unsichtbar.
 */
import { describe, test, expect, vi, beforeAll } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import AppShell from '../AppShell'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { setupApiMock } from '../../test/apiMock'
import { AuthContext, type AuthCtx, type User } from '../../contexts/AuthContext'
import { PersonContactProvider } from '../../contexts/PersonContactContext'

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

const chatMocks = (conversations: unknown[], broadcasts: unknown[] = []) => [
  { url: '/chat/conversations', data: conversations },
  { url: '/chat/broadcasts', data: broadcasts },
]

async function renderShell(conversations: unknown[], broadcasts: unknown[] = []) {
  renderAsPersona(<AppShell />, 'spieler', {
    route: '/',
    mocks: chatMocks(conversations, broadcasts),
  })
  await flushAsync()
  await flushAsync()
}

// Eigener Renderer für den Fall „Nutzer ohne /chat": renderAsPersona leitet
// navRoutes aus der Persona ab und enthält /chat immer.
async function renderShellWithoutChat(conversations: unknown[]) {
  setupApiMock(chatMocks(conversations))
  const user: User = {
    id: 1,
    email: 'ohnechat@test.local',
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
    // /chat fehlt bewusst; die übrigen Verein-Items bleiben, damit das Modul
    // überhaupt gerendert wird.
    navRoutes: ['/', '/profil', '/kalender', '/mein-team', '/dokumente', '/dienste'],
    passwordChangeRecommended: false,
    dismissPasswordChangeHint: () => {},
    login: async () => {},
    logout: async () => {},
    startImpersonation: async () => {},
    stopImpersonation: async () => {},
  }
  render(
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
}

// Der Modul-Header trägt bei Unread den Badge im Accessible Name („Verein 3"),
// deshalb Regex statt exaktem String.
const vereinHeader = () => screen.getByRole('button', { name: /^Verein/ })
const hamburger = () => screen.getByRole('button', { name: /^Menü öffnen/ })
const dot = () => hamburger().querySelector('.bg-brand-yellow')

describe('AppShell — Badge am Modul-Header', () => {
  test('eingeklapptes Modul „Verein" zeigt die Zahl, obwohl der Eintrag nicht gerendert ist', async () => {
    await renderShell([conv(1, 2), conv(2, 1)])

    // Auf '/' ist „Nutzer" das offene Modul — „Nachrichten" steht damit nicht
    // im DOM. Genau das ist der Fall, den der Badge lösen soll.
    expect(screen.queryByText('Nachrichten')).not.toBeInTheDocument()
    expect(vereinHeader()).toHaveTextContent('3')
  })

  test('bei offenem Modul tragen Header und Eintrag den Badge gleichzeitig', async () => {
    await renderShell([conv(1, 3)])

    fireEvent.click(vereinHeader())
    await flushAsync()

    expect(vereinHeader()).toHaveTextContent('3')
    const item = screen.getByText('Nachrichten').closest('a')
    expect(item).toHaveTextContent('3')
  })

  test('ohne Ungelesenes trägt der Header keine Null', async () => {
    await renderShell([conv(1, 0)])

    expect(vereinHeader().textContent?.trim()).toBe('Verein')
  })

  test('Mitteilungen zählen im Modul-Badge mit', async () => {
    await renderShell([conv(1, 1)], [
      { id: 9, senderName: 'Vorstand', body: 'Info', sentAt: '2026-08-16T09:00:00Z', isRead: false, isSent: false },
    ])

    expect(vereinHeader()).toHaveTextContent('2')
  })

  test('ohne /chat in navRoutes bleibt der Header ohne Badge', async () => {
    await renderShellWithoutChat([conv(1, 5)])

    // Die Summe läuft über die sichtbaren Items — ein Eintrag, den der Nutzer
    // nicht sehen darf, zählt nicht mit.
    expect(vereinHeader().textContent?.trim()).toBe('Verein')
  })
})

describe('AppShell — Hinweis-Punkt am Hamburger', () => {
  test('Punkt und Zahl im aria-label bei geschlossener Sidebar', async () => {
    await renderShell([conv(1, 2), conv(2, 1)])

    expect(dot()).not.toBeNull()
    expect(hamburger()).toHaveAttribute('aria-label', 'Menü öffnen, 3 ungelesene Nachrichten')
  })

  test('Punkt trägt keine Zahl', async () => {
    await renderShell([conv(1, 12)])

    expect(dot()?.textContent).toBe('')
    expect(hamburger().textContent?.trim()).toBe('')
  })

  test('Singular bei genau einer ungelesenen Nachricht', async () => {
    await renderShell([conv(1, 1)])

    expect(hamburger()).toHaveAttribute('aria-label', 'Menü öffnen, 1 ungelesene Nachricht')
  })

  test('ohne Ungelesenes kein Punkt und unverändertes aria-label', async () => {
    await renderShell([conv(1, 0)])

    expect(dot()).toBeNull()
    expect(hamburger()).toHaveAttribute('aria-label', 'Menü öffnen')
  })

  test('ohne /chat in navRoutes kein Punkt', async () => {
    await renderShellWithoutChat([conv(1, 5)])

    expect(dot()).toBeNull()
    expect(hamburger()).toHaveAttribute('aria-label', 'Menü öffnen')
  })
})
