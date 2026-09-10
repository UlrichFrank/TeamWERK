import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen, act, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
let chatEventDispatch: ((event: string) => void) | null = null
vi.mock('../../hooks/useChatEvents', () => ({
  useChatEvents: (cb: (event: string) => void) => {
    chatEventDispatch = cb
  },
}))

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

const CONV_GROUP = {
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

const CONV_DIRECT = {
  id: 8,
  type: 'direct' as const,
  name: null,
  createdBy: 1,
  unreadCount: 0,
  lastMessage: null,
  members: [
    { id: 1, name: 'Ich' },
    { id: 2, name: 'Andere' },
  ],
}

const POLL_INITIAL = {
  allowMultiple: false,
  closedAt: null,
  voterCount: 1,
  options: [
    { id: 11, label: 'Pizza', count: 1, voted: false, voters: [{ id: 2, name: 'Andere' }] },
    { id: 12, label: 'Nudeln', count: 0, voted: false, voters: [] },
  ],
}

const POLL_UPDATED = {
  allowMultiple: false,
  closedAt: null,
  voterCount: 2,
  options: [
    { id: 11, label: 'Pizza', count: 1, voted: false, voters: [{ id: 2, name: 'Andere' }] },
    { id: 12, label: 'Nudeln', count: 1, voted: false, voters: [{ id: 3, name: 'Dritte' }] },
  ],
}

function pollMsg(id: number, poll: unknown) {
  return {
    id,
    senderId: 2,
    senderName: 'Andere',
    preview: 'Pizza oder Nudeln?',
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
    poll,
  }
}

describe('ChatPage — Umfragen: Live-Update', () => {
  test('chat:poll-updated aktualisiert nur die betroffene Nachricht, kein Listen-Reload', async () => {
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [CONV_GROUP] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: [pollMsg(1, POLL_INITIAL)] },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
        { url: /\/chat\/messages\/1\/poll/, data: POLL_UPDATED },
      ],
    })
    await flushAsync()
    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    expect(screen.getByText('Pizza oder Nudeln?')).toBeInTheDocument()
    expect(screen.getByText(/1 Stimme · Stimmen anzeigen/)).toBeInTheDocument()

    const messagesCallsBefore = getApiMock().history.get.filter((c) =>
      /\/chat\/conversations\/7\/messages/.test(c.url ?? ''),
    ).length

    expect(chatEventDispatch).not.toBeNull()
    act(() => {
      chatEventDispatch!('chat:poll-updated:7:1')
    })
    await flushAsync()

    // Neue Stimme sichtbar — nachgeladen über GET .../poll, nicht über einen
    // Voll-Reload der Nachrichtenliste.
    expect(screen.getByText(/2 Stimmen · Stimmen anzeigen/)).toBeInTheDocument()

    const messagesCallsAfter = getApiMock().history.get.filter((c) =>
      /\/chat\/conversations\/7\/messages/.test(c.url ?? ''),
    ).length
    expect(messagesCallsAfter).toBe(messagesCallsBefore)

    const pollCalls = getApiMock().history.get.filter((c) =>
      /\/chat\/messages\/1\/poll/.test(c.url ?? ''),
    ).length
    expect(pollCalls).toBeGreaterThanOrEqual(1)
  })

  test('chat:poll-updated einer nicht offenen Konversation wird ignoriert', async () => {
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [CONV_GROUP] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: [pollMsg(1, POLL_INITIAL)] },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
      ],
    })
    await flushAsync()
    // Konversation bewusst NICHT öffnen.

    expect(chatEventDispatch).not.toBeNull()
    act(() => {
      chatEventDispatch!('chat:poll-updated:7:1')
    })
    await flushAsync()

    const pollCalls = getApiMock().history.get.filter((c) =>
      /\/chat\/messages\/1\/poll/.test(c.url ?? ''),
    ).length
    expect(pollCalls).toBe(0)
  })
})

describe('ChatPage — Umfrage-Button im Composer', () => {
  test('fehlt in Direktkonversationen', async () => {
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [CONV_DIRECT] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/8\/messages/, data: [] },
        { method: 'any', url: /\/chat\/conversations\/8\/read/, data: {} },
      ],
    })
    await flushAsync()
    fireEvent.click(screen.getByText('Andere'))
    await flushAsync()

    expect(screen.getByPlaceholderText('Nachricht schreiben…')).toBeInTheDocument()
    expect(screen.queryByLabelText('Umfrage erstellen')).toBeNull()
  })

  test('ist in Gruppenkonversationen vorhanden und öffnet das Erstell-Modal', async () => {
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [CONV_GROUP] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: [] },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
      ],
    })
    await flushAsync()
    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    const pollButton = screen.getByLabelText('Umfrage erstellen')
    fireEvent.click(pollButton)
    await flushAsync()
    expect(screen.getByText('Umfrage erstellen', { selector: 'h2' })).toBeInTheDocument()
  })
})
