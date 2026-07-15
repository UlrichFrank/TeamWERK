import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

// Layout-Mocks für den Windowed-Scroll-Container, damit der End-Scroll
// (box.scrollTop = box.scrollHeight) beobachtbar ist.
const VIEWPORT = 300
const CONTENT_HEIGHT = 5000
const scrollBox = { value: 0 }

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
  Element.prototype.scrollTo = function (this: HTMLElement, arg: unknown) {
    const opts = typeof arg === 'object' && arg !== null ? (arg as { top?: number }) : null
    if (opts && typeof opts.top === 'number') this.scrollTop = opts.top
  } as unknown as Element['scrollTo']
  // Synchroner rAF-Mock: der Custom-Smooth-Loop in ChatPage schließt sich
  // damit in einem Tick statt über ~16 Frames verteilt. Ohne diesen Mock
  // sitzt der Loop in jsdom fest (kein echter Frame-Callback).
  let t = 0
  globalThis.requestAnimationFrame = (cb: FrameRequestCallback) => {
    t += 400
    cb(t)
    return 0
  }
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? VIEWPORT : 0
    },
  })
  Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? CONTENT_HEIGHT : 0
    },
  })
  Object.defineProperty(HTMLElement.prototype, 'scrollTop', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? scrollBox.value : 0
    },
    set(v: number) {
      if (this.hasAttribute('data-windowed-scroll')) scrollBox.value = v
    },
  })
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
    // openConversation muss der Container-scrollTop auf scrollHeight
    // gesetzt werden (End-Scroll).
    scrollBox.value = 0

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

    // Sticky-Guard wurde übersteuert, also fährt der Auto-Scroll ans Ende.
    // smoothScrollToBottom läuft als rAF-Loop (~250 ms in jsdom), deshalb
    // waitFor statt sofort assert.
    await waitFor(() => {
      expect(scrollBox.value).toBe(CONTENT_HEIGHT)
    })
  })
})
