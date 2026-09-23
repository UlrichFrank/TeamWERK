import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import AttendanceStatsView from '../AttendanceStatsView'

// openspec/changes/anwesenheit-quote-aufteilen: drei Anteile statt einer Quote,
// die Entschuldigungen aus dem Nenner nahm.

const mockGet = vi.fn()
vi.mock('../../lib/api', () => ({ api: { get: (...a: unknown[]) => mockGet(...a) } }))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function stats(counts: Record<string, number>) {
  return {
    member_id: 1, season_id: 1, start_date: '2026-08-01', end_date: '2026-09-23',
    counts: {
      member_id: 1, member_name: 'A',
      training_present: 0, training_missed: 0, training_excused: 0,
      game_present: 0, game_missed: 0, game_excused: 0,
      ...counts,
    },
    events: [],
  }
}

beforeEach(() => mockGet.mockReset())

describe('AttendanceStatsView — Anteile', () => {
  test('zeigt anwesend/entschuldigt/fehlt als Anteil am Gesamt, keine Quote', async () => {
    mockGet.mockResolvedValue({ data: stats({ training_present: 5, training_excused: 3, training_missed: 2 }) })
    render(<AttendanceStatsView memberId={1} />)

    const training = (await screen.findByText('Trainings')).closest('div.rounded-xl') as HTMLElement
    expect(within(training).getByText('10 Termine')).toBeTruthy()
    expect(within(training).getByTestId('pillar-present').textContent).toContain('50 %')
    expect(within(training).getByTestId('pillar-present').textContent).toContain('(5)')
    expect(within(training).getByTestId('pillar-excused').textContent).toContain('30 %')
    expect(within(training).getByTestId('pillar-missed').textContent).toContain('20 %')
    expect(screen.queryByText(/Quote/)).toBeNull()
  })

  test('ohne gezählte Termine keine Prozentwerte', async () => {
    mockGet.mockResolvedValue({ data: stats({}) })
    render(<AttendanceStatsView memberId={1} />)
    const games = (await screen.findByText('Spiele')).closest('div.rounded-xl') as HTMLElement
    expect(within(games).queryByText(/%/)).toBeNull()
    expect(within(games).getByText('0 Termine')).toBeTruthy()
  })
})
