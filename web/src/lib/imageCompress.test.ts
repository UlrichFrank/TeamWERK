import { beforeEach, describe, expect, it, vi } from 'vitest'
import { compressImage } from './imageCompress'

// jsdom liefert kein funktionierendes createImageBitmap oder canvas.toBlob.
// Wir stubben beides so, dass der Compress-Loop deterministisch die MIME-Wahl
// des Aufrufers durchläuft und Blobs der gewünschten Größe zurückliefert.

const bigFile = (name: string, size = 4 * 1024 * 1024): File => {
  const buf = new Uint8Array(size)
  return new File([buf], name, { type: 'image/jpeg' })
}

const smallFile = (name: string, size = 100 * 1024): File => {
  const buf = new Uint8Array(size)
  return new File([buf], name, { type: 'image/png' })
}

beforeEach(() => {
  vi.stubGlobal('createImageBitmap', vi.fn(async () => ({
    width: 3000,
    height: 2000,
    close: () => {},
  }) as unknown as ImageBitmap))

  const proto = HTMLCanvasElement.prototype as unknown as {
    getContext: (ctx: string) => unknown
    toBlob: (cb: (b: Blob | null) => void, mime: string, q: number) => void
  }
  proto.getContext = () =>
    ({ drawImage: () => {} }) as unknown
  proto.toBlob = function (cb, mime, _q) {
    // ~500 KB pro Aufruf, so dass die erste Qualitätsstufe schon unter 1 MB ist
    const blob = new Blob([new Uint8Array(500 * 1024)], { type: mime })
    cb(blob)
  }
})

describe('compressImage', () => {
  it('returns file unchanged when already below target', async () => {
    const f = smallFile('logo.png')
    const res = await compressImage(f)
    expect(res.blob).toBe(f)
    expect(res.fileName).toBe('logo.png')
  })

  it('defaults produce a webp or jpg output for large files', async () => {
    const res = await compressImage(bigFile('camera.jpg'))
    expect(res.fileName).toMatch(/\.(webp|jpg)$/)
    expect(res.blob.size).toBeLessThanOrEqual(1 << 20)
  })

  it('JPEG-only opts produce .jpg output (no WebP)', async () => {
    const res = await compressImage(bigFile('camera.jpg'), {
      formats: [{ mime: 'image/jpeg', ext: '.jpg' }],
    })
    expect(res.fileName).toBe('camera.jpg')
    expect(res.blob.type).toBe('image/jpeg')
  })

  it('respects custom targetBytes', async () => {
    // 500 KB Ziel — der Mock liefert genau 500 KB, sollte also passen
    const res = await compressImage(bigFile('camera.jpg', 2 * 1024 * 1024), {
      targetBytes: 512 * 1024,
      formats: [{ mime: 'image/jpeg', ext: '.jpg' }],
    })
    expect(res.fileName).toBe('camera.jpg')
    expect(res.blob.size).toBeLessThanOrEqual(512 * 1024)
  })

  it('returns smallest attempt when no quality step reaches target', async () => {
    // toBlob liefert immer 2 MB → Ziel 1 MB wird nie erreicht, aber die
    // kleinste Variante wird zurückgegeben.
    const proto = HTMLCanvasElement.prototype as unknown as {
      toBlob: (cb: (b: Blob | null) => void, mime: string, q: number) => void
    }
    proto.toBlob = function (cb, mime, _q) {
      cb(new Blob([new Uint8Array(2 * 1024 * 1024)], { type: mime }))
    }
    const res = await compressImage(bigFile('huge.jpg'), {
      formats: [{ mime: 'image/jpeg', ext: '.jpg' }],
    })
    expect(res.fileName).toBe('huge.jpg')
    expect(res.blob.type).toBe('image/jpeg')
  })
})
