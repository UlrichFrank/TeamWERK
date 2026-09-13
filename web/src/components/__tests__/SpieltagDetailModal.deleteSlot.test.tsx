import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import SpieltagDetailModal from '../SpieltagDetailModal'

// Dienst-Slots werden nur noch über den Bearbeiten-Dialog im Kalender-Modal
// gelöscht (Trash2-Icon im "Dienst bearbeiten"-Dialog), nicht mehr direkt aus
// der Slot-Liste (/dienste) heraus — DutySlotList selbst trägt seit diesem
// Change keinen Löschpfad mehr.

let mock: MockAdapter
let hasCapability: (cap: string) => boolean

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@test.local', role: 'admin', clubFunctions: [] },
    hasCapability: (cap: string) => hasCapability(cap),
  }),
}))

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
    slots: [{
      id: 77, duty_type_name: 'Kasse', event_time: '09:30', hours_value: 1.5,
      role_description: '', slots_total: 1, slots_filled: 0, audiences: [],
    }],
  })
  mock.onGet(/\/duty-board/).reply(200, [{
    slots: [{
      id: 77, duty_type: 'Kasse', duty_type_id: 9, has_instruction: false,
      event_time: '09:30', hours_value: 1.5, slots_total: 1, vacancies: 1,
      claimed_by_me: false, audiences: [], assignees: [],
    }],
  }])
  mock.onGet('/duty-types').reply(200, [])
  mock.onDelete(/\/duty-slots\/\d+/).reply(204)
}

async function openEditDialog() {
  render(
    <MemoryRouter>
      <SpieltagDetailModal gameId={50} onClose={() => {}} />
    </MemoryRouter>,
  )
  const user = userEvent.setup()
  // "Bearbeiten" steckt seit dem ⋮-Menü auch auf Desktop (dienst-kommentare)
  // im ActionMenu statt in einem direkt sichtbaren Button — erst öffnen.
  await user.click((await screen.findAllByLabelText('Aktionen'))[0])
  await user.click((await screen.findAllByText('Bearbeiten'))[0])
  await screen.findByText('Dienst bearbeiten')
  return user
}

const deleteBody = () => JSON.parse(mock.history.delete[0].data)

beforeEach(() => {
  mock = new MockAdapter(api)
  hasCapability = () => false
  mockGame()
})
afterEach(() => {
  mock.restore()
})

describe('SpieltagDetailModal — Dienst löschen im Bearbeiten-Dialog', () => {
  test('Trash2-Button im Bearbeiten-Dialog öffnet die Löschbestätigung, Grund landet im DELETE-Body', async () => {
    const user = await openEditDialog()
    await user.click(screen.getByLabelText('Dienst löschen'))

    await screen.findByText('Dienst löschen?')
    fireEvent.change(screen.getByLabelText(/Grund/), {
      target: { value: 'Dienst wird nicht mehr gebraucht' },
    })
    await user.click(screen.getByRole('button', { name: 'Löschen' }))

    await waitFor(() => expect(mock.history.delete).toHaveLength(1))
    expect(mock.history.delete[0].url).toBe('/duty-slots/77')
    expect(deleteBody()).toEqual({ reason: 'Dienst wird nicht mehr gebraucht', silent: false })
  })

  test('Häkchen "Ohne Benachrichtigung löschen" fehlt ohne die Capability', async () => {
    const user = await openEditDialog()
    await user.click(screen.getByLabelText('Dienst löschen'))

    await screen.findByText('Dienst löschen?')
    expect(screen.queryByLabelText('Ohne Benachrichtigung löschen')).not.toBeInTheDocument()
  })

  test('mit Capability kann stumm gelöscht werden', async () => {
    hasCapability = (cap) => cap === 'suppress_event_notification'
    const user = await openEditDialog()
    await user.click(screen.getByLabelText('Dienst löschen'))

    await user.click(await screen.findByLabelText('Ohne Benachrichtigung löschen'))
    await user.click(screen.getByRole('button', { name: 'Löschen' }))

    await waitFor(() => expect(mock.history.delete).toHaveLength(1))
    expect(deleteBody().silent).toBe(true)
  })

  test('Abbrechen setzt Grund und Häkchen zurück', async () => {
    hasCapability = (cap) => cap === 'suppress_event_notification'
    const user = await openEditDialog()
    await user.click(screen.getByLabelText('Dienst löschen'))

    fireEvent.change(await screen.findByLabelText(/Grund/), { target: { value: 'Tippfehler' } })
    await user.click(screen.getByLabelText('Ohne Benachrichtigung löschen'))
    await user.click(screen.getByRole('button', { name: 'Abbrechen' }))

    await user.click((await screen.findAllByLabelText('Aktionen'))[0])
    await user.click((await screen.findAllByText('Bearbeiten'))[0])
    await user.click(screen.getByLabelText('Dienst löschen'))
    expect(await screen.findByLabelText(/Grund/)).toHaveValue('')
    expect(screen.getByLabelText('Ohne Benachrichtigung löschen')).not.toBeChecked()
  })
})
