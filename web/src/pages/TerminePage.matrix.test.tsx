import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import TerminePage from './TerminePage'

// openspec/changes/termin-matrix: Tabellenansicht auf /termine.

const mockGet = vi.fn()

vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard' },
    hasCapability: () => false,
  }),
}))

const SEASON = { id: 7, name: '2026/27', start_date: '2026-08-01', end_date: '2027-06-30' }

const TEAMS = [
  { id: 1, name: 'A-Jugend männlich', age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 1, is_active: true },
  { id: 2, name: 'B-Jugend weiblich', age_class: 'B-Jugend', gender: 'f', team_number: 1, group_count: 1, is_active: true },
]

const MATRIX = {
  team_id: 1,
  team_name: 'A-Jugend männlich',
  events: [
    { kind: 'training', id: 11, date: '2026-09-07', time: '18:00', event_type: 'training', title: 'Training', cancelled: false },
    { kind: 'game', id: 21, date: '2026-09-13', time: '15:00', event_type: 'heim', title: 'Ludwigsburg', cancelled: false },
  ],
  members: [
    {
      member_id: 101, name: 'Philip Lei', extended: false,
      cells: [{ status: 'confirmed', is_default: false }, { status: 'confirmed', is_default: false }],
    },
    {
      member_id: 102, name: 'Lias Stein', extended: false,
      cells: [{ status: 'confirmed', is_default: false }, { status: null, is_default: false }],
    },
    {
      member_id: 103, name: 'Justus W', extended: true,
      cells: [{ status: 'declined', is_default: false }, { status: 'confirmed', is_default: true }],
    },
  ],
}

function seedRoutes() {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/seasons/active')) return Promise.resolve({ data: SEASON })
    if (url.includes('/rsvp-matrix')) return Promise.resolve({ data: MATRIX })
    if (url.startsWith('/training-sessions')) return Promise.resolve({ data: [] })
    if (url.startsWith('/games/my')) return Promise.resolve({ data: [] })
    if (url.startsWith('/practice-groups/my')) return Promise.resolve({ data: [] })
    if (url.startsWith('/teams')) return Promise.resolve({ data: TEAMS })
    return Promise.resolve({ data: [] })
  })
}

function renderAt(initialPath: string) {
  const router = createMemoryRouter([{ path: '/termine', element: <TerminePage /> }], {
    initialEntries: [initialPath],
  })
  render(<RouterProvider router={router} />)
  return { search: () => new URLSearchParams(router.state.location.search) }
}

const matrixCalls = () => mockGet.mock.calls.map(c => c[0] as string).filter(u => u.includes('/rsvp-matrix'))

beforeEach(() => {
  mockGet.mockReset()
  Element.prototype.scrollIntoView = vi.fn()
})

describe('TerminePage — Tabellenansicht', () => {
  test('Umschalten setzt view=tabelle und die erste Mannschaft', async () => {
    seedRoutes()
    const { search } = renderAt('/termine')
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByLabelText('Tabellenansicht')).toBeTruthy())
    await waitFor(() => expect(mockGet).toHaveBeenCalledWith('/teams'))
    await user.click(screen.getByLabelText('Tabellenansicht'))

    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    expect(search().get('view')).toBe('tabelle')
    expect(search().get('team')).toBe('1')
    expect(matrixCalls()[0]).toMatch(/^\/teams\/1\/rsvp-matrix\?from=\d{4}-\d{2}-\d{2}&to=2027-06-30$/)
  })

  test('zeigt Zeile je Spieler, Spalte je Termin und die Teilnahme-Quote', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1')
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())

    const row = (name: string) => screen.getByText(name).closest('tr') as HTMLElement
    expect(within(row('Philip Lei')).getByText('2 (100 %)')).toBeTruthy()
    expect(within(row('Lias Stein')).getByText('1 (50 %)')).toBeTruthy()
    // Voreinstellung zählt als Zusage.
    expect(within(row('Justus W')).getByText('1 (50 %)')).toBeTruthy()
    expect(within(row('Lias Stein')).getByLabelText('keine Rückmeldung')).toBeTruthy()
    expect(within(row('Justus W')).getByLabelText('zugesagt (Voreinstellung)')).toBeTruthy()

    // Spaltenkopf führt zur Detailseite.
    expect(screen.getByRole('link', { name: /Heimspiel 13\.09\./ }).getAttribute('href')).toBe('/termine/spiel/21')
  })

  test('Typ-Filter blendet Spalten aus und die Quote folgt', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1&types=training')
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    expect(screen.queryByRole('link', { name: /Heimspiel/ })).toBeNull()
    const row = screen.getByText('Lias Stein').closest('tr') as HTMLElement
    expect(within(row).getByText('1 (100 %)')).toBeTruthy()
  })

  test('Mannschaftswechsel lädt die Matrix der neuen Mannschaft', async () => {
    seedRoutes()
    const { search } = renderAt('/termine?view=tabelle&team=1')
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    await user.selectOptions(screen.getByLabelText('Mannschaft'), '2')
    await waitFor(() => expect(matrixCalls().some(u => u.startsWith('/teams/2/'))).toBe(true))
    expect(search().get('team')).toBe('2')
  })

  test('Listenansicht lädt keine Matrix', async () => {
    seedRoutes()
    renderAt('/termine')
    await waitFor(() => expect(mockGet).toHaveBeenCalledWith(expect.stringMatching(/^\/games\/my/)))
    expect(matrixCalls()).toHaveLength(0)
  })
})
