import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import AuthImage from '../AuthImage'

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
})
