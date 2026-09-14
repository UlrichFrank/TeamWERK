import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import { render } from '@testing-library/react'
import FileViewer from '../FileViewer'
import { HistoryBaseContext } from '../../hooks/useGoBack'

// pdfjs-dist ist nicht in jsdom testbar (Worker). PdfRenderer wird sowieso lazy
// geladen, also stubben wir den dynamischen Import.
vi.mock('../PdfRenderer', () => ({
  default: ({ blob }: { blob: Blob }) => (
    <div data-testid="pdf-renderer-stub">PDF[{blob.size}]</div>
  ),
}))

beforeEach(() => {
  // jsdom kennt URL.createObjectURL nicht durchgängig.
  URL.createObjectURL = vi.fn(() => 'blob:mock')
  URL.revokeObjectURL = vi.fn()
})

describe('FileViewer (blob source)', () => {
  test('rendert Bild als <img> mit Blob-URL', () => {
    const blob = new Blob(['xxxx'], { type: 'image/png' })
    render(
      <MemoryRouter>
        <FileViewer source="blob" blob={blob} filename="foto.png" mimeType="image/png" fallbackPath="/" />
      </MemoryRouter>,
    )
    const img = screen.getByAltText('foto.png') as HTMLImageElement
    expect(img.src).toBe('blob:mock')
  })

  test('rendert PDF via lazy-PdfRenderer (gestubbt)', async () => {
    const blob = new Blob(['xxxx'], { type: 'application/pdf' })
    render(
      <MemoryRouter>
        <FileViewer source="blob" blob={blob} filename="datei.pdf" mimeType="application/pdf" fallbackPath="/" />
      </MemoryRouter>,
    )
    expect(await screen.findByTestId('pdf-renderer-stub')).toBeInTheDocument()
  })

  test('zeigt Download-Fallback für unbekannten MIME-Type', () => {
    const blob = new Blob(['xxxx'], { type: 'application/octet-stream' })
    render(
      <MemoryRouter>
        <FileViewer source="blob" blob={blob} filename="data.bin" mimeType="application/octet-stream" fallbackPath="/" />
      </MemoryRouter>,
    )
    expect(screen.getByText(/kann nicht in der App angezeigt werden/i)).toBeInTheDocument()
    const dl = screen.getByText('Herunterladen').closest('a') as HTMLAnchorElement
    expect(dl.getAttribute('download')).toBe('data.bin')
  })

  test('Zurück-Button → navigate(-1) wenn History vorhanden, sonst fallbackPath', () => {
    function LocationProbe() {
      const loc = useLocation()
      return <div data-testid="path">{loc.pathname}</div>
    }
    const blob = new Blob(['x'], { type: 'image/png' })
    render(
      <MemoryRouter initialEntries={['/dokumente/anzeigen/1']}>
        <Routes>
          <Route
            path="/dokumente/anzeigen/:id"
            element={
              <FileViewer source="blob" blob={blob} filename="x.png" mimeType="image/png" fallbackPath="/dokumente" />
            }
          />
          <Route path="/dokumente" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByLabelText('Zurück'))
    // Leerer History-Stack im MemoryRouter (nur 1 Eintrag) → Fallback greift.
    expect(screen.getByTestId('path').textContent).toBe('/dokumente')
  })
})

describe('FileViewer — Zurück nach Deep-Link', () => {
  function LocationProbe() {
    const loc = useLocation()
    return <div data-testid="path">{loc.pathname}</div>
  }

  function renderViewer(base: number) {
    const blob = new Blob(['x'], { type: 'image/png' })
    render(
      <HistoryBaseContext.Provider value={base}>
        <MemoryRouter initialEntries={['/termine/spiel/5', '/dokumente/anzeigen/1']} initialIndex={1}>
          <Routes>
            <Route
              path="/dokumente/anzeigen/:id"
              element={<FileViewer source="blob" blob={blob} filename="x.png" mimeType="image/png" fallbackPath="/dokumente" />}
            />
            <Route path="*" element={<LocationProbe />} />
          </Routes>
        </MemoryRouter>
      </HistoryBaseContext.Provider>,
    )
  }

  afterEach(() => window.history.replaceState(null, ''))

  test('history.length > 1, aber kein In-App-Vorgänger → Fallback statt App verlassen', () => {
    // Per Push/Deep-Link geöffnet: fremde Einträge davor, App-Index = Sitzungsstart.
    window.history.pushState(null, '', '/vorher')
    window.history.replaceState({ idx: 3 }, '')
    expect(window.history.length).toBeGreaterThan(1)
    renderViewer(3)
    fireEvent.click(screen.getByLabelText('Zurück'))
    expect(screen.getByTestId('path').textContent).toBe('/dokumente')
  })

  test('In-App-Vorgänger vorhanden → navigate(-1) zurück zum Termin', () => {
    window.history.replaceState({ idx: 4 }, '')
    renderViewer(3)
    fireEvent.click(screen.getByLabelText('Zurück'))
    expect(screen.getByTestId('path').textContent).toBe('/termine/spiel/5')
  })
})
