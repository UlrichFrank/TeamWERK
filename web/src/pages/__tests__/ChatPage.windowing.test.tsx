import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen, act, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
let chatEventDispatch: ((event: string) => void) | null = null
vi.mock('../../hooks/useChatEvents', () => ({
  useChatEvents: (cb: (event: string) => void) => {
    chatEventDispatch = cb
  },
}))

// Chat-Scroll-Layout: Container groß genug, dass der Fake-Content (200
// „Bubbles") deutlich weiter scrollen kann als der Viewport.
const VIEWPORT = 300
const CONTENT_HEIGHT = 20000
const scrollBox = { value: 0 }

beforeAll(() => {
  // scrollIntoView existiert in jsdom nicht.
  Element.prototype.scrollIntoView = vi.fn()
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
  id: 7,
  type: 'group' as const,
  name: 'Mannschaft',
  createdBy: 99,
  unreadCount: 0,
  lastMessage: null,
  members: [{ id: 1, name: 'Ich' }, { id: 2, name: 'Andere' }],
}

const MESSAGES = Array.from({ length: 200 }, (_, i) => ({
  id: i,
  senderId: 2,
  senderName: 'Andere',
  preview: `Nachricht-${i}`,
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

describe('ChatPage — Scroll-Verhalten', () => {
  test('rendert alle Nachrichten (kein Windowing im Chat)', async () => {
    // Windowing im Chat ist bewusst deaktiviert: variable Bubble-Höhen mit
    // fixer estimatedRowHeight lassen den Sichtbereich beim Scrollen springen
    // (siehe Fix zu „Position springt ständig"). Deshalb müssen alle geladenen
    // Nachrichten im DOM landen — sowohl früh als auch spät in der Liste.
    scrollBox.value = 0
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [CONV] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: MESSAGES },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
      ],
    })
    await flushAsync()

    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    expect(screen.getByText('Nachricht-0')).toBeInTheDocument()
    expect(screen.getByText('Nachricht-180')).toBeInTheDocument()
    expect(screen.getByText('Nachricht-199')).toBeInTheDocument()
  })

  test('hochgescrollte Nutzer werden bei State-Änderungen nicht ans Ende gerissen', async () => {
    // Sticky-to-Bottom: Auto-Scroll fährt nur ans Ende, wenn der Nutzer eh
    // dort steht. Wer hochscrollt, um alte Nachrichten zu lesen, darf durch
    // eingehende SSE-Events / Reactions-Reloads nicht zurückgeschleudert
    // werden.
    scrollBox.value = 0
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [CONV] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: MESSAGES },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
      ],
    })
    await flushAsync()

    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    const container = document.querySelector(
      '[data-windowed-scroll]',
    ) as HTMLElement
    expect(container).not.toBeNull()

    // Nutzer scrollt weit nach oben (Abstand zum Ende deutlich > 100 px).
    act(() => {
      scrollBox.value = 0
      fireEvent.scroll(container)
    })

    const scrollIntoView = Element.prototype.scrollIntoView as unknown as {
      mock: { calls: unknown[] }
    }
    const callsBefore = scrollIntoView.mock.calls.length

    // SSE liefert eine neue Nachricht → messages ändert sich. Das darf den
    // hochgescrollten Nutzer NICHT ans Ende scrollen.
    expect(chatEventDispatch).not.toBeNull()
    act(() => {
      chatEventDispatch!('chat:new-message:7')
    })
    await flushAsync()

    expect(scrollIntoView.mock.calls.length).toBe(callsBefore)
  })
})
