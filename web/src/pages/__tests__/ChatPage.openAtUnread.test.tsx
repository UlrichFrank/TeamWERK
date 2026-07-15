import { describe, test, expect, vi, beforeAll, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

// Layout-Mocks für den WindowedRows-Container: der Auto-Scroll-Effekt
// checkt distanceFromBottom aus scrollHeight/scrollTop/clientHeight; wir
// simulieren einen Container mit Content > Viewport.
const VIEWPORT = 300
const CONTENT_HEIGHT = 5000
const scrollBox = { value: 0 }

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
  // jsdom implementiert scrollTo nicht — Polyfill, der scrollTop setzt.
  Element.prototype.scrollTo = function (this: HTMLElement, arg: unknown) {
    const opts = typeof arg === 'object' && arg !== null ? (arg as { top?: number }) : null
    if (opts && typeof opts.top === 'number') this.scrollTop = opts.top
  } as unknown as Element['scrollTo']
  // Synchroner rAF-Mock: der Custom-Smooth-Loop in ChatPage schließt sich
  // damit in einem Tick statt über ~16 Frames verteilt.
  let rafTime = 0
  globalThis.requestAnimationFrame = (cb: FrameRequestCallback) => {
    rafTime += 400
    cb(rafTime)
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

beforeEach(() => {
  scrollBox.value = 0
  ;(Element.prototype.scrollIntoView as unknown as { mock: { calls: unknown[] } })
    .mock.calls.length = 0
})

function makeMessages(count: number) {
  return Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    senderId: 42,
    senderName: 'Andere',
    preview: `Nachricht-${i + 1}`,
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
}

function makeConv(unreadCount: number) {
  return {
    id: 7,
    type: 'group' as const,
    name: 'Mannschaft',
    createdBy: 99,
    unreadCount,
    lastMessage: null,
    members: [
      { id: 1, name: 'Ich' },
      { id: 42, name: 'Andere' },
    ],
  }
}

describe('ChatPage — Öffnen positioniert am ersten Ungelesenen', () => {
  test('unreadCount=3 → Divider "3 ungelesene Nachrichten" wird gerendert und scrollIntoView({block:start}) läuft', async () => {
    const conv = makeConv(3)
    const messages = makeMessages(20)
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [conv] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: messages },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
      ],
    })
    await flushAsync()

    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    // Divider mit korrektem Zähler-Text ist sichtbar.
    expect(screen.getByText('3 ungelesene Nachrichten')).toBeInTheDocument()

    // scrollIntoView wurde mindestens einmal mit block:'start' aufgerufen
    // (der Divider-Effekt; der End-Scroll-Fall benutzt behavior:'smooth').
    const spy = Element.prototype.scrollIntoView as unknown as {
      mock: { calls: Array<unknown[]> }
    }
    const hasBlockStart = spy.mock.calls.some((args) => {
      const opts = args[0] as { block?: string } | undefined
      return opts?.block === 'start'
    })
    expect(hasBlockStart).toBe(true)
  })

  test('unreadCount=0 → KEIN Divider, scrollIntoView({behavior:smooth}) ans Ende (Regression zu bisherigem Verhalten)', async () => {
    const conv = makeConv(0)
    const messages = makeMessages(20)
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [conv] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: messages },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
      ],
    })
    await flushAsync()

    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    // Kein Divider im DOM.
    expect(screen.queryByText(/ungelesene Nachrichten$/)).toBeNull()

    // End-Scroll läuft als rAF-basierter Smooth-Loop; das finale Ziel ist
    // scrollHeight (der Loop endet mit einem harten Snap gegen sub-pixel-Rest).
    await waitFor(() => {
      expect(scrollBox.value).toBe(CONTENT_HEIGHT)
    })
  })

  test('unreadCount > geladene Seite → Chip "N weitere ungelesene älter" sichtbar, kein Divider', async () => {
    const conv = makeConv(150)
    const messages = makeMessages(100)
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [conv] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: messages },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
      ],
    })
    await flushAsync()

    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    // Chip mit korrekter Differenz sichtbar (150 - 100 = 50).
    expect(
      screen.getByText(/50 weitere\s+ungelesene Nachrichten älter/),
    ).toBeInTheDocument()

    // Kein Divider — alle Nachrichten sind ungelesen, Chip übernimmt die
    // visuelle Rolle.
    expect(screen.queryByText(/^\d+ ungelesene Nachrichten$/)).toBeNull()
  })
})
