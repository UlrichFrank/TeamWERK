import { describe, test, expect, vi, beforeAll, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DutyPage from './DutyPage'

// Teil von openspec/changes/zurueck-position-wiederherstellen: /dienste liest
// ?focus=slot-<id> und scrollt/hebt die passende Zeile hervor — analog zum
// bereits bestehenden Mechanismus in TerminePage.tsx.

beforeAll(() => {
  // jsdom kennt scrollIntoView nicht.
  Element.prototype.scrollIntoView = vi.fn()
})

function boardGroup(overrides: Record<string, unknown> = {}) {
  return {
    game_id: 1,
    team_ids: [1],
    team_names: ['Team A'],
    date: '2026-09-01',
    event_time: '10:00',
    opponent: 'Gegner',
    event_type: 'heim',
    label: null,
    past: false,
    slots: [
      {
        id: 555,
        duty_type: 'Kasse',
        duty_type_id: 42,
        has_instruction: true,
        event_time: '10:00',
        hours_value: 1,
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

function seedRoutes(groups: unknown[]) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/duty-board')) return Promise.resolve({ data: groups })
    if (url.startsWith('/teams')) return Promise.resolve({ data: [] })
    if (url.startsWith('/family/proxy-accounts')) return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
}

function renderPage(route: string) {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <DutyPage />
    </MemoryRouter>,
  )
}

describe('DutyPage — Fokus-Scroll (?focus=slot-<id>)', () => {
  beforeEach(() => {
    mockGet.mockReset()
  })

  test('scrollt zur fokussierten Slot-Zeile und hebt sie kurz hervor', async () => {
    seedRoutes([boardGroup()])
    renderPage('/dienste?focus=slot-555')

    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())

    const row = await waitFor(() => {
      const el = document.getElementById('duty-slot-555')
      expect(el).not.toBeNull()
      return el as HTMLElement
    })

    await waitFor(() => expect(Element.prototype.scrollIntoView).toHaveBeenCalled())
    expect(row.classList.contains('ring-2')).toBe(true)
    expect(row.classList.contains('ring-brand-yellow')).toBe(true)
  })

  test('zeigt keinen "nicht verfügbar"-Hinweis, wenn der fokussierte Slot geladen ist', async () => {
    seedRoutes([boardGroup()])
    renderPage('/dienste?focus=slot-555')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.queryByText('Dieser Dienst ist nicht verfügbar.')).toBeNull()
  })

  test('zeigt "nicht verfügbar"-Hinweis, wenn der fokussierte Slot nicht in den geladenen Daten steckt', async () => {
    seedRoutes([boardGroup()])
    renderPage('/dienste?focus=slot-999')

    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    await waitFor(() => expect(screen.getByText('Dieser Dienst ist nicht verfügbar.')).toBeTruthy())
    // Die eigentlich fokussierte Zeile existiert nicht — kein Highlight, kein Fehler.
    expect(document.getElementById('duty-slot-999')).toBeNull()
  })

  test('ohne focus-Parameter erscheint kein Hinweis und nichts wird hervorgehoben', async () => {
    seedRoutes([boardGroup()])
    renderPage('/dienste')
    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    expect(screen.queryByText('Dieser Dienst ist nicht verfügbar.')).toBeNull()
    const row = document.getElementById('duty-slot-555') as HTMLElement
    expect(row.classList.contains('ring-2')).toBe(false)
  })

  // Regression: "In Diensten öffnen" im Kalender-Modal navigierte bisher zu
  // /dienste ohne jeden Marker — focus=game-<id> muss die ganze Spiel-Gruppe
  // hervorheben, nicht nur einen Einzel-Slot.
  test('scrollt zur fokussierten Spiel-Gruppe (focus=game-<id>) und hebt sie hervor', async () => {
    seedRoutes([boardGroup({ game_id: 1 })])
    renderPage('/dienste?focus=game-1')

    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())

    const group = await waitFor(() => {
      const el = document.getElementById('duty-game-1')
      expect(el).not.toBeNull()
      return el as HTMLElement
    })

    await waitFor(() => expect(Element.prototype.scrollIntoView).toHaveBeenCalled())
    expect(group.classList.contains('ring-2')).toBe(true)
    expect(group.classList.contains('ring-brand-yellow')).toBe(true)
    expect(screen.queryByText('Dieser Dienst ist nicht verfügbar.')).toBeNull()
  })

  test('zeigt "nicht verfügbar"-Hinweis, wenn die fokussierte Spiel-Gruppe nicht existiert', async () => {
    seedRoutes([boardGroup({ game_id: 1 })])
    renderPage('/dienste?focus=game-999')

    await waitFor(() => expect(screen.getByText('Kasse')).toBeTruthy())
    await waitFor(() => expect(screen.getByText('Dieser Dienst ist nicht verfügbar.')).toBeTruthy())
  })
})
