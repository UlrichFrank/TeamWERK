/**
 * „Testen als" für Proxy-Accounts.
 *
 * Ein Proxy-Account (`can_login = 0`) kann sich per Definition nie selbst einloggen —
 * Impersonation ist damit der einzige Weg, seine Sicht zu prüfen. Der frühere
 * `!u.proxy`-Guard blendete den Menüpunkt genau dort aus, wo er gebraucht wird, obwohl
 * `POST /api/impersonate/{id}` serverseitig nie auf `can_login` gatet (siehe
 * TestImpersonate_ProxyAccount). Geprüft wird: (1) der Menüpunkt erscheint auch für
 * Proxy-Zeilen und ruft startImpersonation mit deren user_id; (2) die übrigen Guards
 * (nur Admin, nicht man selbst, kein Admin-Ziel) bleiben scharf.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
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

const PROXY_USER = {
  id: 42,
  first_name: 'Mia',
  last_name: 'Muster',
  email: '',
  role: 'standard',
  proxy: true,
}

const ADMIN_USER = {
  id: 7,
  first_name: 'Andrea',
  last_name: 'Admin',
  email: 'andrea@example.org',
  role: 'admin',
  proxy: false,
}

function mockApi(users: unknown[]) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/users')) return Promise.resolve({ data: { items: users, total: users.length } })
    return Promise.resolve({ data: [] })
  })
}

const startImpersonation = vi.fn()

function renderPage(selfRole: string) {
  const self: User = { id: 1, email: 'self@example.org', role: selfRole, clubFunctions: [], isParent: false }
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
    startImpersonation,
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

/** Öffnet das Aktionsmenü der Zeile, in der `name` steht. */
async function openRowMenu(name: string) {
  const cell = await screen.findByText(new RegExp(name))
  const row = cell.closest('tr')
  if (!row) throw new Error(`Zeile für ${name} nicht gefunden`)
  fireEvent.click(row.querySelector('button[aria-label="Aktionen"]') as HTMLButtonElement)
}

describe('AdminUsersPage — „Testen als" bei Proxy-Accounts', () => {
  beforeEach(() => {
    mockGet.mockReset()
    startImpersonation.mockReset()
  })

  test('Admin sieht „Testen als" auch bei einem Proxy-Account und löst damit die Impersonation aus', async () => {
    mockApi([PROXY_USER])
    renderPage('admin')

    await openRowMenu('Mia')
    const action = await screen.findByRole('button', { name: 'Testen als' })
    fireEvent.click(action)

    expect(startImpersonation).toHaveBeenCalledWith(42, 'Mia Muster')
  })

  test('Nicht-Admin sieht „Testen als" auch bei einem Proxy-Account nicht', async () => {
    mockApi([PROXY_USER])
    renderPage('standard')

    await openRowMenu('Mia')
    await waitFor(() => expect(screen.getByText('Löschen')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Testen als' })).toBeNull()
  })

  test('Admin-Konten bleiben von der Impersonation ausgenommen', async () => {
    mockApi([ADMIN_USER])
    renderPage('admin')

    await openRowMenu('Andrea')
    await waitFor(() => expect(screen.getByText('Löschen')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Testen als' })).toBeNull()
  })
})
