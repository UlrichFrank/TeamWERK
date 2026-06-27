/**
 * DocumentFileLinkPage (Deep-Link `/dokumente/datei/:fileId`):
 *  - Mobile  → <Navigate replace> auf In-App-Viewer-Route.
 *  - Desktop → location.replace auf Download-URL (nativer Browser-Viewer).
 *  - Desktop + 403 → Fehler-UI mit Zurück-Link.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { render } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import DocumentFileLinkPage from '../DocumentFileLinkPage'

function mockMatchMedia(matches: boolean) {
  return vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
    matches,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }) as unknown as MediaQueryList)
}

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
})

afterEach(() => {
  mock.restore()
  vi.restoreAllMocks()
})

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="path">{loc.pathname}</div>
}

describe('DocumentFileLinkPage', () => {
  test('Mobile: rendert <Navigate replace> auf /dokumente/anzeigen/:fileId', async () => {
    mockMatchMedia(true)

    render(
      <MemoryRouter initialEntries={['/dokumente/datei/42']}>
        <Routes>
          <Route path="/dokumente/datei/:fileId" element={<DocumentFileLinkPage />} />
          <Route path="/dokumente/anzeigen/:fileId" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('path').textContent).toBe('/dokumente/anzeigen/42')
    })
  })

  test('Desktop: holt Token und ruft window.location.replace mit Download-URL', async () => {
    mockMatchMedia(false)
    mock.onGet('/files/42/download-token').reply(200, { token: 'tok-dl' })

    const replaceSpy = vi.fn()
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, replace: replaceSpy },
    })

    render(
      <MemoryRouter initialEntries={['/dokumente/datei/42']}>
        <Routes>
          <Route path="/dokumente/datei/:fileId" element={<DocumentFileLinkPage />} />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(replaceSpy).toHaveBeenCalledWith('/api/files/42/download?token=tok-dl')
    })
  })

  test('Desktop + 403: zeigt Fehler-UI mit Zurück-Link', async () => {
    mockMatchMedia(false)
    mock.onGet('/files/42/download-token').reply(403)

    render(
      <MemoryRouter initialEntries={['/dokumente/datei/42']}>
        <Routes>
          <Route path="/dokumente/datei/:fileId" element={<DocumentFileLinkPage />} />
        </Routes>
      </MemoryRouter>,
    )

    expect(await screen.findByText(/keinen Zugriff/i)).toBeInTheDocument()
    expect(screen.getByText('Zurück zu Dokumente')).toBeInTheDocument()
  })
})
