import { describe, test, expect, vi, beforeEach } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import ChatImageGrid from '../ChatImageGrid'
import AlbumLightbox from '../AlbumLightbox'
import type { ChatMediaItem } from '../../lib/chatMedia'

// Der Blob-Abruf hängt bewusst für immer: die Tests prüfen den Zustand OHNE
// geladene Bilddaten — genau den Moment, in dem die Sprechblase ihre Größe
// schon haben muss (Chat-Bild-Gotcha).
vi.mock('../../lib/api', () => ({
  api: { get: vi.fn(() => new Promise(() => {})) },
}))

beforeEach(() => {
  ;(URL as unknown as { createObjectURL: () => string }).createObjectURL = () => 'blob:mock'
  ;(URL as unknown as { revokeObjectURL: () => void }).revokeObjectURL = () => {}
})

function album(n: number): ChatMediaItem[] {
  return Array.from({ length: n }, (_, i) => ({ id: i + 1, url: `/media/${i + 1}`, width: 800, height: 600 }))
}

describe('ChatImageGrid', () => {
  test('ein Bild erscheint ohne Raster, mit den Server-Dimensionen', () => {
    const { container } = render(<ChatImageGrid media={album(1)} onOpen={() => {}} />)
    expect(screen.queryByTestId('chat-image-grid')).toBeNull()
    const placeholder = container.querySelector('[aria-busy="true"]') as HTMLElement
    expect(placeholder.style.width).toBe('800px')
    expect(placeholder.style.aspectRatio).toBe('800 / 600')
  })

  test.each([
    [2, 2],
    [3, 3],
    [4, 4],
    [5, 4],
    [10, 4],
  ])('%i Bilder → %i Kacheln', (n, tiles) => {
    render(<ChatImageGrid media={album(n)} onOpen={() => {}} />)
    expect(screen.getAllByTestId('chat-image-tile')).toHaveLength(tiles)
  })

  test('ab dem fünften Bild trägt die vierte Kachel „+N" für die übrigen', () => {
    render(<ChatImageGrid media={album(7)} onOpen={() => {}} />)
    const tiles = screen.getAllByTestId('chat-image-tile')
    expect(tiles[3]).toHaveTextContent('+3')
    expect(screen.getAllByText(/^\+\d+$/)).toHaveLength(1)
  })

  test('bei genau vier Bildern gibt es keinen „+N"-Hinweis', () => {
    render(<ChatImageGrid media={album(4)} onOpen={() => {}} />)
    expect(screen.queryByText(/^\+\d+$/)).toBeNull()
  })

  test('Klick auf eine Kachel meldet ihren Index', () => {
    const onOpen = vi.fn()
    render(<ChatImageGrid media={album(5)} onOpen={onOpen} />)
    fireEvent.click(screen.getAllByTestId('chat-image-tile')[2])
    expect(onOpen).toHaveBeenCalledWith(2)
  })

  test('Kacheln tragen ihre Größe ohne geladene Bilddaten (feste Geometrie)', () => {
    render(<ChatImageGrid media={album(3)} onOpen={() => {}} />)
    const grid = screen.getByTestId('chat-image-grid')
    expect(grid.className).toContain('w-64')
    const tiles = screen.getAllByTestId('chat-image-tile')
    expect(tiles[0].className).toContain('col-span-2')
    expect(tiles[0].className).toContain('aspect-[2/1]')
    expect(tiles[1].className).toContain('aspect-square')
    expect(tiles[2].className).toContain('aspect-square')
    // Die Bild-Box im Inneren füllt die Kachel und setzt keine eigene Größe —
    // die Höhe hängt allein an der Kachel, nicht an den (noch fehlenden) Daten.
    for (const tile of tiles) {
      const inner = tile.querySelector('[aria-busy="true"]') as HTMLElement
      expect(inner).not.toBeNull()
      expect(inner.getAttribute('style') ?? '').toBe('')
      expect(inner.className).toContain('h-full')
    }
  })

  test('leeres Album rendert nichts', () => {
    const { container } = render(<ChatImageGrid media={[]} onOpen={() => {}} />)
    expect(container).toBeEmptyDOMElement()
  })
})

function Harness({ media }: { media: ChatMediaItem[] }) {
  const [box, setBox] = useState<{ urls: string[]; index: number } | null>(null)
  return (
    <>
      <ChatImageGrid media={media} onOpen={(index) => setBox({ urls: media.map((m) => m.url), index })} />
      {box && (
        <AlbumLightbox
          urls={box.urls}
          index={box.index}
          onIndexChange={(index) => setBox({ ...box, index })}
          onClose={() => setBox(null)}
        />
      )}
    </>
  )
}

describe('AlbumLightbox', () => {
  test('Klick auf drittes Bild öffnet die Lightbox bei 3/5, Pfeil rechts → 4/5', () => {
    render(<Harness media={album(5)} />)
    fireEvent.click(screen.getAllByTestId('chat-image-tile')[2])
    expect(screen.getByText('3 / 5')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Nächstes Bild' }))
    expect(screen.getByText('4 / 5')).toBeInTheDocument()
  })

  test('Pfeiltasten blättern, an den Rändern ist Schluss', () => {
    render(<Harness media={album(5)} />)
    fireEvent.click(screen.getAllByTestId('chat-image-tile')[0])
    expect(screen.getByText('1 / 5')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Vorheriges Bild' })).toBeDisabled()

    fireEvent.keyDown(window, { key: 'ArrowLeft' })
    expect(screen.getByText('1 / 5')).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'ArrowRight' })
    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(screen.getByText('3 / 5')).toBeInTheDocument()
  })

  test('die „+N"-Kachel öffnet beim vierten Bild, danach ist das fünfte erreichbar', () => {
    render(<Harness media={album(5)} />)
    fireEvent.click(screen.getAllByTestId('chat-image-tile')[3])
    expect(screen.getByText('4 / 5')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Nächstes Bild' }))
    expect(screen.getByText('5 / 5')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Nächstes Bild' })).toBeDisabled()
  })

  test('ESC schließt', () => {
    render(<Harness media={album(2)} />)
    fireEvent.click(screen.getAllByTestId('chat-image-tile')[1])
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  test('bei einem Einzelbild gibt es weder Pfeile noch Zähler', () => {
    render(<AlbumLightbox urls={['/media/1']} index={0} onIndexChange={() => {}} onClose={() => {}} />)
    expect(screen.queryByRole('button', { name: 'Nächstes Bild' })).toBeNull()
    expect(screen.queryByText('1 / 1')).toBeNull()
  })
})
