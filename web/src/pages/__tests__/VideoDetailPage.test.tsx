import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { Routes, Route } from 'react-router-dom'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import VideoDetailPage from '../VideoDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { api } from '../../lib/api'

// useLiveUpdates öffnet eine EventSource (in jsdom nicht vorhanden) → mocken.
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

// hls.js-Attrappe: zählt Konstruktor-Aufrufe, damit die Player-Auswahl prüfbar ist.
const hlsMock = vi.hoisted(() => {
  const ctor = vi.fn()
  class Hls {
    static isSupported = () => true
    static Events = { ERROR: 'hlsError' }
    static ErrorTypes = { NETWORK_ERROR: 'networkError', MEDIA_ERROR: 'mediaError' }
    constructor(cfg: unknown) { ctor(cfg) }
    loadSource = vi.fn()
    attachMedia = vi.fn()
    on = vi.fn()
    destroy = vi.fn()
  }
  return { ctor, Hls }
})
vi.mock('hls.js', () => ({ default: hlsMock.Hls }))

// Video im Status 'queued' → kein VideoPlayer (vermeidet den dynamischen hls.js-Import).
const VIDEO = {
  id: 42, title: 'Testspiel', description: '', team_id: 7, team_name: 'Herren 1',
  season_id: 3, game_id: null as number | null, status: 'queued', duration_sec: null,
  created_by: 1, created_at: '2026-06-30T00:00:00Z', ready_at: null, failure_reason: null,
}

const GAMES = [
  { id: 5, date: '2026-03-01T00:00:00Z', opponent: 'TV Musterstadt', teams: [{ id: 7, name: 'Herren 1' }] },
  { id: 6, date: '2026-03-08T00:00:00Z', opponent: 'SV Beispiel', teams: [{ id: 99, name: 'Andere' }] },
]

function renderDetail(video: typeof VIDEO = VIDEO) {
  return renderAsPersona(
    <Routes>
      <Route path="/videos/:id" element={<VideoDetailPage />} />
    </Routes>,
    'trainer',
    {
      initialEntries: ['/videos/42'],
      mocks: [
        { url: /\/videos\/42$/, data: video },
        { url: /\/games/, data: { items: GAMES, total: GAMES.length } },
      ],
    },
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('VideoDetailPage – Player-Auswahl (AirPlay)', () => {
  const PLAY = { token: 'tok', master_url: '/api/videos/42/hls/master.m3u8' }

  function renderReady() {
    return renderAsPersona(
      <Routes>
        <Route path="/videos/:id" element={<VideoDetailPage />} />
      </Routes>,
      'trainer',
      {
        initialEntries: ['/videos/42'],
        mocks: [
          { url: /\/videos\/42\/play$/, data: PLAY },
          { url: /\/videos\/42$/, data: { ...VIDEO, status: 'ready' } },
        ],
      },
    )
  }

  test('wählt native Wiedergabe wenn canPlayType nicht leer ist', async () => {
    vi.spyOn(HTMLMediaElement.prototype, 'canPlayType').mockReturnValue('maybe')
    const { container } = renderReady()

    await waitFor(() =>
      expect(container.querySelector('video')?.getAttribute('src')).toBe('/api/videos/42/hls/master.m3u8?st=tok'),
    )
    // Kein MSE-Pfad auf Safari/WebKit — sonst kommt über AirPlay nur Ton an.
    expect(hlsMock.ctor).not.toHaveBeenCalled()
  })

  test('fällt auf hls.js zurück wenn canPlayType leer ist', async () => {
    vi.spyOn(HTMLMediaElement.prototype, 'canPlayType').mockReturnValue('')
    const { container } = renderReady()

    await waitFor(() => expect(hlsMock.ctor).toHaveBeenCalledTimes(1))
    expect(container.querySelector('video')?.getAttribute('src')).toBeNull()
  })
})

describe('VideoDetailPage – Fehlergrund', () => {
  test('unsupported_input_format wird als Hinweis auf das Tool angezeigt', async () => {
    renderAsPersona(
      <Routes>
        <Route path="/videos/:id" element={<VideoDetailPage />} />
      </Routes>,
      'trainer',
      {
        initialEntries: ['/videos/42'],
        mocks: [{ url: /\/videos\/42$/, data: { ...VIDEO, status: 'failed', failure_reason: 'unsupported_input_format' } }],
      },
    )
    expect(await screen.findByText(/Videoformat wird nicht unterstützt/)).toBeInTheDocument()
    expect(screen.queryByText('unsupported_input_format')).toBeNull()
  })
})

describe('VideoDetailPage – Spiel-Zuordnung im Edit-Modal', () => {
  test('Selector zeigt nur Spiele des Video-Teams; Ändern sendet game_id im PATCH', async () => {
    const patch = vi.spyOn(api, 'patch').mockResolvedValue({ data: {} })
    renderDetail()
    await flushAsync()

    fireEvent.click(await screen.findByRole('button', { name: /Bearbeiten/i }))

    const select = await screen.findByLabelText(/^Spiel$/i) as HTMLSelectElement
    // Nur das Spiel von Team 7 ist wählbar (Spiel 6 gehört Team 99).
    expect(screen.getByRole('option', { name: /TV Musterstadt/i })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: /SV Beispiel/i })).not.toBeInTheDocument()

    fireEvent.change(select, { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: /^Speichern$/i }))

    await waitFor(() => expect(patch).toHaveBeenCalled())
    expect(patch).toHaveBeenCalledWith('/videos/42', expect.objectContaining({ game_id: 5 }))
  })

  test('"Kein Spiel zuordnen" sendet game_id: null', async () => {
    const patch = vi.spyOn(api, 'patch').mockResolvedValue({ data: {} })
    // Video ist bereits Spiel 5 zugeordnet → Selector vorbelegt.
    renderDetail({ ...VIDEO, game_id: 5 })
    await flushAsync()

    fireEvent.click(await screen.findByRole('button', { name: /Bearbeiten/i }))
    const select = await screen.findByLabelText(/^Spiel$/i) as HTMLSelectElement
    await waitFor(() => expect(select.value).toBe('5'))

    fireEvent.change(select, { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: /^Speichern$/i }))

    await waitFor(() => expect(patch).toHaveBeenCalled())
    expect(patch).toHaveBeenCalledWith('/videos/42', expect.objectContaining({ game_id: null }))
  })
})
