import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

beforeAll(() => {
  // scrollIntoView existiert in jsdom nicht — brauchen wir aber, um zu
  // verifizieren, dass der Deep-Link am Ende der Konversation landet.
  Element.prototype.scrollIntoView = vi.fn()
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
})

const CONV = {
  id: 55,
  type: 'direct' as const,
  name: 'Direkt an Anna',
  createdBy: 1,
  unreadCount: 0,
  lastMessage: null,
  members: [
    { id: 1, name: 'Ich' },
    { id: 42, name: 'Anna' },
  ],
}

const MESSAGES = Array.from({ length: 20 }, (_, i) => ({
  id: i + 1,
  senderId: 42,
  senderName: 'Anna',
  preview: `Alt-${i + 1}`,
  truncated: false,
  sentAt: '2026-06-28T10:00:00Z',
  replyToId: null,
  replyToBody: null,
  replyToSenderName: null,
  editedAt: null,
  deletedAt: null,
  isSystem: false,
  reactions: [],
}))

describe('ChatPage — Deep-Link ?openUser=', () => {
  test('landet am Ende der Konversation (nicht am Anfang)', async () => {
    // Regression: früher rief der openUser-Effekt setActiveConv+loadMessages
    // direkt und umging damit forceScrollToEndRef → man landete bei
    // Direktnachrichten mit Verlauf ganz oben. Nach der Konsolidierung auf
    // openConversation muss scrollIntoView aufgerufen werden.
    const scrollSpy = Element.prototype.scrollIntoView as unknown as {
      mock: { calls: unknown[] }
    }
    scrollSpy.mock.calls.length = 0

    renderAsPersona(<ChatPage />, 'spieler', {
      route: '/chat?openUser=42',
      mocks: [
        // GET first, then any-method fallback für den POST /chat/conversations
        // aus dem openUser-Deep-Link (der Mock-Helper kennt kein 'post', deshalb
        // GET registrieren + 'any' als Fallback für alle anderen Methoden).
        { url: '/chat/conversations', data: [] },
        { method: 'any', url: '/chat/conversations', data: CONV },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/55\/messages/, data: MESSAGES },
        {
          method: 'any',
          url: /\/chat\/conversations\/55\/read/,
          data: {},
        },
      ],
    })

    await flushAsync()
    await flushAsync()

    // Konversation ist geöffnet (Header sichtbar) und mindestens eine
    // Nachricht wird gerendert — beweist, dass openConversation den Pfad
    // durchlaufen hat.
    expect(screen.getByText('Alt-1')).toBeInTheDocument()

    // Sticky-Guard wurde übersteuert (forceScrollToEndRef=true), also fährt
    // der Auto-Scroll ans Ende. Vor dem Fix passierte das NICHT, weil der
    // openUser-Handler direkt setActiveConv+loadMessages aufrief und den
    // Force-Flag umging.
    expect(scrollSpy.mock.calls.length).toBeGreaterThan(0)
  })
})
