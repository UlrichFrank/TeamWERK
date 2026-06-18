import { act, render, type RenderResult } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import type { ReactNode } from 'react'
import { AuthContext, type AuthCtx, type User } from '../contexts/AuthContext'
import { type Persona, personaById } from './personas'
import { setupApiMock, type MockEntry } from './apiMock'

/**
 * Flush pending microtasks and effect-driven state updates.
 * Use after renderAsPersona() in tests whose component triggers fetches in useEffect —
 * otherwise React emits "not wrapped in act(...)" warnings when those promises resolve
 * after the test has already asserted.
 */
export async function flushAsync(): Promise<void> {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0))
  })
}

function personaToUser(p: Persona): User {
  return {
    id: 1,
    email: `${p.id}@test.local`,
    role: p.role,
    clubFunctions: p.clubFunctions,
    isParent: p.isParent,
  }
}

// Mirror of internal/policy.Capabilities — keep in sync with rules.go.
function personaCapabilities(p: Persona): string[] {
  const cf = p.clubFunctions
  const isAdmin = p.role === 'admin'
  const isVorstandLike = isAdmin || cf.includes('vorstand')
  const isTrainerLike = isAdmin || cf.includes('trainer') || cf.includes('sportliche_leitung')
  const caps: string[] = []
  if (isVorstandLike) {
    caps.push('manage_members', 'manage_games', 'manage_duties', 'manage_kader',
      'manage_users', 'manage_seasons', 'manage_club', 'manage_duty_types')
  } else if (isTrainerLike) {
    caps.push('manage_games', 'manage_duties', 'manage_kader')
  }
  if (isTrainerLike) caps.push('manage_trainings', 'fulfill_duties')
  if (isVorstandLike || cf.includes('trainer') || cf.includes('sportliche_leitung')) caps.push('broadcast_messages')
  if (isVorstandLike) caps.push('broadcast_all')
  if (isAdmin) caps.push('impersonate', 'manage_documents', 'moderate_chat')
  return caps
}

// Mirror of internal/policy.NavFor — keep route list in sync with rules.go.
function personaNavRoutes(p: Persona): string[] {
  const cf = p.clubFunctions
  const isAdmin = p.role === 'admin'
  const isVorstandLike = isAdmin || cf.includes('vorstand')
  const isTrainerLike = isAdmin || cf.includes('trainer') || cf.includes('sportliche_leitung')
  const routes = ['/']
  if (!isAdmin) routes.push('/profil')
  routes.push('/kalender', '/termine', '/mein-team', '/dokumente', '/dienste', '/mitfahrgelegenheiten', '/chat')
  if (isTrainerLike || isVorstandLike) routes.push('/kader')
  if (isVorstandLike) routes.push('/nutzer', '/mitglieder', '/diensttypen', '/dienstplan-vorlagen', '/veranstaltungsorte', '/einstellungen')
  return routes
}

function makeCtx(user: User, capabilities: string[], navRoutes: string[]): AuthCtx {
  return {
    user,
    loading: false,
    impersonating: null,
    mapsProvider: 'auto',
    setMapsProvider: () => {},
    capabilities,
    hasCapability: (cap: string) => capabilities.includes(cap),
    navRoutes,
    login: async () => {},
    logout: async () => {},
    startImpersonation: async () => {},
    stopImpersonation: async () => {},
  }
}

interface RenderOptions {
  route?: string
  initialEntries?: string[]
  /** Extra API mock entries registered before the catch-all. */
  mocks?: MockEntry[]
}

export function renderAsPersona(
  ui: ReactNode,
  personaId: string,
  options: RenderOptions = {},
): RenderResult {
  const { route = '/', initialEntries, mocks } = options
  setupApiMock(mocks)
  const persona = personaById(personaId)
  const ctx = makeCtx(personaToUser(persona), personaCapabilities(persona), personaNavRoutes(persona))

  return render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter initialEntries={initialEntries ?? [route]}>
        {ui}
      </MemoryRouter>
    </AuthContext.Provider>,
  )
}

export function renderAsPersonaNoRouter(
  ui: ReactNode,
  personaId: string,
  options: Pick<RenderOptions, 'mocks'> = {},
): RenderResult {
  setupApiMock(options.mocks)
  const persona = personaById(personaId)
  const ctx = makeCtx(personaToUser(persona), personaCapabilities(persona), personaNavRoutes(persona))

  return render(
    <AuthContext.Provider value={ctx}>
      {ui}
    </AuthContext.Provider>,
  )
}
