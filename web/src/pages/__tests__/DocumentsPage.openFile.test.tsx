/**
 * DocumentsPage — Klick auf Datei:
 *  - Desktop → In-App-Viewer-Route (kein window.open, vorher PWA-Sackgasse).
 *  - Mobile  → direkt nativer Viewer (Blob-Download), keine In-App-Render-Seite.
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

  test('Mobile: Klick auf Datei → nativer Viewer (Blob-Download), keine In-App-Render-Seite', async () => {
    // Mobile-Breakpoint simulieren.
    const mqSpy = vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
      matches: query.includes('639'),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as unknown as MediaQueryList)

    // Object-URL + Anchor-Klick in jsdom mocken.
    const createUrlSpy = vi
      .spyOn(URL, 'createObjectURL')
      .mockReturnValue('blob:mock-url')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => {})

    mock.onGet('/files/42/download-token').reply(200, { token: 'tok123' })
    mock
      .onGet('/files/42/download?token=tok123')
      .reply(200, new Blob(['%PDF-1.4'], { type: 'application/pdf' }))

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

    const matches = await screen.findAllByText('Satzung.pdf')
    fireEvent.click(matches[0])

    await waitFor(() => expect(clickSpy).toHaveBeenCalled())
    expect(createUrlSpy).toHaveBeenCalled()
    // Es darf NICHT zur In-App-Render-Route navigiert worden sein.
    expect(screen.queryByTestId('path')).toBeNull()

    mqSpy.mockRestore()
    createUrlSpy.mockRestore()
    clickSpy.mockRestore()
  })
})
