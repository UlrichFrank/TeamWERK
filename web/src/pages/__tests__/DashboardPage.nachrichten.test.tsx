import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import DashboardPage from '../DashboardPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

// Dashboard abonniert Live-Updates; im Test neutralisieren.
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

const DASHBOARD = {
  currentSeason: null,
  meineTermine: [],
  meineDienste: null,
  carpoolingConfirmed: [],
  carpoolingOpenGroups: [],
}

const CONV_UNREAD = [{
  id: 1, type: 'direct', name: null, unreadCount: 2,
  lastMessage: { body: 'Bis morgen', sentAt: '2026-07-10T10:00:00Z' },
  members: [{ id: 1, name: 'Ich' }, { id: 2, name: 'Anna Trainer' }],
}]
const BC_UNREAD = [{
  id: 5, senderName: 'Bob Vorstand', body: 'Hallenschluss um 22 Uhr',
  sentAt: '2026-07-11T09:00:00Z', isRead: false, isSent: false,
}]

describe('DashboardPage — Nachrichten-Section', () => {
  test('zeigt ungelesene Konversation (Partnername) und Mitteilung (Absender) + Zum-Chat-Link', async () => {
    renderAsPersona(<DashboardPage />, 'spieler', {
      mocks: [
        { url: '/dashboard', data: DASHBOARD },
        { url: '/chat/conversations', data: CONV_UNREAD },
        { url: '/chat/broadcasts', data: BC_UNREAD },
      ],
    })
    await flushAsync()

    expect(screen.getByText('Nachrichten')).toBeInTheDocument()
    // Direkt-Chat ohne Namen → Partnername (Mitglied ≠ eigene id=1)
    expect(screen.getByText('Anna Trainer')).toBeInTheDocument()
    expect(screen.getByText('Bob Vorstand')).toBeInTheDocument()
    expect(screen.getAllByText('Zum Chat').length).toBeGreaterThan(0)
  })

  test('leerer Zustand ohne Ungelesenes', async () => {
    renderAsPersona(<DashboardPage />, 'spieler', {
      mocks: [
        { url: '/dashboard', data: DASHBOARD },
        { url: '/chat/conversations', data: [{ id: 1, type: 'group', name: 'Team', unreadCount: 0, lastMessage: null, members: [] }] },
        { url: '/chat/broadcasts', data: [{ id: 5, senderName: 'Bob', body: 'alt', sentAt: '2026-07-01T09:00:00Z', isRead: true, isSent: false }] },
      ],
    })
    await flushAsync()

    expect(screen.getByText('Keine ungelesenen Nachrichten.')).toBeInTheDocument()
  })
})
