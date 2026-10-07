import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import DutyPage from './DutyPage'

// kalender-sprung: „Im Kalender öffnen“ im Kopf der Dienst-Blöcke.

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', clubFunctions: [] },
    hasCapability: () => false,
  }),
}))

function group(overrides: Record<string, unknown>) {
  return {
    game_id: 17, team_ids: [1], team_names: ['Team A'], date: '2026-10-11T00:00:00Z', event_time: '14:15',
    opponent: 'Gegner', event_type: 'heim', label: null, past: false,
    slots: [{
      id: 555, duty_type: 'Kasse', duty_type_id: 42, has_instruction: false, event_time: '14:15',
      hours_value: 1, slots_total: 2, vacancies: 1, claimed_by_me: false, assignees: [],
    }],
    ...overrides,
  }
}

function renderPage(groups: unknown[]) {
  mockGet.mockImplementation((raw?: string) => {
    const url = raw ?? ''
    if (url.startsWith('/duty-board')) return Promise.resolve({ data: groups })
    return Promise.resolve({ data: [] })
  })
  const router = createMemoryRouter(
    [
      { path: '/dienste', element: <DutyPage /> },
      { path: '/kalender', element: <div data-testid="kalender" /> },
    ],
    { initialEntries: ['/dienste'] },
  )
  render(<RouterProvider router={router} />)
  return router
}

describe('DutyPage — Im Kalender öffnen', () => {
  beforeEach(() => mockGet.mockReset())

  test('Block eines Spiels springt zu /kalender mit date und focus', async () => {
    const user = userEvent.setup()
    const router = renderPage([group({})])
    await user.click(await screen.findByRole('button', { name: 'Im Kalender öffnen' }))

    await screen.findByTestId('kalender')
    expect(router.state.location.search).toBe('?date=2026-10-11&focus=game-17')
  })

  test('Block ohne Termin hat keine Aktion', async () => {
    renderPage([group({ game_id: null, label: 'Hallenputz', opponent: null, event_type: null })])
    await screen.findByText(/Hallenputz/)
    expect(screen.queryByRole('button', { name: 'Im Kalender öffnen' })).toBeNull()
  })
})
