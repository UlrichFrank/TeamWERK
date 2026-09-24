import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from './DashboardPage'

// openspec/changes/dienste-erweiterter-kader: „Meine Dienste“ zeigt Aushilfe im
// erweiterten Kader als eigenen Abschnitt, die Bilanz trennt Aushilfe ohne Ziel.

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

const BASE = { currentSeason: null, meineTermine: [], carpoolingConfirmed: [], carpoolingOpenGroups: [], events: [] }

function seed(meineDienste: unknown) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/dashboard')) return Promise.resolve({ data: { ...BASE, meineDienste } })
    return Promise.resolve({ data: [] })
  })
}

function renderDashboard() {
  render(<MemoryRouter initialEntries={['/']}><DashboardPage /></MemoryRouter>)
}

const STAMM = {
  nextGame: { id: 1, date: '2099-09-26', opponent: 'TV Bittenfeld' },
  mySlots: [{ dutyTypeName: 'Kampfgericht', eventTime: '13:30' }],
  openSlotsCount: 0,
  dutyAccount: [{ memberId: 11, name: 'Lena Beispiel', teamId: 5, teamLabel: 'mC1', geleistet: 4, vorhersage: 1, soll: 6 }],
  recentAssignments: [],
}

beforeEach(() => {
  mockGet.mockReset()
})

describe('DashboardPage — Aushilfe im erweiterten Kader', () => {
  test('Aushilfe-Block zeigt eigene Zusage und offene Dienste des erweiterten Teams', async () => {
    seed({
      ...STAMM,
      aushilfe: {
        mySlots: [{ date: '2099-09-20', eventTime: '10:00', dutyTypeName: 'Bewirtung', label: 'SG Weinstadt', teamLabel: 'mB1' }],
        nextGame: { id: 2, date: '2099-09-27', opponent: 'HC Oppenweiler' },
        teamLabel: 'mB1',
        openSlotsCount: 3,
      },
      dutyAccountAushilfe: [],
    })
    renderDashboard()

    const block = await screen.findByTestId('aushilfe-block')
    expect(block).toHaveTextContent('Aushilfe (erw. Kader)')
    expect(within(block).getByText('Bewirtung')).toBeInTheDocument()
    expect(block).toHaveTextContent('3 offene Dienste zum Aushelfen')
    expect(block).toHaveTextContent('HC Oppenweiler · mB1')
    // Stamm-Block bleibt oberhalb unverändert.
    expect(screen.getByText('Kampfgericht')).toBeInTheDocument()
  })

  test('ohne aushilfe kein Abschnitt', async () => {
    seed({ ...STAMM, aushilfe: null, dutyAccountAushilfe: [] })
    renderDashboard()
    await screen.findByText('Kampfgericht')
    expect(screen.queryByTestId('aushilfe-block')).not.toBeInTheDocument()
    expect(screen.queryByTestId('bilanz-aushilfe')).not.toBeInTheDocument()
  })

  test('Bilanz zeigt Aushilfe getrennt und ohne Ziel', async () => {
    seed({
      ...STAMM,
      aushilfe: null,
      dutyAccountAushilfe: [{ memberId: 11, name: 'Lena Beispiel', teamId: 9, teamLabel: 'mB1', geleistet: 2, vorhersage: 1 }],
    })
    renderDashboard()
    const bilanz = await screen.findByTestId('bilanz-aushilfe')
    const row = within(bilanz).getByText('Lena Beispiel').closest('a')!
    expect(row).toHaveAttribute('href', '/dienste/rangliste?team=9')
    expect(row).toHaveTextContent('2 geleistet')
    expect(row).toHaveTextContent('1 eingetragen')
    expect(row).not.toHaveTextContent('Ziel')
  })
})
