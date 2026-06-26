/**
 * DocumentsPage — Klick auf Datei navigiert zur In-App-Viewer-Route
 * (kein window.open mehr — vorher PWA-Standalone-Sackgasse).
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { render } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import { AuthContext, type AuthCtx, type User } from '../../contexts/AuthContext'
import DocumentsPage from '../DocumentsPage'

const ADMIN_USER: User = {
  id: 1,
  email: 'a@b.de',
  role: 'admin',
  clubFunctions: [],
  isParent: false,
}

const ADMIN_CTX: AuthCtx = {
  user: ADMIN_USER,
  loading: false,
  impersonating: null,
  mapsProvider: 'auto',
  setMapsProvider: () => {},
  capabilities: ['manage_documents'],
  hasCapability: (cap: string) => cap === 'manage_documents',
  navRoutes: ['/dokumente'],
  passwordChangeRecommended: false,
  dismissPasswordChangeHint: () => {},
  login: async () => {},
  logout: async () => {},
  startImpersonation: async () => {},
  stopImpersonation: async () => {},
}

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  mock.onGet('/folders').reply(200, [])
  mock.onGet('/folders/9/contents').reply(200, {
    folders: [],
    files: [{
      id: 42,
      name: 'Satzung.pdf',
      size: 1024,
      mime_type: 'application/pdf',
      uploaded_by_name: 'admin',
      created_at: '2026-01-01T00:00:00Z',
    }],
    can_read: true,
    can_write: true,
  })
})

afterEach(() => {
  mock.restore()
})

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="path">{loc.pathname}</div>
}

describe('DocumentsPage.openFile', () => {
  test('Klick auf Datei → navigate(/dokumente/anzeigen/:id), kein window.open', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)

    render(
      <AuthContext.Provider value={ADMIN_CTX}>
        <MemoryRouter initialEntries={['/dokumente/9']}>
          <Routes>
            <Route path="/dokumente/:folderId" element={<DocumentsPage />} />
            <Route path="/dokumente/anzeigen/:fileId" element={<LocationProbe />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>,
    )

    // Mobile + Desktop-Layout sind beide im DOM (keine Media-Queries in jsdom).
    // Es reicht, eine Instanz anzuklicken; beide Layouts rufen openFile auf.
    const matches = await screen.findAllByText('Satzung.pdf')
    fireEvent.click(matches[0])

    await waitFor(() => {
      expect(screen.getByTestId('path').textContent).toBe('/dokumente/anzeigen/42')
    })
    expect(openSpy).not.toHaveBeenCalled()
    openSpy.mockRestore()
  })
})
