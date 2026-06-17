import { render, type RenderResult } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import type { ReactNode } from 'react'
import { AuthContext, type AuthCtx, type User } from '../contexts/AuthContext'
import { type Persona, personaById } from './personas'
import { setupApiMock } from './apiMock'

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
}

export function renderAsPersona(
  ui: ReactNode,
  personaId: string,
  options: RenderOptions = {},
): RenderResult {
  setupApiMock()
  const persona = personaById(personaId)
  const ctx = makeCtx(personaToUser(persona))
  const { route = '/', initialEntries } = options

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
): RenderResult {
  setupApiMock()
  const persona = personaById(personaId)
  const ctx = makeCtx(personaToUser(persona))

  return render(
    <AuthContext.Provider value={ctx}>
      {ui}
    </AuthContext.Provider>,
  )
}
