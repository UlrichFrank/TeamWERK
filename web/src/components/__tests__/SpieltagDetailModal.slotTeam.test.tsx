import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import SpieltagDetailModal from '../SpieltagDetailModal'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@test.local', role: 'admin', clubFunctions: [] },
    hasCapability: () => true,
  }),
}))

// `duty_slots.team_id` bestimmt bei spielgebundenen Slots nichts mehr — der Client
// bestimmt die Team-Zuordnung gar nicht mehr, sie löst der Server ausschließlich über
// `game_id` -> `game_teams` auf. Das Modal schickt `team_id` deshalb überhaupt nicht mehr
// mit (weder gesetzt noch `null`), unabhängig davon, wie viele Teams der Termin trägt.

let mock: MockAdapter

const DUTY_TYPES = [{ id: 7, name: 'Hallendienst', audiences: [] }]

function mockGame(teams: Array<{ id: number; name: string }>) {
  mock.onGet('/games/234').reply(200, {
    game: {
      id: 234,
      date: '2026-09-13',
      time: '08:00',
      opponent: 'Handball-Neckar-Cup A-/B-Jugend',
      event_type: 'generisch',
      team_id: teams[0]?.id ?? 0,
      teams,
      season_id: 2,
      can: { edit: true, delete: true, manage_lineup: false },
    },
    slots: [],
  })
  mock.onGet(/\/duty-board/).reply(200, [])
  mock.onGet('/duty-types').reply(200, DUTY_TYPES)
  mock.onPost('/duty-slots').reply(201)
}

async function addSlot() {
  const user = userEvent.setup()
  render(
    <MemoryRouter>
      <SpieltagDetailModal gameId={234} onClose={() => {}} />
    </MemoryRouter>,
  )
  await screen.findByText('+ Dienst hinzufügen')
  await user.click(screen.getByText('+ Dienst hinzufügen'))
  await user.selectOptions(screen.getByRole('combobox'), '7')
  await user.click(screen.getByRole('button', { name: 'Hinzufügen' }))
  await waitFor(() => expect(mock.history.post.length).toBe(1))
  return JSON.parse(mock.history.post[0].data)
}

beforeEach(() => {
  mock = new MockAdapter(api)
})
afterEach(() => {
  mock.restore()
})

describe('SpieltagDetailModal — team_id eines neuen Dienst-Slots', () => {
  test('Termin mit mehreren Teams: kein team_id im Request-Body', async () => {
    mockGame([
      { id: 13, name: 'A-Jugend männlich' },
      { id: 15, name: 'B-Jugend männlich' },
      { id: 20, name: 'B-Jugend männlich 2' },
    ])
    const body = await addSlot()
    expect(body).not.toHaveProperty('team_id')
    expect(body.game_id).toBe(234)
  })

  test('Termin mit genau einem Team: kein team_id im Request-Body', async () => {
    mockGame([{ id: 13, name: 'A-Jugend männlich' }])
    const body = await addSlot()
    expect(body).not.toHaveProperty('team_id')
    expect(body.game_id).toBe(234)
  })
})
