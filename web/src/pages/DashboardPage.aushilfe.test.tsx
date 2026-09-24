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
  mySlots: [{ slotId: 41, dutyTypeName: 'Kampfgericht', eventTime: '13:30', teamLabel: 'mC1' }],
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
        mySlots: [{ slotId: 42, date: '2099-09-20', eventTime: '10:00', dutyTypeName: 'Bewirtung', label: 'SG Weinstadt', teamLabel: 'mB1' }],
        nextGame: { id: 2, date: '2099-09-27', opponent: 'HC Oppenweiler' },
        teamLabel: 'mB1',
        openSlotsCount: 3,
      },
      dutyAccountAushilfe: [],
    })
    renderDashboard()

    const block = await screen.findByTestId('aushilfe-block')
    // Keine Abschnitts-Überschrift mehr — das Kennzeichen steht hinter dem Titel.
    expect(block).not.toHaveTextContent('erw. Kader')
    const row = within(block).getByText('Bewirtung').closest('a')!
    expect(row).toHaveTextContent('Aushilfe')
    expect(row).toHaveTextContent('SG Weinstadt · mB1 · 10:00')
    expect(row).toHaveAttribute('href', '/dienste?focus=slot-42')
    expect(within(block).getByText('3 offene Dienste verfügbar').closest('a')).toHaveAttribute('href', '/dienste?focus=game-2')
    expect(block).toHaveTextContent('3 offene Dienste verfügbar')
    expect(block).toHaveTextContent('HC Oppenweiler · mB1')
    // Stamm-Zeile: Sprung auf den Slot, Mannschaft wie bei der Aushilfe, kein Kennzeichen.
    const stamm = screen.getByText('Kampfgericht').closest('a')!
    expect(stamm).toHaveAttribute('href', '/dienste?focus=slot-41')
    expect(stamm).toHaveTextContent('TV Bittenfeld · mC1 · 13:30')
    expect(stamm).not.toHaveTextContent('Aushilfe')
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
