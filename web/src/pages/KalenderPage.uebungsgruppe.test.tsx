import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import KalenderPage from './KalenderPage'

// Übungsgruppen-Trainings tragen team_id = 0 (Projektion von
// training_sessions.team_id IS NULL) und stehen deshalb nie in der aus
// /teams/names gebauten Kurznamen-Map. Ohne den Rückfall auf das vom Server
// gelieferte team_name (COALESCE auf den Kadernamen) beschriftete die Kachel
// sich stumm mit "Training".

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

const NOW = new Date(2026, 7, 15, 10, 0, 0) // 15.08.2026, lokale Zeit
const MONTH = '2026-08'

function training(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    title: '',
    date: `${MONTH}-14`,
    start_time: '17:00',
    end_time: '18:30',
    team_id: 0,
    team_name: 'Frühtraining',
    season_id: 1,
    note: '',
    status: 'active',
    confirmed_count: 0,
    declined_count: 0,
    maybe_count: 0,
    my_rsvp: null,
    venue: null,
    ...overrides,
  }
}

const TEAM_NAMES = [{ id: 7, age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 1 }]

function seed(trainings: unknown[]) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/training-sessions')) return Promise.resolve({ data: { items: trainings, total: trainings.length } })
    if (url.startsWith('/teams/names')) return Promise.resolve({ data: TEAM_NAMES })
    return Promise.resolve({ data: [] })
  })
}

function renderKalender() {
  return render(
    <MemoryRouter initialEntries={['/kalender']}>
      <KalenderPage />
    </MemoryRouter>,
  )
}

describe('KalenderPage — Übungsgruppen-Training', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(NOW)
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  // Der Kachel-Text steht zusätzlich im title-Attribut ("<Label> · <Uhrzeit>") —
  // darüber wird geprüft, weil "Training" als Wort auch im Typ-Filter der
  // Kopfzeile vorkommt und getByText es dort fände.
  const tileTitle = (container: HTMLElement) =>
    container.querySelector('[title$="· 17:00"]')?.getAttribute('title')

  test('Kachel zeigt den Namen der Übungsgruppe statt "Training"', async () => {
    seed([training()])
    const { container } = renderKalender()
    await waitFor(() => expect(screen.getByText('Frühtraining')).toBeTruthy())
    expect(tileTitle(container)).toBe('Frühtraining · 17:00')
  })

  test('Mannschafts-Training behält den Kurznamen aus /teams/names', async () => {
    seed([training({ team_id: 7, team_name: 'mA' })])
    const { container } = renderKalender()
    await waitFor(() => expect(screen.getByText('mA')).toBeTruthy())
    expect(tileTitle(container)).toBe('mA · 17:00')
  })

  test('ohne team_name und ohne Titel bleibt der Fallback "Training"', async () => {
    seed([training({ team_name: '' })])
    const { container } = renderKalender()
    await waitFor(() => expect(tileTitle(container)).toBe('Training · 17:00'))
  })
})
