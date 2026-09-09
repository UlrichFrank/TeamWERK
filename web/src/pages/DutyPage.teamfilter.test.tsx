import { describe, test, expect, vi, beforeAll, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import DutyPage from './DutyPage'
import { PersonContactProvider } from '../contexts/PersonContactContext'

// Teil von openspec/changes/team-mehrfachfilter-alle-listen: das Board nutzt
// denselben Mannschafts-Filter wie /termine — Mehrfachauswahl, `team` als
// ID-Liste, und der Fokus-Durchlass endet bei aktiver Filteränderung.

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
})

const TEAMS = [
  { id: 1, name: 'A-Jugend männlich 2', age_class: 'A-Jugend', gender: 'm', team_number: 2, group_count: 2, is_active: true },
  { id: 2, name: 'C-Jugend männlich 2', age_class: 'C-Jugend', gender: 'm', team_number: 2, group_count: 2, is_active: true },
  { id: 3, name: 'B-Jugend weiblich', age_class: 'B-Jugend', gender: 'f', team_number: 1, group_count: 1, is_active: true },
]

function boardGroup(overrides: Record<string, unknown> = {}) {
  return {
    game_id: 1,
    team_ids: [1],
    team_names: ['mA2'],
    date: '2026-09-14',
    event_time: '10:00',
    opponent: 'Ludwigsburg',
    event_type: 'heim',
    venue: 'Scharnhauser Park Halle',
    label: null,
    past: false,
    slots: [{
      id: 555, duty_type: 'Kasse', duty_type_id: 42, has_instruction: false,
      event_time: '10:00', hours_value: 1, slots_total: 2, vacancies: 1,
      claimed_by_me: false, assignees: [],
    }],
    ...overrides,
  }
}

const GROUPS = [
  boardGroup(),
  boardGroup({
    game_id: 2, team_ids: [2], team_names: ['mC2'], opponent: 'Oppenweiler',
    slots: [{
      id: 777, duty_type: 'Kuchendienst', duty_type_id: 43, has_instruction: false,
      event_time: '12:00', hours_value: 1, slots_total: 1, vacancies: 1,
      claimed_by_me: false, assignees: [],
    }],
  }),
  boardGroup({
    game_id: 3, team_ids: [3], team_names: ['wB'], opponent: 'Göppingen',
    slots: [{
      id: 888, duty_type: 'Hallenaufsicht', duty_type_id: 44, has_instruction: false,
      event_time: '14:00', hours_value: 1, slots_total: 1, vacancies: 1,
      claimed_by_me: false, assignees: [],
    }],
  }),
]

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', clubFunctions: [] },
    hasCapability: () => false,
  }),
}))

function seedRoutes() {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/duty-board')) return Promise.resolve({ data: GROUPS })
    if (url.startsWith('/teams')) return Promise.resolve({ data: TEAMS })
    if (url.startsWith('/family/proxy-accounts')) return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
}

function renderAt(route: string) {
  const router = createMemoryRouter(
    [{
      path: '/dienste',
      element: <PersonContactProvider><DutyPage /></PersonContactProvider>,
    }],
    { initialEntries: [route] },
  )
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

describe('DutyPage — Team-Mehrfachfilter', () => {
  test('team=1,2 zeigt beide Mannschaften, die dritte nicht', async () => {
    seedRoutes()
    renderAt('/dienste?team=1,2')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.getByText('Kuchendienst')).toBeTruthy()
    expect(screen.queryByText('Hallenaufsicht')).toBeNull()
  })

  test('einzelne Team-ID bleibt gültig', async () => {
    seedRoutes()
    renderAt('/dienste?team=1')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.queryByText('Kuchendienst')).toBeNull()
  })

  test('Abwählen einer Mannschaft schreibt die restlichen in die URL', async () => {
    seedRoutes()
    const { search } = renderAt('/dienste')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'mC2' }))

    await waitFor(() => expect(search().get('team')).toBe('1,3'))
    expect(screen.queryByText('Kuchendienst')).toBeNull()
  })

  test('Abwählen aller Mannschaften ist kein Filter', async () => {
    seedRoutes()
    const { search } = renderAt('/dienste?team=1')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'mA2' }))

    await waitFor(() => expect(search().has('team')).toBe(false))
    expect(screen.getByText('Kuchendienst')).toBeTruthy()
    expect(screen.getByText('Hallenaufsicht')).toBeTruthy()
  })

  test('Team-Filteränderung beendet den Fokus', async () => {
    seedRoutes()
    const { search } = renderAt('/dienste?focus=game-2')
    await waitFor(() => expect(screen.getByText('Kuchendienst')).toBeTruthy())
    const user = await openTeamDropdown()

    await user.click(screen.getByRole('checkbox', { name: 'mC2' }))

    await waitFor(() => expect(search().has('focus')).toBe(false))
    expect(screen.queryByText('Kuchendienst')).toBeNull()
  })
})
