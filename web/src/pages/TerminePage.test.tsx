import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import TerminePage from './TerminePage'

// Minimal Session-Payload für /api/training-sessions
function trainingSession(overrides: Record<string, unknown>) {
  return {
    id: 100,
    series_id: null,
    title: 'Training',
    date: '2026-05-01',
    start_time: '18:00',
    end_time: '20:00',
    venue: null,
    note: '',
    status: 'active',
    cancel_reason: '',
    team_id: 1,
    team_name: 'Team A',
    confirmed_count: 0,
    declined_count: 1,
    maybe_count: 0,
    my_rsvp: 'declined',
    rsvp_default_players: 'none',
    rsvp_default_extended: 'none',
    rsvp_require_reason: 1,
    ...overrides,
  }
}

const mockGet = vi.fn()
const authState = { is_parent: false }
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: authState.is_parent },
    hasCapability: () => false,
  }),
}))

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/termine']}>
      <TerminePage />
    </MemoryRouter>,
  )
}

// mockGet wird für mehrere Routen aufgerufen; wir routen nach URL.
function seedRoutes(sessions: unknown[], games: unknown[] = []) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/training-sessions')) return Promise.resolve({ data: sessions })
    if (url.startsWith('/games/my')) return Promise.resolve({ data: games })
    if (url.startsWith('/teams')) return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
}

describe('TerminePage — eigener Absagegrund', () => {
  beforeEach(() => {
    mockGet.mockReset()
    authState.is_parent = false
  })

  test('zeigt my_reason unter den RSVP-Buttons, wenn im Payload gesetzt', async () => {
    seedRoutes([trainingSession({ my_reason: 'Klavierstunde' })])
    renderPage()
    await waitFor(() => expect(screen.getByText('Klavierstunde')).toBeTruthy())
  })

  test('rendert keine Grund-Zeile, wenn my_reason nicht gesetzt ist', async () => {
    seedRoutes([trainingSession({})])
    renderPage()
    // Warten bis Card gerendert (Team-Name ist ein zuverlässiger Marker)
    await waitFor(() => expect(screen.getByText(/Training/)).toBeTruthy())
    expect(screen.queryByText('Klavierstunde')).toBeNull()
  })

  test('zeigt Kind-Reason im children_rsvp-Payload für Elternteil', async () => {
    authState.is_parent = true
    seedRoutes([
      trainingSession({
        my_rsvp: null,
        children_rsvp: [
          { member_id: 42, name: 'Anna', rsvp: 'declined', reason: 'Krank' },
        ],
      }),
    ])
    renderPage()
    await waitFor(() => expect(screen.getByText('Krank')).toBeTruthy())
  })
})
