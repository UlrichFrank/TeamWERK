import { describe, test, expect, vi, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import VideoUploadPage from '../VideoUploadPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { api } from '../../lib/api'

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

  test('blockiert Dateien über 2 GB', async () => {
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
      mocks: [{ url: /\/teams/, data: TEAMS }],
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
      size_bytes: 1024 * 1024,
    }))

    await waitFor(() => expect(tusStart).toHaveBeenCalled())
    // Der echte Upload ist der letzte tus.Upload-Aufruf (vorher läuft ein
    // findPreviousUploads-Probe-Upload ohne Metadaten).
    const calls = UploadMock.mock.calls
    const opts = calls[calls.length - 1][1] as { metadata: Record<string, string> }
    expect(opts.metadata.video_id).toBe('42')
  })
})
