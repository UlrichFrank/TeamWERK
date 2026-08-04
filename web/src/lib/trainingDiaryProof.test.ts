import { describe, test, expect, vi, beforeEach } from 'vitest'

// Beide Abhängigkeiten werden gemockt: geprüft wird ausschließlich, ob der
// Kompressions-Pfad mit dem richtigen (engeren) Budget aufgerufen wird und ob
// Nicht-Bilder ihn korrekt umgehen.
const compressImage = vi.fn()
const post = vi.fn()

vi.mock('./imageCompress', () => ({
  compressImage: (...args: unknown[]) => compressImage(...args),
}))
vi.mock('./api', () => ({
  api: {
    post: (...args: unknown[]) => post(...args),
    delete: vi.fn(),
  },
}))

const { uploadProof } = await import('./trainingDiaryProof')

describe('uploadProof', () => {
  beforeEach(() => {
    compressImage.mockReset()
    post.mockReset()
    post.mockResolvedValue({ data: { id: 1 } })
  })

  test('komprimiert Bilder auf 150 KB / 1280 px', async () => {
    const blob = new Blob(['x'], { type: 'image/webp' })
    compressImage.mockResolvedValue({ blob, fileName: 'beleg.webp' })
    const file = new File(['x'.repeat(500)], 'foto.jpg', { type: 'image/jpeg' })

    await uploadProof(7, file)

    expect(compressImage).toHaveBeenCalledWith(file, {
      targetBytes: 153600,
      maxEdge: 1280,
    })
    expect(post).toHaveBeenCalledWith('/training-diary/7/proof', expect.any(FormData))
  })

  test('reicht PDFs unverändert durch', async () => {
    const file = new File(['%PDF-'], 'plan.pdf', { type: 'application/pdf' })

    await uploadProof(9, file)

    expect(compressImage).not.toHaveBeenCalled()
    expect(post).toHaveBeenCalledWith('/training-diary/9/proof', expect.any(FormData))
  })

  test('legt die Datei unter dem Feldnamen "proof" ab', async () => {
    const file = new File(['%PDF-'], 'plan.pdf', { type: 'application/pdf' })

    await uploadProof(3, file)

    const form = post.mock.calls[0][1] as FormData
    expect(form.get('proof')).toBeInstanceOf(Blob)
  })
})
