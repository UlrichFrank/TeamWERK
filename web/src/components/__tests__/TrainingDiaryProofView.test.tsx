import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { DiaryEntry } from '../../lib/trainingDiary'

const get = vi.fn()
vi.mock('../../lib/api', () => ({
  api: { get: (...args: unknown[]) => get(...args), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

// AuthImage würde einen echten Bildabruf auslösen; hier nur die URL nachweisen.
vi.mock('../AuthImage', () => ({
  default: (props: { url: string; className?: string }) => (
    <img alt="Trainingsnachweis" data-url={props.url} className={props.className} src="" />
  ),
}))

// Der Download läuft absichtlich über den echten downloadProof (Dateiname,
// responseType) — nur die Geräte-Übergabe wird abgefangen.
const openBlobNatively = vi.fn()
vi.mock('../../lib/openFileNatively', () => ({
  openBlobNatively: (...args: unknown[]) => openBlobNatively(...args),
}))

const { default: TrainingDiaryProofView } = await import('../TrainingDiaryProofView')

function entry(overrides: Partial<DiaryEntry> = {}): DiaryEntry {
  return {
    id: 7,
    member_id: 4,
    season_id: 3,
    trained_on: '2026-05-01T00:00:00Z',
    kind: 'kraft',
    kind_custom: null,
    duration_min: 45,
    rpe: 7,
    note: null,
    proof_status: 'present',
    proof_mime: 'image/webp',
    proof_purged_at: null,
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

describe('TrainingDiaryProofView', () => {
  beforeEach(() => {
    get.mockReset()
    openBlobNatively.mockReset()
  })

  test('Bild öffnet im Vollbild und schließt per ESC', async () => {
    const user = userEvent.setup()
    render(<TrainingDiaryProofView entry={entry()} thumbClassName="max-h-48" />)

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Nachweis vergrößern' }))
    const dialog = screen.getByRole('dialog')
    expect(dialog).toBeInTheDocument()
    // Das Vollbild lädt dieselbe geschützte URL wie die Vorschau.
    expect(dialog.querySelector('img')?.getAttribute('data-url')).toBe('/training-diary/7/proof')

    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  test('Vollbild schließt per Klick auf den Hintergrund', async () => {
    const user = userEvent.setup()
    render(<TrainingDiaryProofView entry={entry()} thumbClassName="max-h-48" />)

    await user.click(screen.getByRole('button', { name: 'Nachweis vergrößern' }))
    await user.click(screen.getByRole('dialog'))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  test('PDF wird heruntergeladen statt inline gezeigt', async () => {
    const user = userEvent.setup()
    const blob = new Blob(['%PDF-1.4'], { type: 'application/pdf' })
    get.mockResolvedValue({ data: blob })

    render(
      <TrainingDiaryProofView
        entry={entry({ proof_mime: 'application/pdf' })}
        thumbClassName="max-h-48"
      />,
    )

    // Kein Bildabruf, kein Vergrößern-Button für Nicht-Bilder.
    expect(screen.queryByRole('button', { name: 'Nachweis vergrößern' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Nachweis herunterladen/ }))

    await waitFor(() => expect(openBlobNatively).toHaveBeenCalled())
    expect(get).toHaveBeenCalledWith('/training-diary/7/proof', { responseType: 'blob' })
    // Dateiname trägt das Trainingsdatum und die Endung des MIME-Typs.
    expect(openBlobNatively).toHaveBeenCalledWith(blob, 'trainingsnachweis-01-05-2026.pdf')
  })

  test('meldet einen fehlgeschlagenen Download sichtbar', async () => {
    const user = userEvent.setup()
    get.mockRejectedValue(new Error('403'))

    render(
      <TrainingDiaryProofView
        entry={entry({ proof_mime: 'application/pdf' })}
        thumbClassName="max-h-48"
      />,
    )

    await user.click(screen.getByRole('button', { name: /Nachweis herunterladen/ }))

    expect(await screen.findByText(/konnte nicht geladen werden/)).toBeInTheDocument()
    expect(openBlobNatively).not.toHaveBeenCalled()
  })

  test('Label der Mannschaftsübersicht ist überschreibbar', () => {
    render(
      <TrainingDiaryProofView
        entry={entry({ proof_mime: 'application/pdf' })}
        thumbClassName="max-h-24"
        fileLabel="Datei"
      />,
    )
    expect(screen.getByRole('button', { name: /Datei/ })).toBeInTheDocument()
  })
})
