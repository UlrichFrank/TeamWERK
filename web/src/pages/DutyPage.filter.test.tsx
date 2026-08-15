import { describe, test, expect, vi, beforeAll, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DutyPage from './DutyPage'
import { PersonContactProvider } from '../contexts/PersonContactContext'

// Teil von openspec/changes/termin-textfilter: der Textfilter ist ein Filter,
// kein Suchmodus — er UND-verknüpft sich mit Team-/Typ-Filter, lässt den
// fokussierten Termin durch und weist aus, wenn andere Filter Treffer
// verdecken.

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
})

function boardGroup(overrides: Record<string, unknown> = {}) {
  return {
    game_id: 1,
    team_ids: [1],
    team_names: ['Team A'],
    date: '2026-09-14',
    event_time: '10:00',
    opponent: 'Ludwigsburg',
    event_type: 'heim',
    venue: 'Scharnhauser Park Halle',
    label: null,
    past: false,
    slots: [
      {
        id: 555,
        duty_type: 'Kasse',
        duty_type_id: 42,
        has_instruction: false,
        event_time: '10:00',
        slots_total: 2,
        vacancies: 1,
        claimed_by_me: false,
        assignees: [],
      },
    ],
    ...overrides,
  }
}

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', clubFunctions: [] },
    hasCapability: () => false,
  }),
}))

function seedRoutes(groups: unknown[], teams: unknown[] = []) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/duty-board')) return Promise.resolve({ data: groups })
    if (url.startsWith('/teams')) return Promise.resolve({ data: teams })
    if (url.startsWith('/family/proxy-accounts')) return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
}

function renderPage(route: string) {
  return render(
    // PersonChip (Assignee-Namen) hängt am PersonContactContext — ohne Provider
    // wirft schon das Rendern, unabhängig vom Filter.
    <MemoryRouter initialEntries={[route]}>
      <PersonContactProvider>
        <DutyPage />
      </PersonContactProvider>
    </MemoryRouter>,
  )
}

// Zwei Gruppen: Ludwigsburg (Team A, Kasse) und Göppingen (Team B, Kuchen).
function zweiGruppen() {
  return [
    boardGroup(),
    boardGroup({
      game_id: 2,
      team_ids: [2],
      team_names: ['Team B'],
      date: '2026-10-05',
      opponent: 'Göppingen',
      venue: 'Sporthalle Ost',
      slots: [
        {
          id: 777,
          duty_type: 'Kuchendienst',
          duty_type_id: 43,
          has_instruction: false,
          event_time: '12:00',
          slots_total: 1,
          vacancies: 1,
          claimed_by_me: false,
          assignees: [{ user_id: 9, name: 'Erika Müller' }],
        },
      ],
    }),
  ]
}

describe('DutyPage — Textfilter', () => {
  beforeEach(() => {
    mockGet.mockReset()
  })

  test('ohne q sind alle Gruppen sichtbar', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.getByText('Kuchendienst')).toBeTruthy()
  })

  // Invariante 9: leeres q darf nichts wegschneiden.
  test('leeres q ist ein No-Op', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste?q=%20%20')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.getByText('Kuchendienst')).toBeTruthy()
  })

  test('filtert über den Gegner', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste?q=ludwigsburg')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.queryByText('Kuchendienst')).toBeNull()
  })

  test('filtert über den Ort — das Feld, das es vorher in der API nicht gab', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste?q=scharnhauser')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.queryByText('Kuchendienst')).toBeNull()
  })

  test('filtert über den Diensttyp', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste?q=kuchen')
    await waitFor(() => expect(screen.getByText('Kuchendienst')).toBeTruthy())
    expect(screen.queryByText('Kasse')).toBeNull()
  })

  test('filtert über die zugewiesene Person', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste?q=müller')
    await waitFor(() => expect(screen.getByText('Kuchendienst')).toBeTruthy())
    expect(screen.queryByText('Kasse')).toBeNull()
  })

  test('filtert über ein jahresloses Datum', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste?q=14.09.')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.queryByText('Kuchendienst')).toBeNull()
  })

  // Invariante 7, erster Teil: q komponiert mit dem Team-Filter, es ersetzt ihn
  // nicht.
  test('UND-verknüpft sich mit dem Team-Filter', async () => {
    seedRoutes(zweiGruppen(), [
      { id: 1, name: 'Team A', age_class: '', gender: '', team_number: 1, group_count: 1, is_active: true },
      { id: 2, name: 'Team B', age_class: '', gender: '', team_number: 1, group_count: 1, is_active: true },
    ])
    renderPage('/dienste?team=2&q=ludwigsburg')
    // Ludwigsburg gehört Team A — mit Team-Filter B bleibt nichts übrig.
    await waitFor(() => expect(screen.queryByText('Kasse')).toBeNull())
    expect(screen.queryByText('Kuchendienst')).toBeNull()
  })

  // Invariante 7, zweiter Teil: der verdeckte Treffer wird ausgewiesen.
  test('weist von anderen Filtern verdeckte Treffer mit exakter Anzahl aus', async () => {
    seedRoutes(zweiGruppen(), [
      { id: 1, name: 'Team A', age_class: '', gender: '', team_number: 1, group_count: 1, is_active: true },
      { id: 2, name: 'Team B', age_class: '', gender: '', team_number: 1, group_count: 1, is_active: true },
    ])
    renderPage('/dienste?team=2&q=ludwigsburg')
    await waitFor(() =>
      expect(screen.getByText(/1 Termin passt, wird aber von aktiven Filtern ausgeblendet/)).toBeTruthy(),
    )
    expect(screen.getByText('Filter zurücksetzen')).toBeTruthy()
  })

  test('ohne weiteren aktiven Filter erscheint kein Ausgeblendet-Hinweis', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste?q=gibtesnicht')
    await waitFor(() => expect(screen.getByText('Keine Dienste passen zum Filter.')).toBeTruthy())
    expect(screen.queryByText('Filter zurücksetzen')).toBeNull()
  })

  // Invariante 8: der Fokus-Durchlass steht über dem Textfilter, sonst zerreißt
  // der Zurück-Sprung, sobald ein q in der URL steht.
  test('fokussierte Gruppe bleibt trotz nicht passendem q sichtbar', async () => {
    seedRoutes(zweiGruppen())
    renderPage('/dienste?focus=slot-555&q=gibtesnicht')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    // Nur die fokussierte Gruppe — die andere filtert q korrekt weg.
    expect(screen.queryByText('Kuchendienst')).toBeNull()
  })
})
