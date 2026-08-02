/**
 * DocumentsPage — „Neuer Ordner" auf oberster Ebene.
 *
 * Auf `/dokumente` (kein currentFolderId) gibt es keinen Eltern-Ordner, dessen ACL
 * das Schreiben erlauben könnte. Sichtbarkeit hängt dort allein an der Capability
 * `create_root_folder` (Admin + Vorstand) — nicht an `manage_documents` (Admin only).
 */
import { describe, test, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import { AuthContext, type AuthCtx, type User } from '../../contexts/AuthContext'
import DocumentsPage from '../DocumentsPage'

function ctxWithCaps(caps: string[], clubFunctions: string[]): AuthCtx {
  const user: User = { id: 1, email: 'v@b.de', role: 'standard', clubFunctions, isParent: false }
  return {
    user,
    loading: false,
    impersonating: null,
    mapsProvider: 'auto',
    setMapsProvider: () => {},
    capabilities: caps,
    hasCapability: (cap: string) => caps.includes(cap),
    navRoutes: ['/dokumente'],
    passwordChangeRecommended: false,
    dismissPasswordChangeHint: () => {},
    login: async () => {},
    logout: async () => {},
    startImpersonation: async () => {},
    stopImpersonation: async () => {},
  }
}

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  mock.onGet('/folders').reply(200, [])
})

afterEach(() => {
  mock.restore()
})

function renderRoot(ctx: AuthCtx) {
  render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter initialEntries={['/dokumente']}>
        <Routes>
          <Route path="/dokumente" element={<DocumentsPage />} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  )
}

describe('DocumentsPage — Ordner auf oberster Ebene', () => {
  test('Vorstand (create_root_folder, kein manage_documents) sieht „Neuer Ordner"', async () => {
    renderRoot(ctxWithCaps(['create_root_folder'], ['vorstand']))
    expect(await screen.findByText('Neuer Ordner')).toBeInTheDocument()
  })

  test('Trainer ohne das Recht sieht „Neuer Ordner" nicht', async () => {
    renderRoot(ctxWithCaps(['manage_games'], ['trainer']))
    await waitFor(() => expect(screen.getByText('Dokumente')).toBeInTheDocument())
    expect(screen.queryByText('Neuer Ordner')).not.toBeInTheDocument()
  })
})
