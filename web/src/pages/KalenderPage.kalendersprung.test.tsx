import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import KalenderPage from './KalenderPage'

// kalender-sprung: /kalender?date=…&focus=<game|training>-<id> öffnet den Termin.

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), put: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../hooks/useCompactHeader', () => ({ useCompactHeader: () => false }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: false, clubFunctions: [] },
    hasCapability: () => false,
    logout: vi.fn(),
  }),
}))

function game(overrides: Record<string, unknown> = {}) {
  return {
    id: 17, date: '2026-10-11T00:00:00Z', time: '14:15', opponent: 'TSB Gmünd', event_type: 'auswärts',
    teams: [{ id: 1, name: 'gD' }], slot_count: 0, filled_count: 0, total_count: 0,
    confirmed_count: 0, declined_count: 0, maybe_count: 0, note: '',
    ...overrides,
  }
}

const training = {
  id: 42, title: 'Athletik', date: '2026-10-14', start_time: '18:00', end_time: '20:00', status: 'active',
  confirmed_count: 0, declined_count: 0, maybe_count: 0, my_rsvp: null, team_id: 0, kader_id: 3,
  season_id: 7, note: '', team_name: 'Athletikgruppe',
}

type Routes = { games?: unknown[]; trainings?: unknown[]; detail?: (url: string) => Promise<unknown> }

function seed({ games = [], trainings = [], detail }: Routes) {
  mockGet.mockImplementation((raw?: string) => {
    const url = raw ?? ''
    if (/^\/games\/\d+$/.test(url) || /^\/training-sessions\/\d+$/.test(url)) {
      return detail ? detail(url) : Promise.reject({ response: { status: 404 } })
    }
    if (url.startsWith('/games')) return Promise.resolve({ data: { items: games, total: games.length } })
    if (url.startsWith('/training-sessions')) return Promise.resolve({ data: { items: trainings, total: trainings.length } })
    return Promise.resolve({ data: [] })
  })
}

function renderAt(entry: string) {
  const router = createMemoryRouter([{ path: '/kalender', element: <KalenderPage /> }], { initialEntries: [entry] })
  render(<RouterProvider router={router} />)
  return router
}

describe('KalenderPage — Fokus-Deep-Link', () => {
  beforeEach(() => mockGet.mockReset())

  test('öffnet den Dialog eines Spiels, hebt die Kachel hervor und entfernt focus aus der URL', async () => {
    seed({ games: [game()] })
    const router = renderAt('/kalender?date=2026-10-11&focus=game-17')

    expect(await screen.findByRole('dialog')).toBeTruthy()
    expect(screen.getByText('Auswärtsspiel')).toBeTruthy()
    expect(screen.getByText('Sonntag, 11. Oktober 2026')).toBeTruthy()
    expect(document.getElementById('kalender-game-17')!.className).toContain('ring-brand-yellow')
    await waitFor(() => expect(router.state.location.search).toBe('?date=2026-10-11'))
  })

  test('Schließen öffnet den Dialog nicht erneut', async () => {
    const user = userEvent.setup()
    seed({ games: [game()] })
    renderAt('/kalender?date=2026-10-11&focus=game-17')

    await screen.findByRole('dialog')
    await user.click(screen.getAllByRole('button', { name: 'Schließen' })[0])
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(screen.getByText(/Oktober 2026/)).toBeTruthy()
  })

  test('öffnet den Dialog eines Trainings', async () => {
    seed({ trainings: [training] })
    renderAt('/kalender?date=2026-10-14&focus=training-42')

    expect(await screen.findByRole('dialog')).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Athletik' })).toBeTruthy()
  })

  test('holt ein Spiel außerhalb der geladenen Liste einzeln nach', async () => {
    seed({ games: [], detail: () => Promise.resolve({ data: game({ id: 99, opponent: 'Spät in der Saison' }) }) })
    renderAt('/kalender?date=2026-10-11&focus=game-99')

    expect(await screen.findByRole('dialog')).toBeTruthy()
    expect(screen.getByText('Spät in der Saison')).toBeTruthy()
    expect(mockGet).toHaveBeenCalledWith('/games/99')
  })

  test('nicht verfügbarer Termin zeigt einen Hinweis statt eines Dialogs', async () => {
    seed({ games: [] })
    const router = renderAt('/kalender?date=2026-10-11&focus=game-99999')

    expect(await screen.findByText('Dieser Termin ist nicht verfügbar')).toBeTruthy()
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(screen.getByText(/Oktober 2026/)).toBeTruthy()
    await waitFor(() => expect(router.state.location.search).toBe('?date=2026-10-11'))
  })

  test('ungültiges Fokus-Format wird ignoriert', async () => {
    seed({ games: [game()] })
    renderAt('/kalender?date=2026-10-11&focus=foobar')

    await screen.findByText(/Oktober 2026/)
    await waitFor(() => expect(mockGet).toHaveBeenCalledWith('/games?limit=500'))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(screen.queryByText('Dieser Termin ist nicht verfügbar')).toBeNull()
  })
})
