import { describe, test, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from './DashboardPage'

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

// Gespeicherter teams.name trägt bei Mannschaft 1 keine Nummer; die Kachel
// „Mein Team" zeigt dieselbe Langform wie die Seite „Mein Team".
const MY_TEAMS = [
  { id: 17, name: 'C-Jugend männlich', display_long: 'C-Jugend männlich 1', isExtended: false },
  { id: 18, name: 'C-Jugend männlich 2', display_long: 'C-Jugend männlich 2', isExtended: false },
]

describe('DashboardPage — Mein-Team-Kachel', () => {
  test('nennt die Mannschaften mit Nummer am Ende', async () => {
    mockGet.mockImplementation((url: string) => {
      if (url.startsWith('/dashboard')) {
        return Promise.resolve({ data: { currentSeason: null, meineTermine: [], carpoolingConfirmed: [], carpoolingOpenGroups: [], events: [] } })
      }
      if (url.startsWith('/teams/my')) return Promise.resolve({ data: MY_TEAMS })
      return Promise.resolve({ data: [] })
    })
    render(<MemoryRouter><DashboardPage /></MemoryRouter>)

    const first = await screen.findByText('C-Jugend männlich 1')
    expect(first.closest('a')).toHaveAttribute('href', '/mein-team?team=17')
    expect(screen.getByText('C-Jugend männlich 2')).toBeInTheDocument()
    expect(screen.queryByText('C-Jugend männlich')).not.toBeInTheDocument()
  })
})
