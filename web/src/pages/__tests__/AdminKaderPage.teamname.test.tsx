import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent, within } from '@testing-library/react'
import AdminKaderPage from '../AdminKaderPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function kader(id: number, ageClass: string, gender: string, teamNumber: number) {
  return {
    id, season_id: 1, age_class: ageClass, gender, team_number: teamNumber, team_id: 100 + id,
    dedicated_birth_year: null, games_per_season: 0, staffel: '',
    birth_years: [], bracket_years: [], members: [], member_count: 0, trainers: [], extended_members: [],
  }
}

const KADER = [
  kader(1, 'C-Jugend', 'm', 1),
  kader(2, 'C-Jugend', 'm', 2),
]

function render() {
  renderAsPersona(<AdminKaderPage />, 'vorstand', {
    mocks: [
      { url: '/seasons', data: [{ id: 1, name: '2026/27', start_date: '2026-07-01', is_active: true }] },
      { url: /^\/kader\?season_id=1/, data: { items: KADER, total: KADER.length } },
      { url: '/age-class-rules', data: [] },
      { url: '/training-group-categories', data: [] },
    ],
  })
}

describe('AdminKaderPage — Teamname mit Nummer am Ende', () => {
  test('zwei Mannschaften derselben Kombination heißen "… männlich 1" und "… männlich 2"', async () => {
    render()
    await flushAsync()
    await flushAsync()
    expect(screen.getByRole('heading', { name: 'C-Jugend männlich 1' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'C-Jugend männlich 2' })).toBeInTheDocument()
  })

  test('Lösch-Bestätigung nennt auch Mannschaft 1 mit Nummer', async () => {
    render()
    await flushAsync()
    await flushAsync()
    const card = screen.getByRole('heading', { name: 'C-Jugend männlich 1' }).closest('div.eckfahne') as HTMLElement
    fireEvent.click(within(card).getByRole('button', { name: 'Aktionen' }))
    fireEvent.click(screen.getByText('Löschen'))
    expect(screen.getByText('C-Jugend männlich 1 wird unwiderruflich gelöscht.')).toBeInTheDocument()
  })
})
