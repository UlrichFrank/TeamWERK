/**
 * Kommentar aus dem Beitrittsantrag (RequestMembershipPage) muss im Abschnitt
 * "Ausstehende Anfragen & Einladungen" auf /nutzer sichtbar sein — bislang stand
 * die Zelle hinter `hidden lg:table-cell`, einer für dieses Table-Layout
 * ungewöhnlich hohen Schwelle (1024px): auf jedem schmaleren Desktop-Fenster
 * (Sidebar + normale Fensterbreite) blieb der Kommentar unsichtbar, obwohl
 * Backend und State ihn längst mitführen. Fix: `hidden md:table-cell`, wie die
 * Nachbarspalte (E-Mail/Rolle) direkt davor.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import AdminUsersPage from '../AdminUsersPage'
import { AuthContext, type AuthCtx, type User } from '../../contexts/AuthContext'

const mockGet = vi.fn()
vi.mock('../../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
  getReference: (url: string) => mockGet(url).then((r: { data: unknown }) => r.data),
}))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const PENDING_REQUEST = {
  id: 501,
  first_name: 'Enno',
  last_name: 'Beispiel',
  email: 'enno@example.org',
  comment: 'Trainer A2, meldet sich für die neue Saison',
  status: 'pending',
  created_at: '2026-09-01T10:00:00Z',
}

const PENDING_INVITATION = {
  id: 601,
  email: 'invite@example.org',
  role: 'standard',
  comment: 'Vater von Silvester Hoen',
  expires_at: '2026-10-01T10:00:00Z',
}

function mockApi() {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/users')) return Promise.resolve({ data: { items: [], total: 0 } })
    if (url.startsWith('/membership-requests')) return Promise.resolve({ data: [PENDING_REQUEST] })
    if (url.startsWith('/invitations')) return Promise.resolve({ data: [PENDING_INVITATION] })
    return Promise.resolve({ data: [] })
  })
}

function renderPage() {
  const self: User = { id: 1, email: 'admin@example.org', role: 'admin', clubFunctions: [], isParent: false }
  const ctx = {
    user: self,
    loading: false,
    impersonating: null,
    mapsProvider: 'auto',
    setMapsProvider: () => {},
    capabilities: [],
    hasCapability: () => false,
    navRoutes: [],
    passwordChangeRecommended: false,
    dismissPasswordChangeHint: () => {},
    login: async () => {},
    logout: async () => {},
    startImpersonation: () => {},
    stopImpersonation: async () => {},
  } as unknown as AuthCtx

  render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter>
        <AdminUsersPage />
      </MemoryRouter>
    </AuthContext.Provider>,
  )
}

describe('AdminUsersPage — Kommentare in "Ausstehende Anfragen & Einladungen"', () => {
  beforeEach(() => {
    mockGet.mockReset()
  })

  test('Kommentar eines Beitrittsantrags wird angezeigt und ist nicht hinter lg: versteckt', async () => {
    mockApi()
    renderPage()

    const cell = await screen.findByText(PENDING_REQUEST.comment)
    expect(cell.closest('td')?.className).not.toMatch(/\blg:table-cell\b/)
  })

  test('Kommentar einer Einladung wird angezeigt und ist nicht hinter lg: versteckt', async () => {
    mockApi()
    renderPage()

    const cell = await screen.findByText(PENDING_INVITATION.comment)
    expect(cell.closest('td')?.className).not.toMatch(/\blg:table-cell\b/)
  })
})
