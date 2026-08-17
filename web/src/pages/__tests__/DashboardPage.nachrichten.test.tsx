import { describe, test, expect, vi, beforeEach } from 'vitest'
import { act, fireEvent, screen } from '@testing-library/react'
import DashboardPage from '../DashboardPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

// Dashboard abonniert Live-Updates; im Test neutralisieren.
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

// useChatEvents wird nicht nur neutralisiert, sondern der Callback festgehalten —
// die Live-Aktualisierung des Badges bei eingeklappter Section lässt sich sonst
// nicht auslösen.
const chatEvents = vi.hoisted(() => ({ cb: null as ((e: string) => void) | null }))
vi.mock('../../hooks/useChatEvents', () => ({
  useChatEvents: (cb: (e: string) => void) => { chatEvents.cb = cb },
}))

// Umschaltbar, damit die Bestands-Tests weiter im Desktop-Layout laufen (dort
// sind alle Sections offen) und die neuen Fälle im Mobil-Layout, wo per Default
// nur 'termine' offen ist und „Nachrichten" damit eingeklappt startet.
const mobile = vi.hoisted(() => ({ current: false }))
vi.mock('../../lib/useMediaQuery', () => ({ useMediaQuery: () => mobile.current }))

beforeEach(() => {
  mobile.current = false
  chatEvents.cb = null
})

const DASHBOARD = {
  currentSeason: null,
  meineTermine: [],
  meineDienste: null,
  carpoolingConfirmed: [],
  carpoolingOpenGroups: [],
}

const CONV_UNREAD = [{
  id: 1, type: 'direct', name: null, unreadCount: 2,
  lastMessage: { body: 'Bis morgen', sentAt: '2026-07-10T10:00:00Z' },
  members: [{ id: 1, name: 'Ich' }, { id: 2, name: 'Anna Trainer' }],
}]
const BC_UNREAD = [{
  id: 5, senderName: 'Bob Vorstand', body: 'Hallenschluss um 22 Uhr',
  sentAt: '2026-07-11T09:00:00Z', isRead: false, isSent: false,
}]

describe('DashboardPage — Nachrichten-Section', () => {
  test('zeigt ungelesene Konversation (Partnername) und Mitteilung (Absender) + Zum-Chat-Link', async () => {
    renderAsPersona(<DashboardPage />, 'spieler', {
      mocks: [
        { url: '/dashboard', data: DASHBOARD },
        { url: '/chat/conversations', data: CONV_UNREAD },
        { url: '/chat/broadcasts', data: BC_UNREAD },
      ],
    })
    await flushAsync()

    expect(screen.getByText('Nachrichten')).toBeInTheDocument()
    // Direkt-Chat ohne Namen → Partnername (Mitglied ≠ eigene id=1)
    expect(screen.getByText('Anna Trainer')).toBeInTheDocument()
    expect(screen.getByText('Bob Vorstand')).toBeInTheDocument()
    expect(screen.getAllByText('Zum Chat').length).toBeGreaterThan(0)
  })

  test('Konversations-Zeile deeplinkt auf die Konversation, Mitteilung auf den Tab', async () => {
    // Regression: die Zeile führte auf das nackte `/chat`. Der Nutzer landete
    // in der Liste, ohne dass die angeklickte Konversation geöffnet und damit
    // als gelesen markiert wurde — alle Badges (Section, Nav-Modul, Hamburger)
    // blieben nach dem Klick unverändert stehen.
    renderAsPersona(<DashboardPage />, 'spieler', {
      mocks: [
        { url: '/dashboard', data: DASHBOARD },
        { url: '/chat/conversations', data: CONV_UNREAD },
        { url: '/chat/broadcasts', data: BC_UNREAD },
      ],
    })
    await flushAsync()

    expect(screen.getByText('Anna Trainer').closest('a')).toHaveAttribute('href', '/chat?conv=1')
    expect(screen.getByText('Bob Vorstand').closest('a')).toHaveAttribute('href', '/chat?tab=broadcasts')
  })

  test('leerer Zustand ohne Ungelesenes', async () => {
    renderAsPersona(<DashboardPage />, 'spieler', {
      mocks: [
        { url: '/dashboard', data: DASHBOARD },
        { url: '/chat/conversations', data: [{ id: 1, type: 'group', name: 'Team', unreadCount: 0, lastMessage: null, members: [] }] },
        { url: '/chat/broadcasts', data: [{ id: 5, senderName: 'Bob', body: 'alt', sentAt: '2026-07-01T09:00:00Z', isRead: true, isSent: false }] },
      ],
    })
    await flushAsync()

    expect(screen.getByText('Keine ungelesenen Nachrichten.')).toBeInTheDocument()
  })
})

// ── Badge am Section-Header ──────────────────────────────────────────────────

function conv(id: number, unreadCount: number) {
  return {
    id,
    type: 'direct',
    name: null,
    unreadCount,
    lastMessage: { body: `Text ${id}`, sentAt: `2026-07-${String(id).padStart(2, '0')}T10:00:00Z` },
    members: [{ id: 1, name: 'Ich' }, { id: 100 + id, name: `Partner ${id}` }],
  }
}

const header = () => screen.getByRole('button', { name: /^Nachrichten/ })

async function renderMobile(conversations: unknown[], broadcasts: unknown[] = []) {
  mobile.current = true
  renderAsPersona(<DashboardPage />, 'spieler', {
    mocks: [
      { url: '/dashboard', data: DASHBOARD },
      { url: '/chat/conversations', data: conversations },
      { url: '/chat/broadcasts', data: broadcasts },
    ],
  })
  await flushAsync()
}

describe('DashboardPage — Badge am Nachrichten-Header', () => {
  test('Badge erscheint, obwohl die Section eingeklappt ist und nie gemountet wurde', async () => {
    // Killer-Case: `Accordion` rendert `{isOpen && children}`. Läge der Abruf
    // in der Section, liefe er hier nie — der Badge bliebe leer.
    await renderMobile([conv(1, 3)])

    expect(screen.queryByText('Partner 1')).not.toBeInTheDocument()
    expect(screen.queryByText('Keine ungelesenen Nachrichten.')).not.toBeInTheDocument()
    expect(header()).toHaveTextContent('3')
  })

  test('Badge zählt Nachrichten, nicht Zeilen', async () => {
    // 7 Konversationen à 2 ungelesen: `rows` bündelt je Konversation zu einer
    // Zeile und deckelt auf 5 — rows.length wäre 5, richtig sind 14.
    await renderMobile(Array.from({ length: 7 }, (_, i) => conv(i + 1, 2)))

    expect(header()).toHaveTextContent('14')

    fireEvent.click(header())
    await flushAsync()
    expect(screen.getAllByRole('listitem')).toHaveLength(5)
  })

  test('Mitteilungen zählen mit', async () => {
    await renderMobile([conv(1, 1)], BC_UNREAD)

    expect(header()).toHaveTextContent('2')
  })

  test('ohne Ungelesenes trägt der Header keine Null', async () => {
    await renderMobile([conv(1, 0)])

    expect(header().textContent?.trim()).toBe('Nachrichten')
  })

  test('chat:new-message erhöht den Badge bei eingeklappter Section', async () => {
    const conversations: unknown[] = [conv(1, 1)]
    await renderMobile(conversations)
    expect(header()).toHaveTextContent('1')

    // Gleiche Array-Referenz wie im Mock → der Nachlade-Request sieht den
    // neuen Stand, ohne dass der Adapter neu verdrahtet werden muss.
    conversations.push(conv(2, 2))

    expect(chatEvents.cb).not.toBeNull()
    await act(async () => {
      chatEvents.cb!('chat:new-message')
      await new Promise(resolve => setTimeout(resolve, 0))
    })

    expect(header()).toHaveTextContent('3')
    // Die Section ist dabei weiterhin zu.
    expect(screen.queryByText('Partner 2')).not.toBeInTheDocument()
  })
})
