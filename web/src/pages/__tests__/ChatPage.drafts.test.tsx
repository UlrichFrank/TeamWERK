import { describe, test, expect, vi, beforeAll, beforeEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

beforeAll(() => {
  // scrollIntoView existiert in jsdom nicht (ChatPage nutzt es nach dem Öffnen).
  Element.prototype.scrollIntoView = vi.fn()
})

beforeEach(() => {
  localStorage.clear()
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

const CONV2 = { ...CONV, id: 8, name: 'Vorstand' }

const mocks = [
  { url: '/chat/conversations', data: [CONV, CONV2] },
  { url: '/chat/broadcasts', data: [] },
  { url: /\/chat\/conversations\/\d+\/messages/, data: [] },
  { method: 'any' as const, url: /\/chat\/conversations\/\d+\/read/, data: {} },
]

// Persona-User hat id 1 (renderAsPersona) → Draft-Store ist pro Nutzer gescoped.
const DRAFT_KEY = 'teamwerk:chat-drafts:1'

describe('ChatPage — Entwurfs-Persistenz', () => {
  test('tippt man einen Entwurf, landet er pro Konversation in localStorage', async () => {
    renderAsPersona(<ChatPage />, 'spieler', { mocks })
    await flushAsync()

    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    const input = screen.getByPlaceholderText('Nachricht schreiben…')
    fireEvent.change(input, { target: { value: 'halb fertiger Text' } })
    await flushAsync()

    expect(JSON.parse(localStorage.getItem(DRAFT_KEY)!)).toEqual({
      '7': 'halb fertiger Text',
    })
  })

  test('nach Remount (Reload/App-Neustart) wird der Entwurf wiederhergestellt', async () => {
    localStorage.setItem(DRAFT_KEY, JSON.stringify({ '7': 'überlebt den Reload' }))

    renderAsPersona(<ChatPage />, 'spieler', { mocks })
    await flushAsync()

    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    const input = screen.getByPlaceholderText('Nachricht schreiben…') as HTMLTextAreaElement
    expect(input.value).toBe('überlebt den Reload')
  })

  test('leert man das Feld, wird der Entwurf aus dem Store entfernt', async () => {
    localStorage.setItem(DRAFT_KEY, JSON.stringify({ '7': 'wird gelöscht' }))

    renderAsPersona(<ChatPage />, 'spieler', { mocks })
    await flushAsync()

    fireEvent.click(screen.getByText('Mannschaft'))
    await flushAsync()

    const input = screen.getByPlaceholderText('Nachricht schreiben…')
    fireEvent.change(input, { target: { value: '' } })
    await flushAsync()

    // Letzter Entwurf entfernt → gesamter Store-Key ebenfalls (map leer).
    expect(localStorage.getItem(DRAFT_KEY)).toBeNull()
  })
})
