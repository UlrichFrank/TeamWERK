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

function makeCtx(user: User): AuthCtx {
  return {
    user,
    loading: false,
    impersonating: null,
    mapsProvider: 'auto',
    setMapsProvider: () => {},
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
  const ctx = makeCtx(personaToUser(persona))

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
  const ctx = makeCtx(personaToUser(persona))

  return render(
    <AuthContext.Provider value={ctx}>
      {ui}
    </AuthContext.Provider>,
  )
}
