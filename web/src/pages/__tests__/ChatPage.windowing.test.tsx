import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen, act, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

const VIEWPORT = 300
const ROW_HEIGHT = 64
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
  // Layout für den Chat-Scroll-Container (data-windowed-scroll) simulieren.
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? VIEWPORT : 0
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
  // Listen-Contract (④): Preview + truncated-Flag statt Volltext. Kurze
  // Nachrichten sind nicht gekürzt → preview = Volltext, truncated=false.
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

describe('ChatPage — Windowing der Nachrichten-Historie', () => {
  test('rendert nur sichtbare Nachrichten; Scrollen tauscht sie aus', async () => {
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

    // Konversation öffnen.
    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    // Frühe Nachricht im DOM, weit hinten liegende nicht.
    expect(screen.getByText('Nachricht-0')).toBeInTheDocument()
    expect(screen.queryByText('Nachricht-180')).toBeNull()

    // Ans Ende scrollen: Nachricht 180 liegt bei 180*64 = 11520px.
    const container = document.querySelector('[data-windowed-scroll]') as HTMLElement
    expect(container).not.toBeNull()
    act(() => {
      scrollBox.value = 180 * ROW_HEIGHT
      fireEvent.scroll(container)
    })

    expect(screen.getByText('Nachricht-180')).toBeInTheDocument()
    expect(screen.queryByText('Nachricht-0')).toBeNull()
  })
})
