/**
 * DocumentsPage — Klick auf Datei:
 *  - Desktop → nativer Browser-PDF-Viewer im neuen Tab (window.open + Token-URL).
 *  - Mobile  → direkt nativer Viewer (Blob-Download), keine In-App-Render-Seite.
 *
 * SEPA-Mandat bleibt aus diesen Tests raus — eigener Pfad.
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
  test('Desktop: Klick auf Datei → window.open + Token-URL (kein navigate)', async () => {
    // Desktop: matchMedia matched NICHT — `isMobile` ist false.
    const mqSpy = vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as unknown as MediaQueryList)

    const fakeTab = { location: { href: '' }, close: vi.fn() }
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => fakeTab as unknown as Window)

    mock.onGet('/files/42/download-token').reply(200, { token: 'tok-desktop' })

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

    // window.open synchron im Click (Popup-Blocker-Workaround).
    expect(openSpy).toHaveBeenCalledWith('about:blank', '_blank')
    // Nach Token-Fetch: Download-URL wird gesetzt.
    await waitFor(() => {
      expect(fakeTab.location.href).toBe('/api/files/42/download?token=tok-desktop')
    })
    // Es darf NICHT zur In-App-Render-Route navigiert worden sein.
    expect(screen.queryByTestId('path')).toBeNull()

    mqSpy.mockRestore()
    openSpy.mockRestore()
  })

  test('Desktop, Token-Fehler: tab.close() + Fehler-State', async () => {
    const mqSpy = vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as unknown as MediaQueryList)

    const fakeTab = { location: { href: '' }, close: vi.fn() }
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => fakeTab as unknown as Window)

    mock.onGet('/files/42/download-token').reply(500)

    render(
      <AuthContext.Provider value={ADMIN_CTX}>
        <MemoryRouter initialEntries={['/dokumente/9']}>
          <Routes>
            <Route path="/dokumente/:folderId" element={<DocumentsPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>,
    )

    const matches = await screen.findAllByText('Satzung.pdf')
    fireEvent.click(matches[0])

    await waitFor(() => expect(fakeTab.close).toHaveBeenCalled())
    expect(await screen.findByText(/Datei konnte nicht geöffnet werden/)).toBeInTheDocument()

    mqSpy.mockRestore()
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
