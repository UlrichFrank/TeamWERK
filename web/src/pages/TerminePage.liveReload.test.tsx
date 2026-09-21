import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import TerminePage from './TerminePage'

// Regression: Nach einer Zu-/Absage broadcastet der Server (`h.broadcastGame`,
// `h.broadcastSession`) und der eigene Client empfängt denselben SSE-Event.
// Lief der daraus folgende Reload nicht still, ersetzte `setLoading(true)` die
// gesamte Liste durch eine einzeilige „Laden…"-Zeile. Der Scrollcontainer
// (<main> in AppShell) kollabiert dabei auf Zeilenhöhe, der Browser klemmt
// scrollTop auf 0 — man landete nach jedem Klick wieder ganz oben.
// jsdom kennt kein Layout, prüfbar ist aber die Ursache: die Liste darf beim
// SSE-Reload nicht aus dem DOM verschwinden.

const mockGet = vi.fn()
let liveHandler: ((event: string) => void) | null = null

vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({
  useLiveUpdates: (cb: (event: string) => void) => { liveHandler = cb },
}))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: false },
    hasCapability: () => false,
  }),
}))

const SEASON = { id: 7, name: '2026/27', start_date: '2026-05-01', end_date: '2027-06-30' }

function session(overrides: Record<string, unknown> = {}) {
  return {
    id: 100,
    series_id: null,
    title: 'Athletik',
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
    declined_count: 0,
    maybe_count: 0,
    my_rsvp: null,
    am_i_participant: true,
    rsvp_default_players: 'none',
    rsvp_default_extended: 'none',
    rsvp_require_reason: 0,
    ...overrides,
  }
}

describe('TerminePage — Live-Update lädt still nach', () => {
  beforeEach(() => {
    mockGet.mockReset()
    liveHandler = null
    mockGet.mockImplementation((url: string) => {
      if (url.startsWith('/seasons/active')) return Promise.resolve({ data: SEASON })
      if (url.startsWith('/training-sessions')) return Promise.resolve({ data: [session()] })
      return Promise.resolve({ data: [] })
    })
  })

  test('Liste bleibt beim SSE-Reload gemountet — kein „Laden…"-Platzhalter', async () => {
    render(<MemoryRouter initialEntries={['/termine']}><TerminePage /></MemoryRouter>)
    await waitFor(() => expect(document.getElementById('termin-training-100')).toBeTruthy())

    // Die Antwort auf den Reload bewusst offen halten: genau in diesem Fenster
    // stand vorher der Platzhalter.
    let resolveReload: (v: unknown) => void = () => {}
    mockGet.mockImplementation((url: string) => {
      if (url.startsWith('/seasons/active')) return Promise.resolve({ data: SEASON })
      if (url.startsWith('/training-sessions')) return new Promise(res => { resolveReload = res })
      return Promise.resolve({ data: [] })
    })

    act(() => { liveHandler?.('trainings') })

    expect(screen.queryByText('Laden…')).toBeNull()
    expect(document.getElementById('termin-training-100')).toBeTruthy()

    await act(async () => { resolveReload({ data: [session()] }) })
    expect(document.getElementById('termin-training-100')).toBeTruthy()
  })
})
