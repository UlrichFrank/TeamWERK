import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import AuthImage from '../AuthImage'
import { api } from '../../lib/api'

// Wir mocken den axios-Client: gibt bei jedem GET einen fake Blob zurück.
vi.mock('../../lib/api', () => ({
  api: {
    get: vi.fn(() =>
      Promise.resolve({ data: new Blob(['fake-bytes'], { type: 'image/png' }) }),
    ),
  },
}))

// URL.createObjectURL/revokeObjectURL sind in jsdom nicht implementiert.
beforeEach(() => {
  ;(URL as unknown as { createObjectURL: () => string }).createObjectURL = () =>
    'blob:mock'
  ;(URL as unknown as { revokeObjectURL: () => void }).revokeObjectURL = () => {}
})

describe('AuthImage — Aspect-Ratio-Strategie', () => {
  test('mit naturalWidth/Height rendert aspect-ratio ab dem ersten Frame und überspringt den Image()-Probe', async () => {
    // Wenn wir aus Server-Dims rendern, darf KEIN Image()-Objekt erzeugt
    // werden (das war der Client-seitige Preload-Weg, der nur als Fallback
    // gebraucht wird). Wir überwachen den globalen Image-Constructor.
    const originalImage = globalThis.Image
    const imageSpy = vi.fn(function (this: unknown) {})
    imageSpy.prototype = originalImage.prototype
    globalThis.Image = imageSpy as unknown as typeof Image

    const { container } = render(
      <AuthImage
        url="/media/42"
        alt="test"
        className="rounded"
        naturalWidth={1200}
        naturalHeight={800}
      />,
    )

    // Warten bis das Blob als <img> gerendert ist.
    await waitFor(() => {
      const img = container.querySelector('img')
      expect(img).not.toBeNull()
      expect(img?.getAttribute('style')).toMatch(/aspect-ratio:\s*1200\s*\/\s*800/)
    })

    expect(imageSpy).not.toHaveBeenCalled()
    globalThis.Image = originalImage
  })

  test('ohne Server-Dims fällt auf Image()-Probe zurück und setzt aspect-ratio nachträglich', async () => {
    // Wir stellen sicher, dass ein Image() erzeugt und sein onload synchron
    // getriggert wird — dann muss AuthImage genau diese Dims verwenden.
    const originalImage = globalThis.Image
    class FakeImage {
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      naturalWidth = 640
      naturalHeight = 480
      set src(_v: string) {
        setTimeout(() => this.onload?.(), 0)
      }
    }
    globalThis.Image = FakeImage as unknown as typeof Image

    const { container } = render(
      <AuthImage url="/media/99" alt="test" className="rounded" />,
    )

    await waitFor(() => {
      const img = container.querySelector('img')
      expect(img).not.toBeNull()
      expect(img?.getAttribute('style')).toMatch(/aspect-ratio:\s*640\s*\/\s*480/)
    })

    globalThis.Image = originalImage
  })

  // Ein leerer Platzhalter-div mit aspect-ratio trägt in der shrink-to-fit-
  // Sprechblase 0 px zur max-content-Breite bei (gemessen 24×16 px statt
  // 344×262 px in Chromium und WebKit) — deshalb braucht der Platzhalter bei
  // bekannten Server-Dims eine explizite width.
  test('mit Server-Dims trägt der Platzhalter eine explizite width (sonst kollabiert er in der shrink-to-fit-Sprechblase)', async () => {
    vi.mocked(api.get).mockReturnValueOnce(new Promise(() => {}))

    const { container } = render(
      <AuthImage
        url="/media/7"
        alt="test"
        className="max-w-full"
        naturalWidth={1200}
        naturalHeight={800}
      />,
    )

    await waitFor(() => {
      const placeholder = container.querySelector('[aria-busy="true"]')
      expect(placeholder).not.toBeNull()
      expect(placeholder?.getAttribute('style')).toMatch(/aspect-ratio:\s*1200\s*\/\s*800/)
      expect(placeholder?.getAttribute('style')).toMatch(/width:\s*1200px/)
    })

    vi.mocked(api.get).mockReturnValueOnce(new Promise(() => {}))

    const { container: containerOhneDims } = render(
      <AuthImage url="/media/8" alt="test" className="max-w-full" />,
    )

    await waitFor(() => {
      const placeholder = containerOhneDims.querySelector('[aria-busy="true"]')
      expect(placeholder).not.toBeNull()
      expect(placeholder?.getAttribute('style')).toMatch(/min-height:\s*6rem/)
      expect(placeholder?.getAttribute('style')).not.toMatch(/width:/)
    })
  })
})
