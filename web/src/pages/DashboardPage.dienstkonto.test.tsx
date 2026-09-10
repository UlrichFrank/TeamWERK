import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from './DashboardPage'

// Teil von openspec/changes/dienste-familien-rangliste: `meineDienste.dutyAccount`
// ist jetzt eine Liste (eine Position pro Kind) statt eines einzelnen
// aggregierten Objekts. Mocking-Stil analog DutyPage.teamfilter.test.tsx:
// api direkt mocken statt axios-mock-adapter, damit die genaue Payload-Form
// (Array statt Objekt) unmissverständlich im Test steht.

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', clubFunctions: [] },
    hasCapability: () => false,
  }),
}))

const BASE_DASHBOARD = {
  currentSeason: null,
  meineTermine: [],
  carpoolingConfirmed: [],
  carpoolingOpenGroups: [],
  events: [],
}

function seedRoutes(meineDienste: unknown) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/dashboard')) return Promise.resolve({ data: { ...BASE_DASHBOARD, meineDienste } })
    if (url.startsWith('/chat/')) return Promise.resolve({ data: [] })
    if (url.startsWith('/teams/my')) return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
}

function renderDashboard() {
  render(
    <MemoryRouter initialEntries={['/']}>
      <DashboardPage />
    </MemoryRouter>,
  )
}

beforeEach(() => {
  mockGet.mockReset()
})

describe('DashboardPage — Meine-Dienste-Kachel (Familien-Rangliste)', () => {
  test('mehrere Kinder zeigen mehrere Zeilen, je mit Link auf die richtige Mannschaft', async () => {
    seedRoutes({
      nextGame: null,
      mySlots: [],
      openSlotsCount: 0,
      dutyAccount: [
        { memberId: 11, name: 'Anna Beispiel', teamId: 5, teamLabel: 'wC1', geleistet: 2, vorhersage: 1, soll: 4 },
        { memberId: 12, name: 'Ben Beispiel', teamId: 7, teamLabel: 'mB2', geleistet: 0, vorhersage: 0, soll: 3 },
      ],
      recentAssignments: [],
    })

    renderDashboard()

    const annaRow = await screen.findByText('Anna Beispiel')
    const benRow = screen.getByText('Ben Beispiel')

    expect(annaRow.closest('a')).toHaveAttribute('href', '/dienste/rangliste?team=5')
    expect(benRow.closest('a')).toHaveAttribute('href', '/dienste/rangliste?team=7')

    // Zähler-Text mit deutschem Format.
    expect(screen.getByText('2 + 1 von 4 Dienste')).toBeInTheDocument()
    expect(screen.getByText('0 + 0 von 3 Dienste')).toBeInTheDocument()
  })

  test('soll = 0 zeigt keinen Fortschrittsbalken, nur den Zähler', async () => {
    seedRoutes({
      nextGame: null,
      mySlots: [],
      openSlotsCount: 0,
      dutyAccount: [
        { memberId: 21, name: 'Chris Beispiel', teamId: 9, teamLabel: 'mA1', geleistet: 0, vorhersage: 0, soll: 0 },
      ],
      recentAssignments: [],
    })

    const { container } = render(
      <MemoryRouter initialEntries={['/']}>
        <DashboardPage />
      </MemoryRouter>,
    )

    const row = await screen.findByText('Chris Beispiel')
    expect(screen.getByText('0 Dienste')).toBeInTheDocument()
    // Kein Segment-Balken-Element (weder Geleistet- noch Vorhersage-Segment).
    expect(container.querySelectorAll('.bg-brand-green').length).toBe(0)
    expect(container.querySelectorAll('.bg-brand-info').length).toBe(0)
    expect(row.closest('a')).toHaveAttribute('href', '/dienste/rangliste?team=9')
  })

  test('leeres dutyAccount-Array zeigt keine Konto-Zeilen', async () => {
    seedRoutes({
      nextGame: null,
      mySlots: [],
      openSlotsCount: 0,
      dutyAccount: [],
      recentAssignments: [],
    })

    const { container } = render(
      <MemoryRouter initialEntries={['/']}>
        <DashboardPage />
      </MemoryRouter>,
    )

    await screen.findByText('Kein kommendes Spiel mit Diensten.')
    // Keine Rangliste-Links (= keine Konto-Zeile), unabhängig vom statischen
    // "Bisherige Dienste"/"Alle Dienste"-Text drumherum.
    expect(container.querySelectorAll('a[href^="/dienste/rangliste"]').length).toBe(0)
  })
})
