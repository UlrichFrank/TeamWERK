import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
})

function conv(id: number, unreadCount: number) {
  return {
    id,
    type: 'direct' as const,
    name: `Konv ${id}`,
    createdBy: 1,
    unreadCount,
    lastMessage: { body: 'Hallo', sentAt: '2026-08-16T10:00:00Z' },
    members: [
      { id: 1, name: 'Ich' },
      { id: 40 + id, name: `Partner ${id}` },
    ],
  }
}

function broadcast(id: number, isRead: boolean, isSent: boolean) {
  return {
    id,
    senderName: 'Vorstand',
    body: `Mitteilung ${id}`,
    sentAt: '2026-08-16T09:00:00Z',
    isRead,
    isSent,
    editedAt: null,
    mediaId: null,
    mediaUrl: null,
  }
}

async function renderChat(conversations: unknown[], broadcasts: unknown[]) {
  renderAsPersona(<ChatPage />, 'spieler', {
    route: '/chat',
    mocks: [
      { url: '/chat/conversations', data: conversations },
      { url: '/chat/broadcasts', data: broadcasts },
    ],
  })
  await flushAsync()
}

const chatsTab = () => screen.getByRole('button', { name: /^Chats/ })
const mitteilungenTab = () => screen.getByRole('button', { name: /^Mitteilungen/ })
const heading = () => screen.getByRole('heading', { level: 1 })

describe('ChatPage — Ungelesen-Badges an den Tabs', () => {
  test('Chats-Tab zeigt nur den Konversations-Anteil, nicht die Gesamtsumme', async () => {
    await renderChat(
      [conv(1, 2), conv(2, 1)],
      [broadcast(10, false, false), broadcast(11, false, false)],
    )

    // Die beiden Tab-Badges partitionieren die Zahl an der Überschrift:
    // 3 (Konversationen) + 2 (Mitteilungen) = 5. Zeigte der Chats-Tab
    // `total`, stünde dort fälschlich 5 und die Summe wäre 7.
    expect(heading()).toHaveTextContent('5')
    expect(chatsTab()).toHaveTextContent('3')
    expect(mitteilungenTab()).toHaveTextContent('2')
  })

  test('ohne ungelesene Konversationen trägt der Chats-Tab keinen Badge', async () => {
    await renderChat([conv(1, 0)], [broadcast(10, false, false)])

    expect(chatsTab()).not.toHaveTextContent('0')
    expect(chatsTab().textContent?.trim()).toBe('Chats')
    expect(mitteilungenTab()).toHaveTextContent('1')
  })

  test('eigene Mitteilung zählt an keinem der beiden Tabs mit', async () => {
    await renderChat([conv(1, 4)], [broadcast(10, false, true)])

    expect(chatsTab()).toHaveTextContent('4')
    expect(mitteilungenTab().textContent?.trim()).toBe('Mitteilungen')
    expect(heading()).toHaveTextContent('4')
  })

  test('ganz ohne Ungelesenes trägt keine der drei Stellen eine Zahl', async () => {
    await renderChat([conv(1, 0)], [broadcast(10, true, false)])

    expect(chatsTab().textContent?.trim()).toBe('Chats')
    expect(mitteilungenTab().textContent?.trim()).toBe('Mitteilungen')
    expect(heading().textContent?.trim()).toBe('Nachrichten')
  })
})
