import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const get = vi.fn()
vi.mock('../../lib/api', () => ({
  api: { get: (...args: unknown[]) => get(...args) },
}))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
// PersonChip hängt am PersonContactProvider (und damit am AuthContext) — für
// diesen Test reicht der Name, siehe MeinTeamPage.onDemand.test.tsx.
vi.mock('../PersonChip', () => ({
  default: ({ name }: { name: string }) => <span>{name}</span>,
}))

const TrainingDiaryStatsView = (await import('../TrainingDiaryStatsView')).default

const STATS = {
  season_id: 3,
  start_date: '2025-09-01',
  end_date: '2026-06-30',
  items: [
    { member_id: 5, member_name: 'Anna Muster', entries: 12, minutes: 540, avg_rpe: 6.2 },
    { member_id: 6, member_name: 'Ben Beispiel', entries: 0, minutes: 0, avg_rpe: 0 },
  ],
}

const DETAIL = {
  items: [
    {
      id: 91,
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
    },
  ],
}

describe('TrainingDiaryStatsView', () => {
  beforeEach(() => {
    get.mockReset()
    get.mockImplementation((url: string) => {
      if (url.includes('training-diary-stats')) return Promise.resolve({ data: STATS })
      return Promise.resolve({ data: DETAIL })
    })
  })

  test('rendert Kennzahlen je Mitglied', async () => {
    render(<TrainingDiaryStatsView teamId={13} />)

    expect(await screen.findByText('Anna Muster')).toBeInTheDocument()
    expect(screen.getByText(/12 Einh\..*540 min.*RPE-Schnitt 6\.2/)).toBeInTheDocument()
    // Mitglied ohne Einträge erscheint mit Nullwerten, ohne RPE-Angabe.
    expect(screen.getByText('Ben Beispiel')).toBeInTheDocument()
    expect(screen.getByText(/0 Einh\..*0 min/)).toBeInTheDocument()
  })

  test('weist auf die Aussagekraft der Zahlen hin', async () => {
    render(<TrainingDiaryStatsView teamId={13} />)
    expect(await screen.findByText(/Selbstauskunft/)).toBeInTheDocument()
  })

  test('lädt die Einzeleinträge erst beim Aufklappen nach', async () => {
    render(<TrainingDiaryStatsView teamId={13} />)
    await screen.findByText('Anna Muster')

    // Vor dem Klick wurde nur die Team-Übersicht geholt.
    expect(get.mock.calls.filter(c => String(c[0]).includes('/members/'))).toHaveLength(0)

    // Name ist jetzt ein PersonChip (eigener <button>) — Klick auf die Kennzahlen
    // im selben Toggle-Container, statt per Name-Rolle (der Name matcht sonst
    // zwei Buttons: den Zeilen-Toggle und den PersonChip).
    fireEvent.click(screen.getByText(/12 Einh\..*540 min.*RPE-Schnitt 6\.2/))

    await waitFor(() => {
      expect(get).toHaveBeenCalledWith('/members/5/training-diary?season=3')
    })
    expect(await screen.findByText('45 min')).toBeInTheDocument()
  })
})
