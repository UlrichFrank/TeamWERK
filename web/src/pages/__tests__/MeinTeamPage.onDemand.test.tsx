import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import MeinTeamPage from '../MeinTeamPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
// PersonChip hängt am PersonContactProvider; für diesen Test reicht der Name.
vi.mock('../../components/PersonChip', () => ({
  default: ({ name }: { name: string }) => <span>{name}</span>,
}))

function makeRoster(id: number, teamName: string) {
  return {
    team: { id, name: teamName, display_long: teamName },
    trainers: [],
    players: [{ userId: 10 + id, name: `Spieler ${id}`, jerseyNumber: 7 }],
    parents: [],
    extended_players: [],
    extended_parents: [],
  }
}

const TEAMS = [
  { id: 1, name: 'Team Eins' },
  { id: 2, name: 'Team Zwei' },
]

function rosterCalls(teamId: number, mock = getApiMock()) {
  return mock.history.get.filter(c => (c.url ?? '') === `/teams/${teamId}/roster`)
}

describe('MeinTeamPage — roster_loads_only_when_expanded', () => {
  test('mehrere Teams: beim Mount wird kein Roster geladen (nur /teams/my)', async () => {
    renderAsPersona(<MeinTeamPage />, 'spieler', {
      mocks: [{ url: '/teams/my', data: TEAMS }],
    })
    const mock = getApiMock()
    await flushAsync()

    // Beide Team-Header sichtbar, aber KEIN Roster-Request.
    expect(screen.getByText('Team Eins')).toBeInTheDocument()
    expect(screen.getByText('Team Zwei')).toBeInTheDocument()
    expect(rosterCalls(1, mock).length).toBe(0)
    expect(rosterCalls(2, mock).length).toBe(0)
  })

  test('Roster wird erst beim Aufklappen geladen und bei erneutem Aufklappen NICHT neu geholt', async () => {
    renderAsPersona(<MeinTeamPage />, 'spieler', {
      mocks: [{ url: '/teams/my', data: TEAMS }],
    })
    const mock = getApiMock()
    mock.reset()
    mock.onGet('/teams/my').reply(200, TEAMS)
    mock.onGet('/teams/1/roster').reply(200, makeRoster(1, 'Team Eins'))
    mock.onGet('/teams/2/roster').reply(200, makeRoster(2, 'Team Zwei'))
    mock.onGet('/profile/me').reply(200, { id: 1, email: 'x', name: 'x', club_functions: [], is_parent: false, children: [] })
    mock.onAny().reply(200, [])

    await flushAsync()
    expect(screen.getByText('Team Eins')).toBeInTheDocument()
    expect(rosterCalls(1, mock).length).toBe(0)

    // Team Eins aufklappen → genau ein Roster-Request; Spieler sichtbar.
    fireEvent.click(screen.getByRole('button', { name: /Team Eins/ }))
    await flushAsync()
    expect(rosterCalls(1, mock).length).toBe(1)
    expect(await screen.findByText('Spieler 1')).toBeInTheDocument()
    // Team Zwei blieb ungeladen.
    expect(rosterCalls(2, mock).length).toBe(0)

    // Einklappen …
    fireEvent.click(screen.getByRole('button', { name: /Team Eins/ }))
    await flushAsync()
    // … und erneut aufklappen: aus Session-Cache, KEIN zweiter Request.
    fireEvent.click(screen.getByRole('button', { name: /Team Eins/ }))
    await flushAsync()
    expect(rosterCalls(1, mock).length).toBe(1)
    expect(screen.getByText('Spieler 1')).toBeInTheDocument()
  })

  test('Einzelteam wird automatisch aufgeklappt und sein Roster geladen', async () => {
    renderAsPersona(<MeinTeamPage />, 'spieler', {
      mocks: [{ url: '/teams/my', data: [{ id: 1, name: 'Team Eins' }] }],
    })
    const mock = getApiMock()
    mock.reset()
    mock.onGet('/teams/my').reply(200, [{ id: 1, name: 'Team Eins' }])
    mock.onGet('/teams/1/roster').reply(200, makeRoster(1, 'Team Eins'))
    mock.onGet('/profile/me').reply(200, { id: 1, email: 'x', name: 'x', club_functions: [], is_parent: false, children: [] })
    mock.onAny().reply(200, [])

    await flushAsync()
    // Einzelteam: automatisch aufgeklappt → Roster geladen.
    expect(rosterCalls(1, mock).length).toBe(1)
    expect(await screen.findByText('Spieler 1')).toBeInTheDocument()
  })
})
