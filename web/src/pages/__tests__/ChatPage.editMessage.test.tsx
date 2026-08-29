import { describe, test, expect, vi, beforeAll, afterEach } from 'vitest'
import { screen, act, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { api } from '../../lib/api'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

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
afterEach(() => vi.restoreAllMocks())

const CONV = {
  id: 7, type: 'group' as const, name: 'Mannschaft', createdBy: 99,
  unreadCount: 0, lastMessage: null,
  members: [{ id: 1, name: 'Ich' }, { id: 2, name: 'Andere' }],
}

// Länger als messagePreviewLen (280) — nur solche Nachrichten landen überhaupt
// im Volltext-Cache, den der Bug stale hielt.
const LONG_OLD = 'ALT-' + 'x'.repeat(400)
const NEW_TEXT = 'NEU-und-kurz'

function msg(preview: string, truncated: boolean, editedAt: string | null) {
  return {
    id: 1, senderId: 1, senderName: 'Ich', preview, truncated, editedAt,
    sentAt: '2026-06-28T10:00:00Z', replyToId: null, replyToBody: null,
    replyToSenderName: null, deletedAt: null, isSystem: false,
    reactions: [], readCount: 0, readTotal: 1, read: false,
  }
}

describe('ChatPage — Bearbeiten einer gekürzten Nachricht', () => {
  test('nach dem Speichern zeigt die Blase den neuen Text, nicht den gecachten Volltext', async () => {
    // Mutable Referenz: der Mock-Adapter serialisiert beim Request, der
    // PUT-Spy schreibt vorher den neuen Serverstand hinein.
    const list: unknown[] = [msg(LONG_OLD.slice(0, 280), true, null)]

    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        { url: '/chat/conversations', data: [CONV] },
        { url: '/chat/broadcasts', data: [] },
        { url: /\/chat\/conversations\/7\/messages/, data: list },
        { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
        { url: '/chat/messages/1', data: { body: LONG_OLD } },
      ],
    })
    const putSpy = vi.spyOn(api, 'put').mockImplementation(async () => {
      list.length = 0
      list.push(msg(NEW_TEXT, false, '2026-06-28T10:05:00Z'))
      return { status: 204, data: null } as never
    })

    await flushAsync()
    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    // Kontextmenü auf der eigenen Nachricht → Bearbeiten. startEdit holt dabei
    // den Volltext nach und legt ihn im Cache ab (genau der stale Eintrag).
    fireEvent.contextMenu(screen.getByText(LONG_OLD.slice(0, 280), { exact: false }))
    await flushAsync()
    fireEvent.click(screen.getByText('Bearbeiten'))
    await flushAsync()

    fireEvent.change(screen.getByPlaceholderText(/Nachricht/i), {
      target: { value: NEW_TEXT },
    })
    await act(async () => {
      fireEvent.click(screen.getByLabelText('Speichern'))
      await new Promise((r) => setTimeout(r, 0))
    })
    await flushAsync()

    expect(putSpy).toHaveBeenCalledWith('/chat/messages/1', { body: NEW_TEXT })
    expect(screen.getByText(NEW_TEXT)).toBeInTheDocument()
    expect(screen.queryByText(LONG_OLD)).toBeNull()
  })
})
