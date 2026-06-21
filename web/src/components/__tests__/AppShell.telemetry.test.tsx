import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, act } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { AuthContext, type AuthCtx, type User } from '../../contexts/AuthContext'
import AppShell from '../AppShell'
import { setupApiMock } from '../../test/apiMock'

vi.mock('../../hooks/usePushSubscription', () => ({ usePushSubscription: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../contexts/VersionContext', () => ({
  useVersion: () => ({ version: null, updateAvailable: false, latestVersion: null }),
  VersionProvider: ({ children }: { children: React.ReactNode }) => children,
}))

const telemetrySpies = vi.hoisted(() => ({
  setChannelDimension: vi.fn(),
  setTeamSlugDimension: vi.fn(),
  setRoleDimension: vi.fn(),
  trackPageview: vi.fn(),
}))

vi.mock('../../lib/telemetry', () => ({
  setChannelDimension: telemetrySpies.setChannelDimension,
  setTeamSlugDimension: telemetrySpies.setTeamSlugDimension,
  setRoleDimension: telemetrySpies.setRoleDimension,
  trackPageview: telemetrySpies.trackPageview,
  slugifyTeam: (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, ''),
}))

function flush() {
  return act(async () => {
    await new Promise(r => setTimeout(r, 0))
  })
}

function makeCtx(opts: { loading?: boolean; user?: User | null }): AuthCtx {
  const { loading = false, user = { id: 1, email: 'u@x.test', role: 'standard', clubFunctions: [], isParent: false } } = opts
  return {
    user,
    loading,
    impersonating: null,
    mapsProvider: 'auto',
    setMapsProvider: () => {},
    capabilities: [],
    hasCapability: () => false,
    navRoutes: ['/'],
    login: async () => {},
    logout: async () => {},
    startImpersonation: async () => {},
    stopImpersonation: async () => {},
  }
}

function renderWithCtx(ctx: AuthCtx, route = '/') {
  return render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/" element={<AppShell />}>
            <Route index element={<div>Home</div>} />
            <Route path="profil" element={<div>Profil</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  )
}

describe('AppShell — Matomo Tracking', () => {
  beforeEach(() => {
    telemetrySpies.setChannelDimension.mockClear()
    telemetrySpies.setTeamSlugDimension.mockClear()
    telemetrySpies.setRoleDimension.mockClear()
    telemetrySpies.trackPageview.mockClear()
  })

  test('sendet Pageview mit allen Custom Dimensions nach Mount', async () => {
    setupApiMock([{ url: '/teams/my', data: [{ id: 1, name: 'H1', isExtended: false }] }])
    renderWithCtx(makeCtx({}))
    await flush()
    expect(telemetrySpies.setChannelDimension).toHaveBeenCalled()
    expect(telemetrySpies.setRoleDimension).toHaveBeenCalledWith('standard')
    expect(telemetrySpies.setTeamSlugDimension).toHaveBeenCalledWith('h1')
    expect(telemetrySpies.trackPageview).toHaveBeenCalled()
  })

  test('während loading === true wird NICHT getrackt', async () => {
    setupApiMock()
    renderWithCtx(makeCtx({ loading: true }))
    await flush()
    expect(telemetrySpies.trackPageview).not.toHaveBeenCalled()
  })

  test('ohne User wird NICHT getrackt', async () => {
    setupApiMock()
    renderWithCtx(makeCtx({ user: null }))
    await flush()
    expect(telemetrySpies.trackPageview).not.toHaveBeenCalled()
  })

  test('Team-Slug none, wenn Nutzer keinem Team angehört', async () => {
    setupApiMock([{ url: '/teams/my', data: [] }])
    renderWithCtx(makeCtx({}))
    await flush()
    expect(telemetrySpies.setTeamSlugDimension).toHaveBeenCalledWith('none')
  })

  test('Team-Slug mixed, wenn Nutzer in mehreren Teams ist', async () => {
    setupApiMock([{ url: '/teams/my', data: [{ id: 1, name: 'H1' }, { id: 2, name: 'F-Jugend' }] }])
    renderWithCtx(makeCtx({}))
    await flush()
    expect(telemetrySpies.setTeamSlugDimension).toHaveBeenCalledWith('mixed')
  })

  test('admin-Rolle wird als admin getrackt', async () => {
    setupApiMock([{ url: '/teams/my', data: [] }])
    const adminUser: User = { id: 2, email: 'a@x.test', role: 'admin', clubFunctions: [], isParent: false }
    renderWithCtx(makeCtx({ user: adminUser }))
    await flush()
    expect(telemetrySpies.setRoleDimension).toHaveBeenCalledWith('admin')
  })
})
