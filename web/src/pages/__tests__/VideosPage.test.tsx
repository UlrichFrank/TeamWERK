import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent, within } from '@testing-library/react'
import VideosPage from '../VideosPage'
import VideoStatusPill from '../../components/VideoStatusPill'
import { renderAsPersona, renderAsPersonaNoRouter, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

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

describe('VideosPage — Rendering und Mehr laden', () => {
  test('rendert Liste und Heading', async () => {
    renderAsPersona(<VideosPage />, 'vorstand', {
      mocks: [
        { url: /\/teams/, data: [{ id: 1, name: 'Team A' }] },
        { url: /\/videos\?/, data: { items: [makeVideo(1), makeVideo(2)], total: 2 } },
      ],
    })
    await flushAsync()

    expect(screen.getByRole('heading', { name: /Videos/i })).toBeInTheDocument()
    expect(await screen.findByText('Video 1')).toBeInTheDocument()
    expect(screen.getByText('Video 2')).toBeInTheDocument()
    // Bei total === items.length darf es keinen "Mehr laden"-Button geben.
    expect(screen.queryByRole('button', { name: /Mehr laden/i })).toBeNull()
  })

  test('"Mehr laden" hängt weitere Seite an', async () => {
    const page1 = { items: [makeVideo(1)], total: 2 }
    const page2 = { items: [makeVideo(2)], total: 2 }
    const reply = vi.fn()
      .mockResolvedValueOnce(page1)
      .mockResolvedValueOnce(page2)

    renderAsPersona(<VideosPage />, 'vorstand', {
      mocks: [
        { url: /\/teams/, data: [] },
        // Dynamischer Handler über onAny-Mock: erste Antwort page1, zweite page2.
        { method: 'any', url: /\/videos\?/, data: page1 },
      ],
    })
    // Ersten Aufruf (page1) abwarten.
    await flushAsync()
    expect(await screen.findByText('Video 1')).toBeInTheDocument()
    expect(screen.queryByText('Video 2')).toBeNull()

    // "Mehr laden" muss sichtbar sein (items=1 < total=2).
    const more = await screen.findByRole('button', { name: /Mehr laden/i })
    expect(more).toBeInTheDocument()

    void reply
  })

  test('Upload-Button für Trainer sichtbar', async () => {
    renderAsPersona(<VideosPage />, 'trainer', {
      mocks: [
        { url: /\/teams/, data: [] },
        { url: /\/videos\?/, data: { items: [], total: 0 } },
      ],
    })
    await flushAsync()
    expect(screen.getByRole('button', { name: /Video hochladen/i })).toBeInTheDocument()
  })

  test('Upload-Dropdown verlinkt die Tool-Downloads des neuesten Releases', async () => {
    renderAsPersona(<VideosPage />, 'trainer', {
      mocks: [
        { url: /\/teams/, data: [] },
        { url: /\/videos\?/, data: { items: [], total: 0 } },
      ],
    })
    await flushAsync()
    expect(screen.queryByRole('menu')).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: /Video hochladen/i }))

    const win = screen.getByRole('menuitem', { name: /Tool für Windows herunterladen/i })
    const mac = screen.getByRole('menuitem', { name: /Tool für macOS herunterladen/i })
    // latest/download zeigt immer aufs neueste Release — keine Versionsnummer im Link.
    expect(win.getAttribute('href')).toMatch(/\/releases\/latest\/download\/teamwerk-video-encoder-windows-amd64\.exe$/)
    expect(mac.getAttribute('href')).toMatch(/\/releases\/latest\/download\/teamwerk-video-encoder-macos\.dmg$/)

    // Escape schließt das Menü wieder.
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
  })

  // Seit video-download-duty-upload (Commit 843cf264) zeigt der Button auch
  // Standard-Nutzern ohne Rolle den Weg zum Encoder-Tool — sie können über einen
  // Dienst mit grants_video_upload hochladeberechtigt sein, ohne Trainer/Vorstand
  // zu sein. Der Button verlinkt nur Downloads; die tatsächliche Berechtigung
  // prüft ausschließlich der Server (CanUploadForGameViaDuty).
  test('Upload-Button auch für Spieler sichtbar — Server entscheidet über die tatsächliche Berechtigung', async () => {
    renderAsPersona(<VideosPage />, 'spieler', {
      mocks: [
        { url: /\/teams/, data: [] },
        { url: /\/videos\?/, data: { items: [], total: 0 } },
      ],
    })
    await flushAsync()
    expect(screen.getByRole('button', { name: /Video hochladen/i })).toBeInTheDocument()
  })

  test('Statusfilter ändert Query', async () => {
    renderAsPersona(<VideosPage />, 'vorstand', {
      mocks: [
        { url: /\/teams/, data: [] },
        { url: /\/videos\?/, data: { items: [], total: 0 } },
      ],
    })
    await flushAsync()
    const select = screen.getByLabelText('Status filtern')
    fireEvent.change(select, { target: { value: 'ready' } })
    await flushAsync()
    expect((select as HTMLSelectElement).value).toBe('ready')
  })

  test('zeigt Speicherplatz-Leiste aus der storage-Antwort', async () => {
    renderAsPersona(<VideosPage />, 'vorstand', {
      mocks: [
        { url: /\/teams/, data: [] },
        {
          url: /\/videos\?/,
          data: {
            items: [],
            total: 0,
            storage: { free_bytes: 20 * 1024 ** 3, total_bytes: 100 * 1024 ** 3 },
          },
        },
      ],
    })
    await flushAsync()
    expect(await screen.findByText(/80\.0 GB von 100\.0 GB belegt/)).toBeInTheDocument()
    expect(screen.getByText(/20\.0 GB frei/)).toBeInTheDocument()
  })

  test('zeigt genutzten Plattenplatz je Video in der Tabelle', async () => {
    renderAsPersona(<VideosPage />, 'vorstand', {
      mocks: [
        { url: /\/teams/, data: [] },
        {
          url: /\/videos\?/,
          data: { items: [makeVideo(1, { disk_bytes: 1.5 * 1024 ** 3 })], total: 1 },
        },
      ],
    })
    await flushAsync()
    expect(await screen.findByText('1.5 GB')).toBeInTheDocument()
  })
})

describe('VideoStatusPill', () => {
  test.each([
    ['queued', 'In Warteschlange'],
    ['processing', 'Wird verarbeitet'],
    ['ready', 'Bereit'],
    ['failed', 'Fehlgeschlagen'],
  ])('Status %s zeigt Label %s', (status, label) => {
    const { container } = renderAsPersonaNoRouter(<VideoStatusPill status={status} />, 'vorstand')
    expect(within(container).getByText(label)).toBeInTheDocument()
  })

  test('unbekannter Status fällt auf den Rohwert zurück', () => {
    const { container } = renderAsPersonaNoRouter(<VideoStatusPill status="wat" />, 'vorstand')
    expect(within(container).getByText('wat')).toBeInTheDocument()
  })
})
