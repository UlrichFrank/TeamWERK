import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import TerminePage from './TerminePage'

// kalender-sprung: „Im Kalender öffnen“ auf den Termin-Karten.

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: false },
    hasCapability: () => false,
  }),
}))

const SEASON = { id: 7, name: '2026/27', start_date: '2026-05-01', end_date: '2027-06-30' }

const training = {
  id: 42, series_id: null, title: 'Training', date: '2026-10-14', start_time: '18:00', end_time: '20:00',
  venue: null, note: '', status: 'active', cancel_reason: '', team_id: 1, team_name: 'Team A',
  confirmed_count: 0, declined_count: 0, maybe_count: 0, my_rsvp: null, am_i_participant: false,
  rsvp_default_players: 'none', rsvp_default_extended: 'none', rsvp_require_reason: 0,
}
const game = {
  id: 17, date: '2026-10-11T00:00:00Z', time: '14:15', opponent: 'TSB Gmünd', event_type: 'heim',
  team_names: 'gD', venue: null, note: '', confirmed_count: 0, declined_count: 0, maybe_count: 0,
  am_i_participant: false,
}

function renderPage() {
  mockGet.mockImplementation((raw?: string) => {
    const url = raw ?? ''
    if (url.startsWith('/seasons/active')) return Promise.resolve({ data: SEASON })
    if (url.startsWith('/training-sessions')) return Promise.resolve({ data: [training] })
    if (url.startsWith('/games/my')) return Promise.resolve({ data: [game] })
    return Promise.resolve({ data: [] })
  })
  const router = createMemoryRouter(
    [
      { path: '/termine', element: <TerminePage /> },
      { path: '/kalender', element: <div data-testid="kalender" /> },
      { path: '/termine/*', element: <div data-testid="detail" /> },
    ],
    { initialEntries: ['/termine'] },
  )
  render(<RouterProvider router={router} />)
  return router
}

describe('TerminePage — Im Kalender öffnen', () => {
  beforeEach(() => mockGet.mockReset())

  test('Spiel-Karte springt zu /kalender mit date und focus, nicht zur Detailseite', async () => {
    const user = userEvent.setup()
    const router = renderPage()
    await screen.findByText('Heim: TSB Gmünd')
    const card = document.getElementById('termin-game-17')!
    await user.click(card.querySelector('button[aria-label="Im Kalender öffnen"]')!)

    await screen.findByTestId('kalender')
    expect(router.state.location.pathname + router.state.location.search).toBe('/kalender?date=2026-10-11&focus=game-17')
    expect(screen.queryByTestId('detail')).toBeNull()
  })

  test('Trainings-Karte springt zu /kalender mit focus=training-<id>', async () => {
    const user = userEvent.setup()
    const router = renderPage()
    await screen.findByText('Heim: TSB Gmünd')
    const card = document.getElementById('termin-training-42')!
    await user.click(card.querySelector('button[aria-label="Im Kalender öffnen"]')!)

    await screen.findByTestId('kalender')
    expect(router.state.location.search).toBe('?date=2026-10-14&focus=training-42')
  })
})
