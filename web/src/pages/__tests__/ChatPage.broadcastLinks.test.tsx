/**
 * Broadcast-Detailansicht rendert URLs als Hyperlinks — gleicher renderWithLinks-Pfad
 * wie die Chat-Bubbles (zuvor wurde der Broadcast-Body als reiner Text interpoliert).
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

describe('ChatPage — Broadcast-Detail rendert URLs als Hyperlinks', () => {
  test('http(s)-URL im Broadcast-Body wird zu klickbarem <a target="_blank">', async () => {
    renderAsPersona(<ChatPage />, 'spieler', {
      mocks: [
        {
          url: '/chat/broadcasts',
          data: [
            {
              id: 42,
              senderName: 'Coach',
              body: 'Infos unter https://team-stuttgart.org/info',
              sentAt: '2026-06-28T10:00:00Z',
              isRead: false,
              isSent: false,
              editedAt: null,
            },
          ],
        },
      ],
    })
    await flushAsync()

    fireEvent.click(screen.getByText('Mitteilungen'))
    await flushAsync()

    // Broadcast in der Liste öffnen (Klick auf Absendername).
    fireEvent.click(screen.getByText('Coach'))
    await flushAsync()

    const link = screen.getByRole('link', { name: /team-stuttgart\.org\/info/ })
    expect(link).toHaveAttribute('href', 'https://team-stuttgart.org/info')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })
})
