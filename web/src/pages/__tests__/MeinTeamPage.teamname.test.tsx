import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import MeinTeamPage from '../MeinTeamPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../components/PersonChip', () => ({
  default: ({ name }: { name: string }) => <span>{name}</span>,
}))

// Gespeicherter teams.name: Mannschaft 1 ohne Nummer (kader.teamLabel).
// Die Langform des Servers trägt die Nummer am Ende — auch für Mannschaft 1.
const TEAMS = [
  { id: 17, name: 'C-Jugend männlich', display_short: 'mC1', display_long: 'C-Jugend männlich 1' },
  { id: 18, name: 'C-Jugend männlich 2', display_short: 'mC2', display_long: 'C-Jugend männlich 2' },
]

function roster(id: number, displayLong: string, name: string) {
  return {
    team: { id, name, display_long: displayLong },
    trainers: [],
    players: [{ userId: 100 + id, name: `Spieler ${id}`, jerseyNumber: 7 }],
    parents: [],
    extended_players: [],
    extended_parents: [],
  }
}

describe('MeinTeamPage — Teamname eingeklappt = aufgeklappt', () => {
  test('eingeklappte Karten zeigen die Langform mit Nummer am Ende', async () => {
    renderAsPersona(<MeinTeamPage />, 'spieler', {
      mocks: [{ url: '/teams/my', data: TEAMS }],
    })
    await flushAsync()

    expect(screen.getByRole('heading', { name: 'C-Jugend männlich 1' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'C-Jugend männlich 2' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'C-Jugend männlich' })).not.toBeInTheDocument()
  })

  test('Aufklappen ändert den Titel nicht', async () => {
    renderAsPersona(<MeinTeamPage />, 'spieler', {
      mocks: [{ url: '/teams/my', data: TEAMS }],
    })
    const mock = getApiMock()
    mock.reset()
    mock.onGet('/teams/my').reply(200, TEAMS)
    mock.onGet('/teams/17/roster').reply(200, roster(17, 'C-Jugend männlich 1', 'C-Jugend männlich'))
    mock.onGet('/profile/me').reply(200, { id: 1, email: 'x', name: 'x', club_functions: [], is_parent: false, children: [] })
    mock.onAny().reply(200, [])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: /C-Jugend männlich 1/ }))
    await flushAsync()
    expect(await screen.findByText('Spieler 17')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'C-Jugend männlich 1' })).toBeInTheDocument()
  })

  test('ohne Langform fällt der Titel auf teams.name zurück', async () => {
    renderAsPersona(<MeinTeamPage />, 'spieler', {
      mocks: [{ url: '/teams/my', data: [{ id: 1, name: 'Herren' }, { id: 2, name: 'Damen' }] }],
    })
    await flushAsync()
    expect(screen.getByRole('heading', { name: 'Herren' })).toBeInTheDocument()
  })
})
