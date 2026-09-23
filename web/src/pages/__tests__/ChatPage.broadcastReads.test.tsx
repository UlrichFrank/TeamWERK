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

function broadcast(id: number, extra: Record<string, unknown>) {
  return {
    id,
    senderName: 'Vorstand',
    body: `Mitteilung ${id}`,
    sentAt: '2026-09-01T09:00:00Z',
    isRead: true,
    isSent: false,
    editedAt: null,
    mediaId: null,
    mediaUrl: null,
    ...extra,
  }
}

async function openBroadcast(bc: ReturnType<typeof broadcast>) {
  renderAsPersona(<ChatPage />, 'vorstand', {
    route: '/chat?tab=broadcasts',
    mocks: [
      { url: '/chat/conversations', data: [] },
      { url: '/chat/broadcasts', data: [bc] },
      {
        url: /\/chat\/broadcasts\/10\/reads/,
        data: [{ userId: 2, name: 'Anna Beispiel', readAt: '2026-09-01T10:05:00Z' }],
      },
    ],
  })
  await flushAsync()
  fireEvent.click(screen.getByText(bc.body))
  await flushAsync()
}

describe('ChatPage — Lesebestätigung für Mitteilungen', () => {
  test('eigene Mitteilung zeigt „3 / 10 gelesen"', async () => {
    await openBroadcast(broadcast(10, { isSent: true, readCount: 3, readTotal: 10 }))
    expect(screen.getByRole('button', { name: '3 / 10 gelesen' })).toBeInTheDocument()
  })

  test('fremde Mitteilung zeigt keinen Lese-Zustand', async () => {
    await openBroadcast(broadcast(10, {}))
    expect(screen.queryByText(/gelesen$/)).not.toBeInTheDocument()
  })

  test('SSE chat:broadcast-read erhöht die Anzeige um eins', async () => {
    await openBroadcast(broadcast(10, { isSent: true, readCount: 3, readTotal: 10 }))
    act(() => {
      chatEventDispatch!('chat:broadcast-read:10')
    })
    await flushAsync()
    expect(screen.getByRole('button', { name: '4 / 10 gelesen' })).toBeInTheDocument()
  })

  test('Event für eine andere Mitteilung lässt die Anzeige unverändert', async () => {
    await openBroadcast(broadcast(10, { isSent: true, readCount: 3, readTotal: 10 }))
    act(() => {
      chatEventDispatch!('chat:broadcast-read:99')
    })
    await flushAsync()
    expect(screen.getByRole('button', { name: '3 / 10 gelesen' })).toBeInTheDocument()
  })

  test('Klick öffnet die Leserliste aus /chat/broadcasts/{id}/reads', async () => {
    await openBroadcast(broadcast(10, { isSent: true, readCount: 1, readTotal: 10 }))
    fireEvent.click(screen.getByRole('button', { name: '1 / 10 gelesen' }))
    await flushAsync()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Anna Beispiel')).toBeInTheDocument()
  })
})
