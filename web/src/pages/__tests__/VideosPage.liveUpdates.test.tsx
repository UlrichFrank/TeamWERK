import { describe, test, expect, vi, beforeEach } from 'vitest'
import type MockAdapter from 'axios-mock-adapter'
import { screen, fireEvent } from '@testing-library/react'
import VideosPage from '../VideosPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

// useLiveUpdates ist String-only; wir fangen die registrierte Callback-Funktion ab,
// um SSE-Events im Test gezielt auszulösen.
let liveHandler: ((event: string) => void) | null = null
vi.mock('../../hooks/useLiveUpdates', () => ({
  useLiveUpdates: (cb: (event: string) => void) => { liveHandler = cb },
}))

// Den von renderAsPersona angelegten Mock neu bestücken: dynamische
// (funktionsbasierte) /videos-Antworten in Prioritätsreihenfolge VOR dem Catch-all.
function installVideoMock(reply: (url: string) => [number, unknown]): MockAdapter {
  const mock = getApiMock()
  mock.reset()
  mock.onGet(/\/videos\?/).reply((config) => reply(config.url ?? ''))
  mock.onGet(/\/teams/).reply(200, [])
  mock.onGet('/profile/me').reply(200, { id: 1, email: 'x', name: 'x', club_functions: [], is_parent: false, children: [] })
  mock.onAny().reply(200, [])
  return mock
}

function makeVideo(id: number, over: Partial<Record<string, unknown>> = {}) {
  return {
    id,
    title: `Video ${id}`,
    description: null,
    team_id: 1,
    team_name: 'Team A',
    season_id: 1,
    game_id: null,
    status: 'ready',
    duration_sec: 90,
    created_by: 1,
    created_at: '2026-06-01T10:00:00Z',
    ready_at: '2026-06-01T11:00:00Z',
    ...over,
  }
}

beforeEach(() => {
  liveHandler = null
})

const videoCalls = (mock: MockAdapter) => mock.history.get.filter(c => /\/videos\?/.test(c.url ?? ''))

describe('VideosPage — keeps_loaded_pages_on_sse_event', () => {
  test('video-updated setzt NICHT auf Seite 0 zurück, patcht per ID und behält den Bestand', async () => {
    renderAsPersona(<VideosPage />, 'vorstand')
    // Mock NACH renderAsPersona installieren (überschreibt dessen setupApiMock),
    // aber VOR flushAsync — der Mount-Fetch läuft erst im flushAsync.
    const mock = installVideoMock((url) => {
      if (/offset=1\b/.test(url)) return [200, { items: [makeVideo(2)], total: 2 }]
      if (/limit=2\b/.test(url)) return [200, { items: [makeVideo(1), makeVideo(2, { title: 'Video 2 – neu' })], total: 2 }]
      return [200, { items: [makeVideo(1)], total: 2 }]
    })

    await flushAsync()
    expect(await screen.findByText('Video 1')).toBeInTheDocument()

    // Zweite Seite nachladen → Bestand [1,2].
    fireEvent.click(await screen.findByRole('button', { name: /Mehr laden/i }))
    await flushAsync()
    expect(screen.getByText('Video 2')).toBeInTheDocument()

    const before = videoCalls(mock).length

    liveHandler?.('video-updated')
    await flushAsync()

    // Beide zuvor geladenen Videos bleiben; das aktualisierte wurde per ID gepatcht.
    expect(screen.getByText('Video 1')).toBeInTheDocument()
    expect(screen.getByText('Video 2 – neu')).toBeInTheDocument()
    // Genau EIN zusätzlicher Request (Reconciliation, limit=2, offset=0) — kein Reset,
    // kein erneutes „Mehr laden".
    const added = videoCalls(mock).slice(before)
    expect(added.length).toBe(1)
    expect(added[0].url).toMatch(/limit=2/)
    expect(added[0].url).toMatch(/offset=0/)
  })

  test('video-deleted entfernt das Element per ID aus dem Bestand', async () => {
    renderAsPersona(<VideosPage />, 'vorstand')
    installVideoMock((url) => {
      if (/limit=2\b/.test(url)) return [200, { items: [makeVideo(2)], total: 1 }]
      return [200, { items: [makeVideo(1), makeVideo(2)], total: 2 }]
    })

    await flushAsync()
    expect(await screen.findByText('Video 1')).toBeInTheDocument()
    expect(screen.getByText('Video 2')).toBeInTheDocument()

    liveHandler?.('video-deleted')
    await flushAsync()

    expect(screen.queryByText('Video 1')).toBeNull()
    expect(screen.getByText('Video 2')).toBeInTheDocument()
  })

  test('video-queued zeigt „N neue"-Chip, lädt erst auf Klick nach', async () => {
    renderAsPersona(<VideosPage />, 'vorstand')
    const mock = installVideoMock(() => [200, { items: [makeVideo(1)], total: 1 }])

    await flushAsync()
    expect(await screen.findByText('Video 1')).toBeInTheDocument()

    const before = videoCalls(mock).length
    liveHandler?.('video-queued')
    liveHandler?.('video-queued')
    await flushAsync()

    // Chip zeigt die Anzahl, KEIN automatischer Refetch.
    expect(screen.getByText('2 neue Videos')).toBeInTheDocument()
    expect(videoCalls(mock).slice(before).length).toBe(0)

    // Klick lädt nach und blendet den Chip aus.
    fireEvent.click(screen.getByText('2 neue Videos'))
    await flushAsync()
    expect(screen.queryByText('2 neue Videos')).toBeNull()
    expect(videoCalls(mock).slice(before).length).toBeGreaterThan(0)
  })
})
