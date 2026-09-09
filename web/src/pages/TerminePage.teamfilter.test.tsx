import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import TerminePage from './TerminePage'

// Teil von openspec/changes/termine-team-mehrfachfilter:
// (1) `team` nimmt mehrere IDs, Deselektion aller = kein Filter (wie beim Typ-Filter),
// (2) der Deep-Link-Fokus endet, sobald der Nutzer selbst Team oder Typ ändert.

const mockGet = vi.fn()

vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: true },
    hasCapability: () => false,
  }),
}))

const SEASON = { id: 7, name: '2026/27', start_date: '2026-05-01', end_date: '2027-06-30' }

const TEAMS = [
  { id: 1, name: 'A-Jugend männlich 2', age_class: 'A-Jugend', gender: 'm', team_number: 2, group_count: 2, is_active: true },
  { id: 2, name: 'C-Jugend männlich 2', age_class: 'C-Jugend', gender: 'm', team_number: 2, group_count: 2, is_active: true },
  { id: 3, name: 'B-Jugend weiblich', age_class: 'B-Jugend', gender: 'f', team_number: 1, group_count: 1, is_active: true },
]

function game(overrides: Record<string, unknown>) {
  return {
    id: 200,
    date: '2026-09-19',
    time: '15:00',
    opponent: 'Ludwigsburg',
    event_type: 'auswärts',
    is_home: false,
    season_id: 7,
    team_names: 'A-Jugend männlich 2',
    team_ids: [1],
    confirmed_count: 0,
    declined_count: 0,
    maybe_count: 0,
    my_rsvp: null,
    am_i_participant: false,
    rsvp_default_players: 'none',
    rsvp_default_extended: 'none',
    rsvp_require_reason: 1,
    venue: null,
    ...overrides,
  }
}

// Die drei Termine aus dem Fehlerbild: ein mA2-Auswärtsspiel, ein mC2-Turnier
// (generisch) und ein Termin einer dritten Mannschaft als Gegenprobe.
const GAMES = [
  game({}),
  game({ id: 201, date: '2026-09-13', opponent: 'Turnier in Oppenweiler', event_type: 'generisch', team_ids: [2], team_names: 'C-Jugend männlich 2' }),
  game({ id: 202, date: '2026-09-20', opponent: 'Göppingen', team_ids: [3], team_names: 'B-Jugend weiblich' }),
]

function seedRoutes() {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/seasons/active')) return Promise.resolve({ data: SEASON })
    if (url.startsWith('/training-sessions')) return Promise.resolve({ data: [] })
    if (url.startsWith('/games/my')) return Promise.resolve({ data: GAMES })
    if (url.startsWith('/teams')) return Promise.resolve({ data: TEAMS })
    return Promise.resolve({ data: [] })
  })
}

function renderAt(initialPath: string) {
  const router = createMemoryRouter([{ path: '/termine', element: <TerminePage /> }], {
    initialEntries: [initialPath],
  })
  render(<RouterProvider router={router} />)
  return {
    search: () => new URLSearchParams(router.state.location.search),
  }
}

async function openTeamDropdown() {
  const user = userEvent.setup()
  await user.click(screen.getByLabelText('Mannschafts-Filter'))
  return user
}

beforeEach(() => {
  mockGet.mockReset()
  // jsdom kennt scrollIntoView nicht; der Fokus-Scroll löst es aus.
  Element.prototype.scrollIntoView = vi.fn()
})

describe('TerminePage — Team-Mehrfachfilter', () => {
  test('team=1,2 zeigt beide Mannschaften, die dritte nicht', async () => {
    seedRoutes()
    renderAt('/termine?past=1&team=1,2')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.getByText(/Turnier in Oppenweiler/)).toBeTruthy()
    expect(screen.queryByText(/Göppingen/)).toBeNull()
  })

  test('einzelne Team-ID bleibt gültig (Bestandslinks)', async () => {
    seedRoutes()
    renderAt('/termine?past=1&team=1')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.queryByText(/Turnier in Oppenweiler/)).toBeNull()
    expect(screen.queryByText(/Göppingen/)).toBeNull()
  })

  test('ohne team-Parameter sind alle Kästchen angehakt', async () => {
    seedRoutes()
    renderAt('/termine?past=1')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    await openTeamDropdown()
    const boxes = screen.getAllByRole('checkbox') as HTMLInputElement[]
    expect(boxes).toHaveLength(TEAMS.length)
    expect(boxes.every(b => b.checked)).toBe(true)
  })

  test('Abwählen einer Mannschaft schreibt die restlichen in die URL', async () => {
    seedRoutes()
    const { search } = renderAt('/termine?past=1')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'mC2' }))

    await waitFor(() => expect(search().get('team')).toBe('1,3'))
    expect(screen.queryByText(/Turnier in Oppenweiler/)).toBeNull()
    expect(screen.getByText(/Ludwigsburg/)).toBeTruthy()
  })

  test('Abwählen aller Mannschaften ist kein Filter — wie beim Typ-Filter', async () => {
    seedRoutes()
    const { search } = renderAt('/termine?past=1&team=1')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'mA2' }))

    await waitFor(() => expect(search().has('team')).toBe(false))
    expect(screen.getByText(/Turnier in Oppenweiler/)).toBeTruthy()
    expect(screen.getByText(/Göppingen/)).toBeTruthy()
    const boxes = screen.getAllByRole('checkbox') as HTMLInputElement[]
    expect(boxes.every(b => b.checked)).toBe(true)
  })

  test('Anhaken aller Mannschaften entfernt den Parameter', async () => {
    seedRoutes()
    const { search } = renderAt('/termine?past=1&team=1,2')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'wB' }))

    await waitFor(() => expect(search().has('team')).toBe(false))
  })

  test('unbrauchbarer team-Parameter wirkt wie kein Filter', async () => {
    seedRoutes()
    renderAt('/termine?past=1&team=abc')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.getByText(/Turnier in Oppenweiler/)).toBeTruthy()
    expect(screen.getByText(/Göppingen/)).toBeTruthy()
  })
})

describe('TerminePage — Fokus endet bei aktiver Filteränderung', () => {
  test('Fokus aus der URL überlebt den ersten Render mit Filter', async () => {
    seedRoutes()
    renderAt('/termine?past=1&team=1&focus=game-201')
    // Dokumentiertes Deep-Link-Verhalten: der fokussierte Termin kommt durch,
    // obwohl der Team-Filter ihn ausschließt.
    await waitFor(() => expect(screen.getByText(/Turnier in Oppenweiler/)).toBeTruthy())
  })

  test('Team-Filteränderung entfernt focus und blendet den Termin aus', async () => {
    seedRoutes()
    const { search } = renderAt('/termine?past=1&focus=game-201')
    await waitFor(() => expect(screen.getByText(/Turnier in Oppenweiler/)).toBeTruthy())
    const user = await openTeamDropdown()

    // mC2 abwählen — genau der Fall aus dem Fehlerbild.
    await user.click(screen.getByRole('checkbox', { name: 'mC2' }))

    await waitFor(() => expect(search().has('focus')).toBe(false))
    expect(screen.queryByText(/Turnier in Oppenweiler/)).toBeNull()
  })

  test('Typ-Filteränderung entfernt focus ebenfalls', async () => {
    seedRoutes()
    const { search } = renderAt('/termine?past=1&focus=game-201')
    await waitFor(() => expect(screen.getByText(/Turnier in Oppenweiler/)).toBeTruthy())
    const user = userEvent.setup()

    // „Sonstiges" abwählen — der Typ des fokussierten Turniers (generisch).
    await user.click(screen.getByLabelText('Sonstiges'))

    await waitFor(() => expect(search().has('focus')).toBe(false))
    expect(screen.queryByText(/Turnier in Oppenweiler/)).toBeNull()
  })

  test('der Vergangene-Toggle lässt den Fokus stehen', async () => {
    seedRoutes()
    const { search } = renderAt('/termine?past=1&team=1&focus=game-201')
    await waitFor(() => expect(screen.getByText(/Turnier in Oppenweiler/)).toBeTruthy())
    const user = userEvent.setup()

    await user.click(screen.getByLabelText('Vergangene anzeigen'))

    await waitFor(() => expect(search().has('past')).toBe(false))
    expect(search().get('focus')).toBe('game-201')
  })
})
