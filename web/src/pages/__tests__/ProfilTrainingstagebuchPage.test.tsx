import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'

const get = vi.fn()
vi.mock('../../lib/api', () => ({
  api: { get: (...args: unknown[]) => get(...args), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

// AuthImage würde einen echten Bildabruf auslösen — hier als Spion, um genau
// das nachzuweisen bzw. auszuschließen.
const authImageRendered = vi.fn()
vi.mock('../../components/AuthImage', () => ({
  default: (props: { url: string }) => {
    authImageRendered(props.url)
    return <img alt="Trainingsnachweis" src="" />
  },
}))

const { ProfilTrainingstagebuchContent } = await import('../ProfilTrainingstagebuchPage')

function entry(overrides: Record<string, unknown>) {
  return {
    id: 1,
    member_id: 5,
    season_id: 3,
    trained_on: '2026-05-01T00:00:00Z',
    kind: 'kraft',
    kind_custom: null,
    duration_min: 45,
    rpe: 7,
    note: null,
    proof_status: 'none',
    proof_mime: null,
    proof_purged_at: null,
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

describe('ProfilTrainingstagebuchPage', () => {
  beforeEach(() => {
    get.mockReset()
    authImageRendered.mockReset()
  })

  test('zeigt bei gelöschtem Nachweis einen Hinweis und ruft kein Bild ab', async () => {
    get.mockResolvedValue({
      data: {
        items: [entry({ proof_status: 'purged', proof_purged_at: '2026-09-28T03:00:00Z' })],
      },
    })

    render(<ProfilTrainingstagebuchContent />)

    expect(await screen.findByText(/Nachweis gelöscht/)).toBeInTheDocument()
    // Kern der Regression: kein Bildabruf, der zwangsläufig 410 liefern würde.
    expect(authImageRendered).not.toHaveBeenCalled()
  })

  test('rendert ein vorhandenes Bild über AuthImage', async () => {
    get.mockResolvedValue({
      data: { items: [entry({ proof_status: 'present', proof_mime: 'image/webp' })] },
    })

    render(<ProfilTrainingstagebuchContent />)

    await screen.findByText(/01\.05\.2026/)
    expect(authImageRendered).toHaveBeenCalledWith('/training-diary/1/proof')
  })

  test('bietet bei fehlendem Nachweis das Nachreichen an', async () => {
    get.mockResolvedValue({ data: { items: [entry({ proof_status: 'none' })] } })

    render(<ProfilTrainingstagebuchContent />)

    expect(await screen.findByLabelText('Nachweis nachreichen')).toBeInTheDocument()
    expect(authImageRendered).not.toHaveBeenCalled()
  })

  test('zeigt Art, Dauer und RPE des Eintrags', async () => {
    get.mockResolvedValue({ data: { items: [entry({})] } })

    render(<ProfilTrainingstagebuchContent />)

    expect(await screen.findByText(/01\.05\.2026 · Kraft/)).toBeInTheDocument()
    expect(screen.getByText(/45 min · RPE 7/)).toBeInTheDocument()
  })
})
