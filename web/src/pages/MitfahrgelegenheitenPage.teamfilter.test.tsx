import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import MitfahrgelegenheitenPage from './MitfahrgelegenheitenPage'

// Teil von openspec/changes/team-mehrfachfilter-alle-listen: Mitfahrten nutzt
// denselben Mannschafts-Filter wie /termine. Der Filter wirkt jetzt
// clientseitig — die Seite lädt einmal und filtert die geladene Menge, statt
// pro Filterklick `?team_id=` neu abzurufen.

const TEAMS = [
  { id: 1, name: 'A-Jugend männlich 2', age_class: 'A-Jugend', gender: 'm', team_number: 2, group_count: 2, is_active: true },
  { id: 2, name: 'C-Jugend männlich 2', age_class: 'C-Jugend', gender: 'm', team_number: 2, group_count: 2, is_active: true },
  { id: 3, name: 'B-Jugend weiblich', age_class: 'B-Jugend', gender: 'f', team_number: 1, group_count: 1, is_active: true },
]

function carpoolGame(id: number, teamIds: number[], opponent: string) {
  return {
    game: {
      id,
      date: '2026-09-19',
      time: '15:00',
      opponent,
      team: 'Team',
      teamIds,
      eventType: 'auswärts',
    },
    biete: [],
    suche: [],
    paarungen: [],
  }
}

const GAMES = [
  carpoolGame(1, [1], 'Ludwigsburg'),
  carpoolGame(2, [2], 'Oppenweiler'),
  carpoolGame(3, [3], 'Göppingen'),
]

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({
  api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() },
}))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, email: 'test@example.com', role: 'standard' } }),
}))

function seedRoutes() {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/mitfahrgelegenheiten')) return Promise.resolve({ data: { games: GAMES, children: [] } })
    if (url.startsWith('/teams')) return Promise.resolve({ data: TEAMS })
    return Promise.resolve({ data: [] })
  })
}

function renderAt(route: string) {
  const router = createMemoryRouter([{ path: '/mitfahrten', element: <MitfahrgelegenheitenPage /> }], {
    initialEntries: [route],
  })
  render(<RouterProvider router={router} />)
  return { search: () => new URLSearchParams(router.state.location.search) }
}

async function openTeamDropdown() {
  const user = userEvent.setup()
  await user.click(screen.getByLabelText('Mannschafts-Filter'))
  return user
}

beforeEach(() => {
  mockGet.mockReset()
})

describe('MitfahrgelegenheitenPage — Team-Mehrfachfilter', () => {
  test('team=1,2 zeigt beide Mannschaften, die dritte nicht', async () => {
    seedRoutes()
    renderAt('/mitfahrten?team=1,2')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.getByText(/Oppenweiler/)).toBeTruthy()
    expect(screen.queryByText(/Göppingen/)).toBeNull()
  })

  test('einzelne Team-ID bleibt gültig', async () => {
    seedRoutes()
    renderAt('/mitfahrten?team=1')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.queryByText(/Oppenweiler/)).toBeNull()
  })

  test('Abwählen einer Mannschaft filtert ohne erneutes Laden', async () => {
    seedRoutes()
    const { search } = renderAt('/mitfahrten')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    const callsBefore = mockGet.mock.calls.filter(c => String(c[0]).startsWith('/mitfahrgelegenheiten')).length
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'mC2' }))

    await waitFor(() => expect(search().get('team')).toBe('1,3'))
    expect(screen.queryByText(/Oppenweiler/)).toBeNull()
    const callsAfter = mockGet.mock.calls.filter(c => String(c[0]).startsWith('/mitfahrgelegenheiten')).length
    expect(callsAfter).toBe(callsBefore)
  })

  test('Abwählen aller Mannschaften ist kein Filter', async () => {
    seedRoutes()
    const { search } = renderAt('/mitfahrten?team=1')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'mA2' }))

    await waitFor(() => expect(search().has('team')).toBe(false))
    expect(screen.getByText(/Oppenweiler/)).toBeTruthy()
    expect(screen.getByText(/Göppingen/)).toBeTruthy()
  })

  test('die Liste wird ohne team_id-Parameter geladen', async () => {
    seedRoutes()
    renderAt('/mitfahrten?team=1')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    const urls = mockGet.mock.calls.map(c => String(c[0])).filter(u => u.startsWith('/mitfahrgelegenheiten'))
    expect(urls.length).toBeGreaterThan(0)
    expect(urls.every(u => !u.includes('team_id'))).toBe(true)
  })
})
