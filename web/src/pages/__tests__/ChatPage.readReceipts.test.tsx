import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen, act, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import MessageReadsModal from '../../components/MessageReadsModal'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
let chatEventDispatch: ((event: string) => void) | null = null
vi.mock('../../hooks/useChatEvents', () => ({
  useChatEvents: (cb: (event: string) => void) => {
    chatEventDispatch = cb
  },
}))

// jsdom-Shims für den Chat-Scroll-Container (wie in ChatPage.windowing.test.tsx).
beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
  Element.prototype.scrollTo = vi.fn() as unknown as Element['scrollTo']
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
})

const CONV = {
  id: 7,
  type: 'group' as const,
  name: 'Mannschaft',
  createdBy: 99,
  unreadCount: 0,
  lastMessage: null,
  members: [
    { id: 1, name: 'Ich' },
    { id: 2, name: 'Andere' },
  ],
}

// Persona 'spieler' → eingeloggter User hat id 1; senderId:1 = eigene Nachricht.
function ownMsg(id: number, extra: Record<string, unknown>) {
  return {
    id,
    senderId: 1,
    senderName: 'Ich',
    preview: `Nachricht-${id}`,
    truncated: false,
    sentAt: '2026-06-28T10:00:00Z',
    replyToId: null,
    replyToBody: null,
    replyToSenderName: null,
    editedAt: null,
    deletedAt: null,
    isSystem: false,
    reactions: [],
    readCount: 0,
    readTotal: 1,
    read: false,
    ...extra,
  }
}

async function openConv(messages: unknown[]) {
  renderAsPersona(<ChatPage />, 'spieler', {
    mocks: [
      { url: '/chat/conversations', data: [CONV] },
      { url: '/chat/broadcasts', data: [] },
      { url: /\/chat\/conversations\/7\/messages/, data: messages },
      { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
    ],
  })
  await flushAsync()
  fireEvent.click(screen.getByText('Mannschaft'))
  await flushAsync()
}

describe('ChatPage — Read-Receipts', () => {
  test('5.1 eigene gelesene Nachricht zeigt CheckCheck (gelesen)', async () => {
    await openConv([ownMsg(1, { read: true, readCount: 1, readTotal: 1 })])
    expect(screen.getByLabelText(/gelesen/)).toBeInTheDocument()
    expect(screen.queryByLabelText('gesendet')).toBeNull()
  })

  test('5.2 Gruppen-Aggregat zeigt N/M gelesen', async () => {
    await openConv([ownMsg(1, { read: true, readCount: 3, readTotal: 8 })])
    expect(screen.getByText('3/8')).toBeInTheDocument()
  })

  test('5.4 SSE chat:read-receipt markiert Nachrichten ≤ upTo als gelesen', async () => {
    const msgs = Array.from({ length: 10 }, (_, i) => ownMsg(i + 1, {}))
    await openConv(msgs)
    // Zu Beginn: alle 10 „gesendet", keine „gelesen".
    expect(screen.queryAllByLabelText(/gelesen/)).toHaveLength(0)
    expect(screen.queryAllByLabelText('gesendet')).toHaveLength(10)

    // Reader (id 2) liest bis Nachricht 5.
    expect(chatEventDispatch).not.toBeNull()
    act(() => {
      chatEventDispatch!('chat:read-receipt:7:2:5')
    })
    await flushAsync()

    expect(screen.queryAllByLabelText(/gelesen/)).toHaveLength(5)
    expect(screen.queryAllByLabelText('gesendet')).toHaveLength(5)
  })
})

describe('MessageReadsModal', () => {
  test('5.3 rendert Leser-Liste mit Uhrzeit', async () => {
    renderAsPersona(<MessageReadsModal messageId={5} onClose={() => {}} />, 'spieler', {
      mocks: [
        {
          url: /\/chat\/messages\/5\/reads/,
          data: [
            { userId: 2, name: 'Anna Beispiel', readAt: '2026-06-28T10:05:00Z' },
            { userId: 3, name: 'Ben Muster', readAt: '2026-06-28T10:09:00Z' },
          ],
        },
      ],
    })
    await flushAsync()
    expect(screen.getByText('Gelesen von')).toBeInTheDocument()
    expect(screen.getByText('Anna Beispiel')).toBeInTheDocument()
    expect(screen.getByText('Ben Muster')).toBeInTheDocument()
    // readAt als HH:MM (lokale Zeit; Muster statt fixem Wert wegen TZ-Abhängigkeit).
    expect(screen.getAllByText(/^\d{2}:\d{2}$/).length).toBeGreaterThanOrEqual(2)
  })
})
