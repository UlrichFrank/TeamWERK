import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import DutySlotList, { type BoardSlot } from '../DutySlotList'
import { WithCapabilities } from '../../test/authCtx'

// DutySlotList ist der tatsächliche Löschpfad für Dienst-Slots — nicht
// SpieltagDetailModal, dessen „Dienst löschen?"-Block über die UI nicht
// erreichbar ist. Dieser Test hält fest, dass der Grund hier ankommt, damit
// die Backend-Arbeit an DELETE /api/duty-slots/{id} eine Oberfläche behält.

let mock: MockAdapter

const BESETZT: BoardSlot = {
  id: 7,
  duty_type: 'Kasse',
  duty_type_id: 1,
  has_instruction: false,
  event_time: '18:00',
  hours_value: 1,
  slots_total: 2,
  vacancies: 0, // slots_filled = 2 → Bestätigungsdialog
  claimed_by_me: false,
}

const UNBESETZT: BoardSlot = { ...BESETZT, id: 8, vacancies: 2 }

function renderList(slots: BoardSlot[], caps: string[]) {
  const onReload = vi.fn()
  render(
    <MemoryRouter>
      <WithCapabilities caps={caps}>
        <DutySlotList slots={slots} isPast={false} canEdit onReload={onReload} />
      </WithCapabilities>
    </MemoryRouter>,
  )
  return onReload
}

const clickDelete = () => fireEvent.click(screen.getAllByLabelText(/löschen/i)[0])
const deleteBody = () => JSON.parse(mock.history.delete[0].data)

beforeEach(() => {
  mock = new MockAdapter(api)
  mock.onDelete(/\/duty-slots\/\d+/).reply(204)
})
afterEach(() => {
  mock.restore()
})

describe('DutySlotList — Löschgrund und Stummschaltung', () => {
  test('besetzter Slot: Grund landet im DELETE-Body', async () => {
    renderList([BESETZT], [])
    clickDelete()

    fireEvent.change(await screen.findByLabelText(/Grund/), {
      target: { value: 'Dienst wird nicht mehr gebraucht' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Löschen' }))

    await waitFor(() => expect(mock.history.delete).toHaveLength(1))
    expect(mock.history.delete[0].url).toBe('/duty-slots/7')
    expect(deleteBody()).toEqual({ reason: 'Dienst wird nicht mehr gebraucht', silent: false })
  })

  test('Häkchen fehlt ohne die Capability', async () => {
    renderList([BESETZT], ['manage_duties'])
    clickDelete()

    await screen.findByLabelText(/Grund/)
    expect(screen.queryByLabelText('Ohne Benachrichtigung löschen')).not.toBeInTheDocument()
  })

  test('Vorstand kann stumm löschen', async () => {
    renderList([BESETZT], ['suppress_event_notification'])
    clickDelete()

    fireEvent.click(await screen.findByLabelText('Ohne Benachrichtigung löschen'))
    fireEvent.click(screen.getByRole('button', { name: 'Löschen' }))

    await waitFor(() => expect(mock.history.delete).toHaveLength(1))
    expect(deleteBody().silent).toBe(true)
  })

  // Unbesetzte Slots werden ohne Rückfrage gelöscht — es gibt niemanden zu
  // benachrichtigen, also ist ein Grund gegenstandslos. Der Request darf dann
  // auch keinen Body tragen (der Server toleriert beides, aber ein leerer
  // Grund im Text wäre irreführend).
  test('unbesetzter Slot löscht ohne Dialog und ohne Body', async () => {
    renderList([UNBESETZT], ['suppress_event_notification'])
    clickDelete()

    await waitFor(() => expect(mock.history.delete).toHaveLength(1))
    expect(mock.history.delete[0].url).toBe('/duty-slots/8')
    expect(mock.history.delete[0].data).toBeUndefined()
    expect(screen.queryByLabelText(/Grund/)).not.toBeInTheDocument()
  })

  test('Abbrechen setzt Grund und Häkchen zurück', async () => {
    renderList([BESETZT], ['suppress_event_notification'])
    clickDelete()

    fireEvent.change(await screen.findByLabelText(/Grund/), { target: { value: 'Tippfehler' } })
    fireEvent.click(screen.getByLabelText('Ohne Benachrichtigung löschen'))
    fireEvent.click(screen.getByRole('button', { name: 'Abbrechen' }))

    clickDelete()
    expect(await screen.findByLabelText(/Grund/)).toHaveValue('')
    expect(screen.getByLabelText('Ohne Benachrichtigung löschen')).not.toBeChecked()
  })
})
