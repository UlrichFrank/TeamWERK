import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import TerminePage from './TerminePage'
import { isoDaysFrom } from '../lib/terminWindow'

// openspec/changes/termin-matrix: Tabellenansicht auf /termine.

const mockGet = vi.fn()
const mockPost = vi.fn()
let canOverride = false

vi.mock('../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
  },
}))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: true },
    hasCapability: (c: string) => c === 'manage_games' && canOverride,
  }),
}))

const NOW = new Date()
const PAST1 = isoDaysFrom(NOW, -16)
const PAST2 = isoDaysFrom(NOW, -10)
const FUTURE = isoDaysFrom(NOW, 5)
const FAR_FUTURE_LOCK = isoDaysFrom(NOW, 4) + 'T22:00:00Z'
const PAST_LOCK = PAST2 + 'T13:00:00Z'

const SEASON = { id: 7, name: 'Saison', start_date: isoDaysFrom(NOW, -60), end_date: isoDaysFrom(NOW, 200) }

const TEAMS = [
  { id: 1, name: 'A-Jugend männlich', age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 1, is_active: true },
  { id: 2, name: 'B-Jugend weiblich', age_class: 'B-Jugend', gender: 'f', team_number: 1, group_count: 1, is_active: true },
]

const cell = (status: string | null, extra: Record<string, unknown> = {}) => ({ status, is_default: false, ...extra })

function matrix() {
  return {
    team_id: 1,
    team_name: 'A-Jugend männlich',
    events: [
      { kind: 'training', id: 11, date: PAST1, time: '18:00', event_type: 'training', title: 'Training', cancelled: false, rsvp_locks_at: PAST1 + 'T14:00:00Z', rsvp_require_reason: false },
      { kind: 'game', id: 21, date: PAST2, time: '15:00', event_type: 'heim', title: 'Ludwigsburg', cancelled: false, rsvp_locks_at: PAST_LOCK, rsvp_require_reason: false },
      { kind: 'training', id: 12, date: FUTURE, time: '18:00', event_type: 'training', title: 'Training', cancelled: false, rsvp_locks_at: FAR_FUTURE_LOCK, rsvp_require_reason: true },
    ],
    members: [
      {
        member_id: 101, name: 'Philip Lei', extended: false, is_self: true, can_respond: true,
        cells: [cell('confirmed'), cell('confirmed'), cell('confirmed')],
      },
      {
        member_id: 102, name: 'Lias Stein', extended: false, is_self: false, can_respond: true,
        cells: [cell('confirmed'), cell(null), cell(null)],
      },
      {
        member_id: 103, name: 'Justus W', extended: true, is_self: false, can_respond: false,
        cells: [cell('declined'), { status: 'confirmed', is_default: true }, cell('declined')],
      },
    ],
  }
}

function seedRoutes() {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/seasons/active')) return Promise.resolve({ data: SEASON })
    if (url.includes('/rsvp-matrix')) return Promise.resolve({ data: matrix() })
    if (url.startsWith('/training-sessions')) return Promise.resolve({ data: [] })
    if (url.startsWith('/games/my')) return Promise.resolve({ data: [] })
    if (url.startsWith('/practice-groups/my')) return Promise.resolve({ data: [] })
    if (url.startsWith('/teams')) return Promise.resolve({ data: TEAMS })
    return Promise.resolve({ data: [] })
  })
  mockPost.mockResolvedValue({ status: 204 })
}

function renderAt(initialPath: string) {
  const router = createMemoryRouter([{ path: '/termine', element: <TerminePage /> }], {
    initialEntries: [initialPath],
  })
  render(<RouterProvider router={router} />)
  return { search: () => new URLSearchParams(router.state.location.search) }
}

const matrixCalls = () => mockGet.mock.calls.map(c => c[0] as string).filter(u => u.includes('/rsvp-matrix'))
const row = (name: string) => screen.getByText(name).closest('tr') as HTMLElement

beforeEach(() => {
  mockGet.mockReset()
  mockPost.mockReset()
  canOverride = false
  Element.prototype.scrollIntoView = vi.fn()
})

describe('TerminePage — Tabellenansicht', () => {
  test('Umschalten setzt view=tabelle und die erste Mannschaft', async () => {
    seedRoutes()
    const { search } = renderAt('/termine')
    const user = userEvent.setup()
    await waitFor(() => expect(mockGet).toHaveBeenCalledWith('/teams'))
    await user.click(screen.getByLabelText('Tabellenansicht'))

    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    expect(search().get('view')).toBe('tabelle')
    expect(search().get('team')).toBe('1')
    expect(matrixCalls()[0]).toMatch(/^\/teams\/1\/rsvp-matrix\?from=\d{4}-\d{2}-\d{2}&to=\d{4}-\d{2}-\d{2}$/)
  })

  test('Teilnahme getrennt nach Bisher und Geplant', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1')
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())

    const cells = (name: string) => within(row(name)).getAllByRole('cell').map(c => c.textContent)
    // Spalten: Bisher, Geplant, dann Termine.
    expect(cells('Philip Lei').slice(0, 2)).toEqual(['2 (100 %)', '1 (100 %)'])
    expect(cells('Lias Stein').slice(0, 2)).toEqual(['1 (50 %)', '0 (0 %)'])
    // Voreinstellung zählt als Zusage.
    expect(cells('Justus W').slice(0, 2)).toEqual(['1 (50 %)', '0 (0 %)'])
    expect(screen.getByRole('link', { name: /Heimspiel/ }).getAttribute('href')).toBe('/termine/spiel/21')
  })

  test('Typ-Filter blendet Spalten aus und die Quote folgt', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1&types=training')
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    expect(screen.queryByRole('link', { name: /Heimspiel/ })).toBeNull()
    expect(within(row('Lias Stein')).getAllByRole('cell')[0].textContent).toBe('1 (100 %)')
  })

  test('nur eigene und Kind-Zeilen sind antippbar — vergangene nicht', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1')
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    // Die beiden vergangenen Termine liegen hinter der Frist: nur der künftige bleibt.
    expect(within(row('Philip Lei')).getAllByRole('button')).toHaveLength(1)
    expect(within(row('Lias Stein')).getAllByRole('button')).toHaveLength(1)
    expect(within(row('Justus W')).queryAllByRole('button')).toHaveLength(0)
  })

  test('Kind: Zusage geht mit member_id raus und lädt die Matrix nach', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1')
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByText('Lias Stein')).toBeTruthy())
    const before = matrixCalls().length

    await user.click(within(row('Lias Stein')).getAllByRole('button')[0])
    const dialog = screen.getByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Zusagen' }))

    await waitFor(() => expect(mockPost).toHaveBeenCalledWith('/training-sessions/12/respond', { status: 'confirmed', reason: '', member_id: 102 }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(matrixCalls().length).toBeGreaterThan(before)
  })

  test('eigene Zeile: Zusagen schaltet wie in der Liste auf Vielleicht, ohne member_id', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1')
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    await user.click(within(row('Philip Lei')).getAllByRole('button')[0])
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Zusagen' }))
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith('/training-sessions/12/respond', { status: 'maybe', reason: '' }))
  })

  test('Absage mit Begründungspflicht öffnet den Begründungs-Dialog', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1')
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    await user.click(within(row('Philip Lei')).getAllByRole('button')[0])
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Absagen' }))

    await user.type(screen.getByPlaceholderText('Begründung…'), 'krank')
    await user.click(screen.getByRole('button', { name: 'OK' }))
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith('/training-sessions/12/respond', { status: 'declined', reason: 'krank' }))
  })

  test('verstrichene Frist: vergangene Zelle öffnet keinen Dialog', async () => {
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1')
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    const cells = within(row('Philip Lei')).getAllByRole('cell')
    expect(within(cells[cells.length - 2]).queryByRole('button')).toBeNull()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  test('mit Override bleibt die Frist ohne Wirkung', async () => {
    canOverride = true
    seedRoutes()
    renderAt('/termine?view=tabelle&team=1')
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByText('Philip Lei')).toBeTruthy())
    expect(within(row('Philip Lei')).getAllByRole('button')).toHaveLength(3)
    await user.click(within(row('Philip Lei')).getAllByRole('button')[1])
    expect((within(screen.getByRole('dialog')).getByRole('button', { name: 'Zusagen' }) as HTMLButtonElement).disabled).toBe(false)
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
