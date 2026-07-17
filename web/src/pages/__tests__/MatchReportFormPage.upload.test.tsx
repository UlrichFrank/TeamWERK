import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, fireEvent, act } from '@testing-library/react'
import { render } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import { AuthContext, type AuthCtx } from '../../contexts/AuthContext'
import MatchReportFormPage from '../MatchReportFormPage'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../MarkdownRenderer', () => ({ default: () => null }))

// compressImage stubben: gibt File unverändert zurück — jsdom-Canvas ist
// unbrauchbar, und für die Upload-Verhaltens-Tests brauchen wir kein echtes
// Verkleinern (der eigene imageCompress.test.ts prüft das Format).
vi.mock('../../lib/imageCompress', () => ({
  compressImage: vi.fn(async (file: File) => ({ blob: file, fileName: file.name })),
}))

let mock: MockAdapter

const draft = (imageCount: number) => ({
  id: 42,
  game_id: 7,
  duty_slot_id: 3,
  author_user_id: 1,
  state: 'draft' as const,
  title: 'Test',
  home_goals: null,
  away_goals: null,
  home_goals_ht: null,
  away_goals_ht: null,
  tournament: false,
  abstract: '',
  body_md: '',
  published_url: null,
  typo3_page_uid: null,
  error_message: null,
  images: Array.from({ length: imageCount }, (_, i) => ({
    id: i + 1,
    position: i + 1,
    caption: '',
    url: `/match-reports/42/images/${i + 1}/blob`,
  })),
  photo_consent_missing: null,
})

const adminCtx: AuthCtx = {
  user: { id: 1, email: 'admin@test.local', role: 'admin', clubFunctions: [], isParent: false },
  loading: false,
  impersonating: null,
  mapsProvider: 'auto',
  setMapsProvider: () => {},
  capabilities: [],
  hasCapability: () => true,
  navRoutes: [],
  passwordChangeRecommended: false,
  dismissPasswordChangeHint: () => {},
  login: async () => {},
  logout: async () => {},
  startImpersonation: async () => {},
  stopImpersonation: async () => {},
}

async function setupWithImages(count: number, uploadReplies: Array<{ status: number; body?: unknown }> = []) {
  mock.onGet('/match-reports/42').reply(200, draft(count))
  mock.onGet(/\/match-reports\/42\/images\/\d+\/blob/).reply(200, new Blob(['stub']))
  let call = 0
  mock.onPost('/match-reports/42/images').reply(() => {
    const r = uploadReplies[call++] ?? { status: 201, body: { id: 999, position: 1, caption: '', url: '/x' } }
    return [r.status, r.body ?? {}]
  })
  const result = render(
    <AuthContext.Provider value={adminCtx}>
      <MemoryRouter initialEntries={['/spielberichte/42']}>
        <Routes>
          <Route path="/spielberichte/:id" element={<MatchReportFormPage />} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  )
  await waitFor(() => expect(screen.getByText(/Bilder \(/)).toBeInTheDocument())
  return result
}

function fileList(names: string[]): FileList {
  const arr = names.map(n => new File(['data'], n, { type: 'image/jpeg' }))
  return arr as unknown as FileList
}

function pickFiles(names: string[]) {
  const input = document.querySelector('input[type="file"]') as HTMLInputElement
  expect(input).not.toBeNull()
  Object.defineProperty(input, 'files', { value: fileList(names), configurable: true })
  fireEvent.change(input)
}

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  URL.createObjectURL = vi.fn(() => 'blob:stub')
  URL.revokeObjectURL = vi.fn()
})

afterEach(() => {
  mock.restore()
})

describe('MatchReportFormPage multi-upload', () => {
  test('lädt drei Bilder sequenziell hoch', async () => {
    await setupWithImages(0, [
      { status: 201 },
      { status: 201 },
      { status: 201 },
    ])
    await act(async () => { pickFiles(['a.jpg', 'b.jpg', 'c.jpg']) })
    await waitFor(() => {
      const posts = mock.history.post.filter(r => r.url === '/match-reports/42/images')
      expect(posts).toHaveLength(3)
    })
    // Keine Fehleranzeige
    expect(screen.queryByText(/Nicht hochgeladen/)).not.toBeInTheDocument()
  })

  test('trimmt Auswahl vor Upload wenn Cap überschritten', async () => {
    await setupWithImages(8, [{ status: 201 }, { status: 201 }])
    await act(async () => { pickFiles(['a.jpg', 'b.jpg', 'c.jpg', 'd.jpg', 'e.jpg']) })
    await waitFor(() => {
      const posts = mock.history.post.filter(r => r.url === '/match-reports/42/images')
      expect(posts).toHaveLength(2)
    })
    expect(screen.getByText(/Nur die ersten 2 Bilder werden hochgeladen/)).toBeInTheDocument()
  })

  test('rendert Upload-Button nicht wenn Cap erreicht', async () => {
    await setupWithImages(10)
    // Bilder-Header ist da, aber Button nicht
    expect(screen.getByText(/Bilder \(10\/10\)/)).toBeInTheDocument()
    expect(document.querySelector('input[type="file"]')).toBeNull()
  })

  test('sammelt Fehler pro Datei und zeigt sie an', async () => {
    await setupWithImages(0, [
      { status: 201 },
      { status: 400, body: { error: 'unsupported_mime' } },
      { status: 201 },
    ])
    await act(async () => { pickFiles(['ok1.jpg', 'bad.heic', 'ok2.jpg']) })
    await waitFor(() => {
      const posts = mock.history.post.filter(r => r.url === '/match-reports/42/images')
      expect(posts).toHaveLength(3)
    })
    expect(await screen.findByText(/Nicht hochgeladen/)).toBeInTheDocument()
    expect(screen.getByText('bad.heic')).toBeInTheDocument()
    expect(screen.getByText(/Format nicht unterstützt/)).toBeInTheDocument()
  })

  test('behandelt Netzfehler mit generischer Meldung', async () => {
    mock.onGet('/match-reports/42').reply(200, draft(0))
    mock.onPost('/match-reports/42/images').networkError()
    render(
      <AuthContext.Provider value={adminCtx}>
        <MemoryRouter initialEntries={['/spielberichte/42']}>
          <Routes>
            <Route path="/spielberichte/:id" element={<MatchReportFormPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>,
    )
    await waitFor(() => expect(screen.getByText(/Bilder \(/)).toBeInTheDocument())
    await act(async () => { pickFiles(['x.jpg']) })
    expect(await screen.findByText(/Upload fehlgeschlagen/)).toBeInTheDocument()
  })
})
