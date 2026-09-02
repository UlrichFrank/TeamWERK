import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import KalenderPage from './KalenderPage'

// Teil von openspec/changes/termin-textfilter: im Monatsgitter ist q ein
// Filter wie der Team-Filter — die Zellen leeren sich, es gibt keinen eigenen
// Ergebnis-Modus. Abwesenheiten müssen mitgefiltert werden, sonst bleiben
// verwaiste Balken stehen.

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../hooks/useCompactHeader', () => ({ useCompactHeader: () => false }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: false, clubFunctions: [] },
    hasCapability: () => false,
    logout: vi.fn(),
  }),
}))
vi.mock('../lib/useEscapeKey', () => ({ useEscapeKey: vi.fn() }))

// Der Kalender startet im aktuellen Monat — die Termine unten müssen also in
// genau diesem Monat liegen, sonst rendert das Gitter sie gar nicht. Statt sich
// auf die Wanduhr zu verlassen (kippte beim Monatswechsel), wird `Date` auf
// August 2026 festgenagelt; Timer bleiben echt, damit waitFor weiter läuft.
const NOW = new Date(2026, 7, 15, 10, 0, 0) // 15.08.2026, lokale Zeit
const MONTH = '2026-08'

function game(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    date: `${MONTH}-14`,
    time: '15:00',
    opponent: 'Ludwigsburg',
    teams: [{ id: 1, name: 'Team A' }],
    event_type: 'heim',
    slot_count: 0,
    filled_count: 0,
    total_count: 0,
    confirmed_count: 0,
    declined_count: 0,
    maybe_count: 0,
    venue: { id: 1, name: 'Scharnhauser Park Halle', street: '', city: 'Ostfildern', postal_code: '', note: '' },
    ...overrides,
  }
}

function absence(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    member_id: 5,
    member_name: 'Erika Müller',
    can_edit: false,
    type: 'vacation',
    start_date: `${MONTH}-10`,
    end_date: `${MONTH}-12`,
    note: '',
    created_by: 1,
    is_own: false,
    ...overrides,
  }
}

function seed({ games = [] as unknown[], absences = [] as unknown[] }) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/games')) return Promise.resolve({ data: games })
    if (url.includes('/absences')) return Promise.resolve({ data: absences })
    return Promise.resolve({ data: [] })
  })
}

function renderKalender(route = '/kalender') {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <KalenderPage />
    </MemoryRouter>,
  )
}

describe('KalenderPage — Textfilter (?q=)', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(NOW)
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  test('ohne q sind beide Termine im Gitter', async () => {
    seed({ games: [game(), game({ id: 2, date: `${MONTH}-20`, opponent: 'Göppingen' })] })
    renderKalender()
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.getByText(/Göppingen/)).toBeTruthy()
  })

  test('q leert die nicht passenden Zellen', async () => {
    seed({ games: [game(), game({ id: 2, date: `${MONTH}-20`, opponent: 'Göppingen' })] })
    renderKalender('/kalender?q=ludwigsburg')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.queryByText(/Göppingen/)).toBeNull()
  })

  test('filtert über den Ort', async () => {
    seed({ games: [game(), game({ id: 2, date: `${MONTH}-20`, opponent: 'Göppingen', venue: null })] })
    renderKalender('/kalender?q=scharnhauser')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.queryByText(/Göppingen/)).toBeNull()
  })

  // Invariante 9
  test('leeres q ändert nichts', async () => {
    seed({ games: [game(), game({ id: 2, date: `${MONTH}-20`, opponent: 'Göppingen' })] })
    renderKalender('/kalender?q=%20%20')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(screen.getByText(/Göppingen/)).toBeTruthy()
  })

  // Der Abwesenheits-Balken ist ein leeres <div>, das seinen Inhalt nur im
  // title-Attribut trägt — getByText greift dort nicht.
  const absenceBars = (container: HTMLElement) =>
    container.querySelectorAll('[title*="Erika Müller"]')

  test('ohne q ist der Abwesenheits-Balken da', async () => {
    seed({ games: [], absences: [absence()] })
    const { container } = renderKalender()
    await waitFor(() => expect(absenceBars(container).length).toBeGreaterThan(0))
  })

  // Invariante 10: ohne die Filterung in absencesForDay blieben die Balken
  // stehen, während die Termine drumherum verschwinden — das Gitter sähe kaputt aus.
  test('bei nicht passendem q bleibt kein Abwesenheits-Balken stehen', async () => {
    seed({ games: [game()], absences: [absence()] })
    const { container } = renderKalender('/kalender?q=ludwigsburg')
    await waitFor(() => expect(screen.getByText(/Ludwigsburg/)).toBeTruthy())
    expect(absenceBars(container).length).toBe(0)
  })

  test('Abwesenheit bleibt sichtbar, wenn q auf ihren Namen passt', async () => {
    seed({ games: [], absences: [absence()] })
    const { container } = renderKalender('/kalender?q=müller')
    await waitFor(() => expect(absenceBars(container).length).toBeGreaterThan(0))
  })
})
