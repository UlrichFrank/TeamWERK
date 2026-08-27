import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import SpieltagDetailModal from '../SpieltagDetailModal'

// Teil von openspec/changes/dienst-dauer: "+ Dienst hinzufügen" belegt Dauer UND
// Uhrzeit aus dem gewählten Diensttyp vor (Uhrzeit = Zeit des Termins +
// default_offset_minutes, Anker "Start"), beide Felder bleiben editierbar.

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@test.local', role: 'admin', clubFunctions: [] },
    hasCapability: () => true,
  }),
}))

let mock: MockAdapter

// Anker "Start", Versatz -30min, Dauer 1,5h — deckt exakt das Szenario aus
// specs/duties/spec.md ("Dienst hinzufügen belegt Dauer und Uhrzeit vor").
const DUTY_TYPES = [
  { id: 9, name: 'Kasse', hours_value: 1.5, default_anchor: 'start', default_offset_minutes: -30, audiences: [] },
]

function mockGame() {
  mock.onGet('/games/50').reply(200, {
    game: {
      id: 50,
      date: '2026-09-13',
      time: '10:00',
      opponent: 'Testgegner',
      event_type: 'heim',
      team_id: 1,
      teams: [{ id: 1, name: 'A-Jugend' }],
      season_id: 2,
      can: { edit: true, delete: true, manage_lineup: false },
    },
    slots: [],
  })
  mock.onGet(/\/duty-board/).reply(200, [])
  mock.onGet('/duty-types').reply(200, DUTY_TYPES)
  mock.onPost('/duty-slots').reply(201)
}

async function openAddModal() {
  render(
    <MemoryRouter>
      <SpieltagDetailModal gameId={50} onClose={() => {}} />
    </MemoryRouter>,
  )
  const user = userEvent.setup()
  await screen.findByText('+ Dienst hinzufügen')
  await user.click(screen.getByText('+ Dienst hinzufügen'))
  return user
}

beforeEach(() => {
  mock = new MockAdapter(api)
  mockGame()
})
afterEach(() => {
  mock.restore()
})

describe('SpieltagDetailModal — Dauer und Uhrzeit aus dem Diensttyp vorbelegen', () => {
  test('Diensttyp-Auswahl füllt Uhrzeit (Anker + Versatz) und Dauer', async () => {
    const user = await openAddModal()
    await user.selectOptions(screen.getAllByRole('combobox')[0], '9')

    const timeInput = document.querySelector('input[type="time"]') as HTMLInputElement
    expect(timeInput.value).toBe('09:30')

    const hoursInput = screen.getByPlaceholderText('z.B. 1h 30min') as HTMLInputElement
    expect(hoursInput.value).toBe('1h 30min')
  })

  test('eine manuelle Dauer-Änderung überschreibt die Vorbelegung und überlebt einen erneuten Render', async () => {
    const user = await openAddModal()
    await user.selectOptions(screen.getAllByRole('combobox')[0], '9')

    const hoursInput = screen.getByPlaceholderText('z.B. 1h 30min') as HTMLInputElement
    await user.clear(hoursInput)
    await user.type(hoursInput, '2h')
    await user.tab() // blur -> committed
    expect(hoursInput.value).toBe('2h')

    // Ein anderes Feld ändern löst einen erneuten Render des Modals aus, ohne
    // dass die Diensttyp-Auswahl erneut greift — die manuelle Dauer bleibt.
    const personenInput = screen.getByDisplayValue('1') as HTMLInputElement
    await user.clear(personenInput)
    await user.type(personenInput, '3')

    expect(hoursInput.value).toBe('2h')

    await user.click(screen.getByRole('button', { name: 'Hinzufügen' }))
    await waitFor(() => expect(mock.history.post.length).toBe(1))
    const body = JSON.parse(mock.history.post[0].data)
    expect(body.hours_value).toBe(2)
  })
})
