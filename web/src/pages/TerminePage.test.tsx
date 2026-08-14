import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, createMemoryRouter, RouterProvider } from 'react-router-dom'
import TerminePage from './TerminePage'

// Minimal Session-Payload für /api/training-sessions
function trainingSession(overrides: Record<string, unknown>) {
  return {
    id: 100,
    series_id: null,
    title: 'Training',
    date: '2026-05-01',
    start_time: '18:00',
    end_time: '20:00',
    venue: null,
    note: '',
    status: 'active',
    cancel_reason: '',
    team_id: 1,
    team_name: 'Team A',
    confirmed_count: 0,
    declined_count: 1,
    maybe_count: 0,
    my_rsvp: 'declined',
    am_i_participant: true,
    rsvp_default_players: 'none',
    rsvp_default_extended: 'none',
    rsvp_require_reason: 1,
    ...overrides,
  }
}

const mockGet = vi.fn()
const authState = { is_parent: false }

vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: authState.is_parent },
    hasCapability: () => false,
  }),
}))

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/termine']}>
      <TerminePage />
    </MemoryRouter>,
  )
}

// mockGet wird für mehrere Routen aufgerufen; wir routen nach URL.
function seedRoutes(sessions: unknown[], games: unknown[] = []) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/training-sessions')) return Promise.resolve({ data: sessions })
    if (url.startsWith('/games/my')) return Promise.resolve({ data: games })
    if (url.startsWith('/teams')) return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
}

describe('TerminePage — eigener Absagegrund', () => {
  beforeEach(() => {
    mockGet.mockReset()
    authState.is_parent = false
  })

  test('zeigt my_reason unter den RSVP-Buttons, wenn im Payload gesetzt', async () => {
    seedRoutes([trainingSession({ my_reason: 'Klavierstunde' })])
    renderPage()
    await waitFor(() => expect(screen.getByText('Klavierstunde')).toBeTruthy())
  })

  test('rendert keine Grund-Zeile, wenn my_reason nicht gesetzt ist', async () => {
    seedRoutes([trainingSession({})])
    renderPage()
    // Warten bis Card gerendert (Team-Name ist ein zuverlässiger Marker)
    await waitFor(() => expect(screen.getByText(/Training/)).toBeTruthy())
    expect(screen.queryByText('Klavierstunde')).toBeNull()
  })

  test('zeigt Kind-Reason im children_rsvp-Payload für Elternteil', async () => {
    authState.is_parent = true
    seedRoutes([
      trainingSession({
        my_rsvp: null,
        am_i_participant: false,
        children_rsvp: [
          { member_id: 42, name: 'Anna', rsvp: 'declined', reason: 'Krank' },
        ],
      }),
    ])
    renderPage()
    await waitFor(() => expect(screen.getByText('Krank')).toBeTruthy())
  })
})

// Regression: Spieler im Kader ohne Response bei Default=none sieht drei
// (inaktive) RSVP-Buttons — nicht: gar keine Buttons. Deckt den bisherigen
// Bug ab, dass showOwn an my_rsvp !== null gekoppelt war.
describe('TerminePage — Buttons sichtbar bei am_i_participant=true (Default=none)', () => {
  beforeEach(() => {
    mockGet.mockReset()
    authState.is_parent = false
  })

  test('Spieler im Kader ohne Response → drei Buttons sichtbar, keiner aktiv', async () => {
    seedRoutes([trainingSession({ my_rsvp: null, am_i_participant: true })])
    renderPage()
    await waitFor(() => expect(screen.getByText('Zusagen')).toBeTruthy())
    expect(screen.getByText('Vielleicht')).toBeTruthy()
    expect(screen.getByText('Absagen')).toBeTruthy()
  })

  test('Fremder Nutzer (kein Kader, kein Elternteil) → keine Buttons', async () => {
    seedRoutes([trainingSession({ my_rsvp: null, am_i_participant: false })])
    renderPage()
    // Karte muss trotzdem rendern
    await waitFor(() => expect(screen.getByText('Training')).toBeTruthy())
    expect(screen.queryByText('Zusagen')).toBeNull()
  })
})

// Regression: eingetragener Urlaub → auto-declined via absence_id, Backend liefert
// my_rsvp='declined' + my_rsvp_locked=true. Frontend rendert alle drei Buttons
// sichtbar aber disabled und zeigt den Hinweis „Durch Abwesenheit gesperrt".
describe('TerminePage — Absence-Lock (my_rsvp_locked=true)', () => {
  beforeEach(() => {
    mockGet.mockReset()
    authState.is_parent = false
  })

  test('Spieler mit Urlaub sieht drei disabled Buttons + Hinweis', async () => {
    seedRoutes([
      trainingSession({
        my_rsvp: 'declined',
        my_rsvp_locked: true,
        am_i_participant: true,
        my_reason: 'Urlaub',
      }),
    ])
    renderPage()
    const zusagen = await screen.findByRole('button', { name: /Zusagen/ })
    expect(zusagen).toBeTruthy()
    expect((zusagen as HTMLButtonElement).disabled).toBe(true)
    const vielleicht = screen.getByRole('button', { name: /Vielleicht/ })
    expect((vielleicht as HTMLButtonElement).disabled).toBe(true)
    const absagen = screen.getByRole('button', { name: /Absagen/ })
    expect((absagen as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByText(/Durch Abwesenheit gesperrt/)).toBeTruthy()
  })

  test('Kind mit Urlaub: Kind-Buttons sind disabled + Hinweis unter Kind', async () => {
    authState.is_parent = true
    seedRoutes([
      trainingSession({
        my_rsvp: null,
        am_i_participant: false,
        children_rsvp: [
          { member_id: 42, name: 'Anna', rsvp: 'declined', reason: 'Krank', rsvp_locked: true },
        ],
      }),
    ])
    renderPage()
    await waitFor(() => expect(screen.getByText('Anna')).toBeTruthy())
    const zusagen = screen.getByRole('button', { name: /Zusagen/ })
    expect((zusagen as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByText(/Durch Abwesenheit gesperrt/)).toBeTruthy()
  })
})

describe('TerminePage — Karte anklicken setzt Focus Parameter', () => {
  beforeEach(() => {
    mockGet.mockReset()
    authState.is_parent = false
  })

  // Nutzt einen echten Data-Router (statt der einfachen MemoryRouter-Hülle von
  // renderPage()), damit wir per router.subscribe() die tatsächliche History-
  // Sequenz beobachten können: erst ein replace auf /termine (mit focus=…),
  // danach ein push zur Detailseite. Ein reines useLocation()-Snapshot würde
  // wegen React-18-Batching nur den finalen Zustand zeigen.
  function renderWithHistoryTracking(initialPath: string) {
    const router = createMemoryRouter(
      [
        { path: '/termine', element: <TerminePage /> },
        { path: '/termine/training/:id', element: <div data-testid="detail-training" /> },
        { path: '/termine/spiel/:id', element: <div data-testid="detail-spiel" /> },
        { path: '/termine/ereignis/:id', element: <div data-testid="detail-ereignis" /> },
      ],
      { initialEntries: [initialPath] },
    )
    const locations: string[] = [router.state.location.pathname + router.state.location.search]
    router.subscribe(state => {
      locations.push(state.location.pathname + state.location.search)
    })
    render(<RouterProvider router={router} />)
    return { locations }
  }

  test('Klick auf Training-Karte setzt focus=training-<id> auf /termine, bevor zur Detailseite navigiert wird', async () => {
    const trainingId = 100
    seedRoutes([trainingSession({ id: trainingId })])
    const user = userEvent.setup()
    const { locations } = renderWithHistoryTracking('/termine')

    await waitFor(() => expect(document.getElementById(`termin-training-${trainingId}`)).toBeTruthy())
    const card = document.getElementById(`termin-training-${trainingId}`)!
    await user.click(card)

    await waitFor(() => expect(screen.getByTestId('detail-training')).toBeTruthy())

    const focusIdx = locations.findIndex(l => l.startsWith('/termine?') && l.includes(`focus=training-${trainingId}`))
    const detailIdx = locations.findIndex(l => l === `/termine/training/${trainingId}`)
    expect(focusIdx).toBeGreaterThanOrEqual(0)
    expect(detailIdx).toBeGreaterThan(focusIdx)
  })
})
