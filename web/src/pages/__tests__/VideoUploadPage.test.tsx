import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import axios from 'axios'
import VideoUploadPage from '../VideoUploadPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { api, getAccessToken, setAccessToken } from '../../lib/api'

// tus-js-client komplett mocken: kein echter Upload, nur Aufruf-Beobachtung.
const tusStart = vi.fn()
const findPrev = vi.fn().mockResolvedValue([])
const resumeFrom = vi.fn()
const UploadMock = vi.fn().mockImplementation(() => ({
  start: tusStart,
  findPreviousUploads: findPrev,
  resumeFromPreviousUpload: resumeFrom,
}))
vi.mock('tus-js-client', () => ({
  Upload: function (this: unknown, ...args: unknown[]) {
    return UploadMock(...args)
  },
}))

// File mit gefälschter Größe (Vitest/jsdom übernimmt size sonst nicht zuverlässig).
function fakeFile(name: string, size: number): File {
  const f = new File(['x'], name, { type: 'video/mp4' })
  Object.defineProperty(f, 'size', { value: size })
  return f
}

const TEAMS = [{
  id: 7, name: 'Herren 1',
  age_class: 'A', gender: 'm', team_number: 1, group_count: 1, is_active: true,
}]

const SEASONS = [
  { id: 2, is_active: false },
  { id: 3, is_active: true },
]

beforeEach(() => {
  vi.clearAllMocks()
  findPrev.mockResolvedValue([])
})

describe('VideoUploadPage', () => {
  test('rendert das Formular', async () => {
    renderAsPersona(<VideoUploadPage />, 'trainer', {
      mocks: [{ url: /\/teams/, data: TEAMS }],
    })
    await flushAsync()
    expect(screen.getByRole('heading', { name: /Video hochladen/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/Titel/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/Team/i)).toBeInTheDocument()
    // Kurz-Name aus buildTeamShortNames: gender=m, age_class=A, group_count=1 → "mA".
    expect(await screen.findByRole('option', { name: 'mA' })).toBeInTheDocument()
  })

  test('blockiert Dateien über 2,5 GB', async () => {
    renderAsPersona(<VideoUploadPage />, 'trainer', {
      mocks: [{ url: /\/teams/, data: TEAMS }],
    })
    await flushAsync()

    const big = fakeFile('big.mp4', 3 * 1024 * 1024 * 1024)
    const input = screen.getByLabelText(/Videodatei/i) as HTMLInputElement
    fireEvent.change(input, { target: { files: [big] } })

    expect(await screen.findByText(/Datei zu groß/i)).toBeInTheDocument()
    // tus-Upload (echter Upload) darf nicht gestartet worden sein.
    expect(tusStart).not.toHaveBeenCalled()
  })

  test('POST /api/videos mit size_bytes und startet tus-Upload', async () => {
    const post = vi.spyOn(api, 'post').mockResolvedValue({
      data: { video_id: 42, upload_url: '/api/videos/upload/' },
    })

    renderAsPersona(<VideoUploadPage />, 'trainer', {
      mocks: [
        { url: /\/teams/, data: TEAMS },
        { url: /\/seasons/, data: SEASONS },
      ],
    })
    await flushAsync()

    fireEvent.change(screen.getByLabelText(/Titel/i), { target: { value: 'Testspiel' } })
    fireEvent.change(screen.getByLabelText(/Team/i), { target: { value: '7' } })

    const small = fakeFile('clip.mp4', 1024 * 1024)
    fireEvent.change(screen.getByLabelText(/Videodatei/i), { target: { files: [small] } })

    fireEvent.click(screen.getByRole('button', { name: /Hochladen/i }))

    await waitFor(() => expect(post).toHaveBeenCalled())
    expect(post).toHaveBeenCalledWith('/videos', expect.objectContaining({
      title: 'Testspiel',
      team_id: 7,
      season_id: 3,
      size_bytes: 1024 * 1024,
    }))

    await waitFor(() => expect(tusStart).toHaveBeenCalled())
    // Der echte Upload ist der letzte tus.Upload-Aufruf (vorher läuft ein
    // findPreviousUploads-Probe-Upload ohne Metadaten).
    const calls = UploadMock.mock.calls
    const opts = calls[calls.length - 1][1] as { metadata: Record<string, string> }
    expect(opts.metadata.video_id).toBe('42')
  })

  test('frischer Upload resumt keine vorhandene Session (kein Hijack)', async () => {
    // Für die Datei liegt eine unterbrochene frühere Session (fremde video_id) vor.
    findPrev.mockResolvedValue([{
      metadata: { video_id: '7' },
      uploadUrl: '/api/videos/upload/abc',
      size: 1024 * 1024,
      creationTime: '2026-06-30T00:00:00Z',
      urlStorageKey: 'tus::x',
    }])
    const post = vi.spyOn(api, 'post').mockResolvedValue({
      data: { video_id: 42, upload_url: '/api/videos/upload/' },
    })

    renderAsPersona(<VideoUploadPage />, 'trainer', {
      mocks: [
        { url: /\/teams/, data: TEAMS },
        { url: /\/seasons/, data: SEASONS },
      ],
    })
    await flushAsync()

    fireEvent.change(screen.getByLabelText(/Titel/i), { target: { value: 'Testspiel' } })
    fireEvent.change(screen.getByLabelText(/Team/i), { target: { value: '7' } })

    const small = fakeFile('clip.mp4', 1024 * 1024)
    fireEvent.change(screen.getByLabelText(/Videodatei/i), { target: { files: [small] } })
    // Warten bis die Resume-Sonde den "Upload fortsetzen"-Zustand gesetzt hat.
    await screen.findByRole('button', { name: /Upload fortsetzen/i })

    fireEvent.click(screen.getByRole('button', { name: /^Hochladen$/i }))

    await waitFor(() => expect(post).toHaveBeenCalled())
    await waitFor(() => expect(tusStart).toHaveBeenCalled())

    // Der frische Upload trägt die NEUE video_id und resumt NICHT die fremde Session.
    const calls = UploadMock.mock.calls
    const opts = calls[calls.length - 1][1] as { metadata: Record<string, string> }
    expect(opts.metadata.video_id).toBe('42')
    expect(resumeFrom).not.toHaveBeenCalled()
  })
})

// Die tus-Auth-Hooks werden isoliert getestet: tus ist vollständig gemockt
// (kein echter Upload im jsdom), daher fangen wir die an `new tus.Upload(...)`
// übergebenen Optionen ab und rufen onShouldRetry/onBeforeRequest direkt auf.
interface AuthHookOpts {
  onShouldRetry: (
    err: { originalResponse?: { getStatus: () => number } | null },
    retryAttempt: number,
    options: { retryDelays?: number[] | null },
  ) => boolean
  onBeforeRequest: (req: { setHeader: (name: string, value: string) => void }) => Promise<void>
}

const RETRY_OPTS = { retryDelays: [0, 1000, 3000, 5000] }
const err401 = { originalResponse: { getStatus: () => 401 } }

// Führt den Upload-Flow durch (POST /videos → startTus → new tus.Upload) und
// liefert die Optionen des echten Uploads (letzter tus.Upload-Aufruf).
async function startUploadAndCaptureOpts(): Promise<AuthHookOpts> {
  vi.spyOn(api, 'post').mockResolvedValue({
    data: { video_id: 42, upload_url: '/api/videos/upload/' },
  })
  renderAsPersona(<VideoUploadPage />, 'trainer', {
    mocks: [
      { url: /\/teams/, data: TEAMS },
      { url: /\/seasons/, data: SEASONS },
    ],
  })
  await flushAsync()
  fireEvent.change(screen.getByLabelText(/Titel/i), { target: { value: 'Testspiel' } })
  fireEvent.change(screen.getByLabelText(/Team/i), { target: { value: '7' } })
  const small = fakeFile('clip.mp4', 1024 * 1024)
  fireEvent.change(screen.getByLabelText(/Videodatei/i), { target: { files: [small] } })
  fireEvent.click(screen.getByRole('button', { name: /Hochladen/i }))
  await waitFor(() => expect(tusStart).toHaveBeenCalled())
  const calls = UploadMock.mock.calls
  return calls[calls.length - 1][1] as unknown as AuthHookOpts
}

describe('VideoUploadPage – Token-Refresh der tus-Hooks', () => {
  let originalLocation: Location

  beforeEach(() => {
    vi.clearAllMocks()
    findPrev.mockResolvedValue([])
    setAccessToken('alt-token')
    originalLocation = window.location
    Object.defineProperty(window, 'location', {
      configurable: true,
      writable: true,
      value: { href: '' },
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      writable: true,
      value: originalLocation,
    })
    setAccessToken(null)
  })

  test('Access-Token läuft mitten im Upload ab: 401 → Refresh → Retry mit neuem Token', async () => {
    const refreshSpy = vi
      .spyOn(axios, 'post')
      .mockResolvedValue({ data: { access_token: 'neu' } })

    const opts = await startUploadAndCaptureOpts()

    // Erster PATCH liefert 401 → onShouldRetry stößt den Refresh an und retryt.
    expect(opts.onShouldRetry(err401, 0, RETRY_OPTS)).toBe(true)

    // Der Retry-Request wartet in onBeforeRequest den laufenden Refresh ab und
    // trägt den neuen Token.
    const setHeader = vi.fn()
    await opts.onBeforeRequest({ setHeader })

    // /api/auth/refresh wurde GENAU einmal aufgerufen (Single-Flight-Guard).
    expect(refreshSpy).toHaveBeenCalledTimes(1)
    expect(refreshSpy).toHaveBeenCalledWith(
      '/api/auth/refresh',
      {},
      expect.objectContaining({ withCredentials: true }),
    )
    expect(setHeader).toHaveBeenCalledWith('Authorization', 'Bearer neu')
    expect(getAccessToken()).toBe('neu')
  })

  test('Refresh-Token abgelaufen: Upload bricht sauber ab und leitet auf /login um', async () => {
    const refreshSpy = vi
      .spyOn(axios, 'post')
      .mockRejectedValue({ response: { status: 401 } })

    const opts = await startUploadAndCaptureOpts()

    // Erster 401 → onShouldRetry retryt (true) und stößt den Refresh an, der
    // seinerseits mit 401 scheitert.
    expect(opts.onShouldRetry(err401, 0, RETRY_OPTS)).toBe(true)
    await flushAsync()

    // Refresh-Fehler (401): Token gelöscht + Redirect auf /login.
    expect(refreshSpy).toHaveBeenCalledTimes(1)
    expect(getAccessToken()).toBeNull()
    expect(window.location.href).toBe('/login')

    // Ein weiterer 401 bricht jetzt sauber ab (return false → tus feuert onError),
    // statt in eine Refresh-Schleife zu laufen.
    expect(opts.onShouldRetry(err401, 1, RETRY_OPTS)).toBe(false)
  })

  test('Nicht-401-Fehler: tus-Default-Retry-Verhalten bleibt erhalten', async () => {
    const refreshSpy = vi.spyOn(axios, 'post')
    const opts = await startUploadAndCaptureOpts()

    const err503 = { originalResponse: { getStatus: () => 503 } }
    // Innerhalb der retryDelays weiter retryen, danach aufgeben.
    expect(opts.onShouldRetry(err503, 0, RETRY_OPTS)).toBe(true)
    expect(opts.onShouldRetry(err503, 3, RETRY_OPTS)).toBe(true)
    expect(opts.onShouldRetry(err503, 4, RETRY_OPTS)).toBe(false)
    // Ein transienter Fehler darf KEINEN Token-Refresh auslösen.
    expect(refreshSpy).not.toHaveBeenCalled()
  })
})
