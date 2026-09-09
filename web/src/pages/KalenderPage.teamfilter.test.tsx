import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import KalenderPage from './KalenderPage'

// Teil von openspec/changes/team-mehrfachfilter-alle-listen: der Kalender nutzt
// denselben Mannschafts-Filter wie /termine. Sein Filterzustand lebt weiterhin
// im Komponenten-State (nicht in der URL) — geprüft wird deshalb über das
// Gitter und über den `team_id`-Parameter der Abwesenheiten-Route, die einzige
// serverseitig gefilterte Datenquelle der Seite.

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../hooks/useCompactHeader', () => ({ useCompactHeader: () => false }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: false, clubFunctions: ['trainer'] },
    // manage_games schaltet den Abwesenheiten-Knopf frei (canSeeTeamAbsences),
    // über den der einzige serverseitige Team-Filter der Seite läuft.
    hasCapability: (cap: string) => cap === 'manage_games',
    logout: vi.fn(),
  }),
}))
vi.mock('../lib/useEscapeKey', () => ({ useEscapeKey: vi.fn() }))

const NOW = new Date(2026, 7, 15, 10, 0, 0)
const MONTH = '2026-08'

const TEAMS = [
  { id: 1, name: 'A-Jugend männlich 2', age_class: 'A-Jugend', gender: 'm', team_number: 2, group_count: 2, is_active: true },
  { id: 2, name: 'C-Jugend männlich 2', age_class: 'C-Jugend', gender: 'm', team_number: 2, group_count: 2, is_active: true },
  { id: 3, name: 'B-Jugend weiblich', age_class: 'B-Jugend', gender: 'f', team_number: 1, group_count: 1, is_active: true },
]

function game(id: number, teamId: number, opponent: string) {
  return {
    id,
    date: `${MONTH}-14`,
    time: '15:00',
    opponent,
    teams: [{ id: teamId, name: `Team ${teamId}` }],
    event_type: 'heim',
    slot_count: 0, filled_count: 0, total_count: 0,
    confirmed_count: 0, declined_count: 0, maybe_count: 0,
    venue: { id: 1, name: 'Halle', street: '', city: 'Ostfildern', postal_code: '', note: '' },
  }
}

const GAMES = [
  game(1, 1, 'Ludwigsburg'),
  game(2, 2, 'Oppenweiler'),
  game(3, 3, 'Göppingen'),
]

function seed() {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/games')) return Promise.resolve({ data: GAMES })
    if (url.includes('/absences')) return Promise.resolve({ data: [] })
    if (url.startsWith('/teams/names')) return Promise.resolve({ data: TEAMS })
    if (url.startsWith('/teams')) return Promise.resolve({ data: TEAMS })
    return Promise.resolve({ data: [] })
  })
}

function absenceUrls(): string[] {
  return mockGet.mock.calls.map(c => String(c[0])).filter(u => u.includes('/absences/calendar'))
}

async function openTeamDropdown() {
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
  await user.click(screen.getByLabelText('Mannschafts-Filter'))
  return user
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(NOW)
  vi.clearAllMocks()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('KalenderPage — Team-Mehrfachfilter', () => {
  test('ohne Filter sind alle Mannschaften im Gitter', async () => {
    seed()
    render(<MemoryRouter initialEntries={['/kalender']}><KalenderPage /></MemoryRouter>)
    await waitFor(() => expect(screen.getByTitle(/Ludwigsburg/)).toBeTruthy())
    expect(screen.getByTitle(/Oppenweiler/)).toBeTruthy()
    expect(screen.getByTitle(/Göppingen/)).toBeTruthy()
  })

  test('Abwählen einer Mannschaft blendet nur deren Termine aus', async () => {
    seed()
    render(<MemoryRouter initialEntries={['/kalender']}><KalenderPage /></MemoryRouter>)
    await waitFor(() => expect(screen.getByTitle(/Oppenweiler/)).toBeTruthy())
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'mC2' }))

    await waitFor(() => expect(screen.queryByTitle(/Oppenweiler/)).toBeNull())
    expect(screen.getByTitle(/Ludwigsburg/)).toBeTruthy()
    expect(screen.getByTitle(/Göppingen/)).toBeTruthy()
  })

  test('Abwählen aller Mannschaften zeigt wieder alles', async () => {
    seed()
    render(<MemoryRouter initialEntries={['/kalender']}><KalenderPage /></MemoryRouter>)
    await waitFor(() => expect(screen.getByTitle(/Ludwigsburg/)).toBeTruthy())
    const user = await openTeamDropdown()

    // Alle drei abwählen — die letzte Abwahl fällt auf „kein Filter" zurück.
    await user.click(screen.getByRole('checkbox', { name: 'mA2' }))
    await user.click(screen.getByRole('checkbox', { name: 'mC2' }))
    await user.click(screen.getByRole('checkbox', { name: 'wB' }))

    await waitFor(() => {
      const boxes = screen.getAllByRole('checkbox') as HTMLInputElement[]
      expect(boxes.every(b => b.checked)).toBe(true)
    })
    expect(screen.getByTitle(/Ludwigsburg/)).toBeTruthy()
    expect(screen.getByTitle(/Oppenweiler/)).toBeTruthy()
    expect(screen.getByTitle(/Göppingen/)).toBeTruthy()
  })

  test('Abwesenheiten werden mit der ID-Liste nachgeladen', async () => {
    seed()
    render(<MemoryRouter initialEntries={['/kalender']}><KalenderPage /></MemoryRouter>)
    await waitFor(() => expect(screen.getByTitle(/Ludwigsburg/)).toBeTruthy())
    // Team-Abwesenheiten einschalten, sonst schickt die Seite kein team_id.
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    await user.click(screen.getByLabelText('Mannschaftsabwesenheiten'))
    await user.click(screen.getByLabelText('Mannschafts-Filter'))

    await user.click(screen.getByRole('checkbox', { name: 'wB' }))

    await waitFor(() => expect(absenceUrls().some(u => u.includes('team_id=1%2C2') || u.includes('team_id=1,2'))).toBe(true))
  })
})
