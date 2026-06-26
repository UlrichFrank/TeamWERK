import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { render, act } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import FileViewerPage from '../FileViewerPage'

vi.mock('../../components/PdfRenderer', () => ({
  default: ({ blob }: { blob: Blob }) => <div data-testid="pdf-renderer-stub">PDF[{blob.size}]</div>,
}))

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  URL.createObjectURL = vi.fn(() => 'blob:mock')
  URL.revokeObjectURL = vi.fn()
})

afterEach(() => {
  mock.restore()
})

function renderViewer(fileId = '42') {
  return render(
    <MemoryRouter initialEntries={[`/dokumente/anzeigen/${fileId}`]}>
      <Routes>
        <Route path="/dokumente/anzeigen/:fileId" element={<FileViewerPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('FileViewerPage', () => {
  test('Happy-Path: Token holen → Blob laden → PDF rendern', async () => {
    mock.onGet('/files/42/download-token').reply(200, { token: 'tok-abc' })
    mock.onGet('/files/42/download?token=tok-abc').reply(
      200,
      new Blob(['xxxx'], { type: 'application/pdf' }),
      { 'content-disposition': 'inline; filename="report.pdf"', 'content-type': 'application/pdf' },
    )

    renderViewer()
    expect(await screen.findByTestId('pdf-renderer-stub')).toBeInTheDocument()
    expect(screen.getByText('report.pdf')).toBeInTheDocument()
  })

  test('403 → Fehler-UI', async () => {
    mock.onGet('/files/42/download-token').reply(403)
    renderViewer()
    expect(await screen.findByText(/keinen Zugriff/i)).toBeInTheDocument()
  })

  test('404 → Fehler-UI', async () => {
    mock.onGet('/files/42/download-token').reply(404)
    renderViewer()
    expect(await screen.findByText(/nicht gefunden/i)).toBeInTheDocument()
  })

  test('Ungültige fileId → Fehler-Hinweis, kein Fetch', async () => {
    renderViewer('abc')
    await act(async () => { await new Promise(r => setTimeout(r, 0)) })
    expect(screen.getByText(/Ungültige Datei-ID/)).toBeInTheDocument()
    expect(mock.history.get.length).toBe(0)
  })
})
