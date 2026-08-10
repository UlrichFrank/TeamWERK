import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import TrainingEditModal from '../TrainingEditModal'
import { WithCapabilities } from '../../test/authCtx'

let mock: MockAdapter

const SESSION = {
  id: 7,
  title: 'Training',
  date: '2026-09-10',
  start_time: '18:00',
  end_time: '20:00',
  status: 'active' as const,
  note: '',
  team_id: 1,
  season_id: 1,
  series_id: undefined as number | undefined,
}

const SERIES = {
  id: 3,
  name: 'Dienstagstraining',
  day_of_week: 1,
  start_time: '18:00',
  end_time: '20:00',
  valid_from: '2026-09-01',
  valid_until: '2027-05-31',
  note: '',
  rsvp_default_players: 'none' as const,
  rsvp_default_extended: 'none' as const,
  rsvp_require_reason: 0,
}

function renderModal(caps: string[], session = SESSION) {
  const onSaved = vi.fn()
  render(
    <WithCapabilities caps={caps}>
      <TrainingEditModal session={session} onClose={() => {}} onSaved={onSaved} />
    </WithCapabilities>,
  )
  return onSaved
}

const openConfirm = () => fireEvent.click(screen.getByLabelText('Training löschen'))
const confirm = () => fireEvent.click(screen.getByRole('button', { name: 'Ja, löschen' }))
const lastDelete = () => mock.history.delete[0]

beforeEach(() => {
  mock = new MockAdapter(api)
  mock.onGet(/\/training-series/).reply(200, [SERIES])
  mock.onDelete(/\/training-sessions\/7/).reply(204)
  mock.onDelete(/\/training-series\/3/).reply(204)
})
afterEach(() => {
  mock.restore()
})

describe('TrainingEditModal — Löschgrund und Stummschaltung', () => {
  test('Einzeltermin: Grund und silent landen im DELETE-Body', async () => {
    const onSaved = renderModal(['suppress_event_notification'])
    openConfirm()

    fireEvent.change(screen.getByLabelText(/Grund/), { target: { value: 'Halle gesperrt' } })
    fireEvent.click(screen.getByLabelText('Ohne Benachrichtigung löschen'))
    confirm()

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(lastDelete().url).toBe('/training-sessions/7')
    expect(JSON.parse(lastDelete().data)).toEqual({ reason: 'Halle gesperrt', silent: true })
  })

  test('Serie: Grund landet im DELETE-Body, Query-Parameter bleiben erhalten', async () => {
    const onSaved = renderModal([], { ...SESSION, series_id: 3 })
    // Serie wird nachgeladen — erst danach ist der Scope-Wechsel wirksam.
    await screen.findByText('Alle der Serie')
    await waitFor(() => expect(mock.history.get.length).toBeGreaterThan(0))

    fireEvent.click(screen.getByLabelText('Alle der Serie'))
    openConfirm()

    fireEvent.change(screen.getByLabelText(/Grund/), { target: { value: 'Saison beendet' } })
    confirm()

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(lastDelete().url).toBe('/training-series/3?scope=all')
    expect(JSON.parse(lastDelete().data)).toEqual({ reason: 'Saison beendet', silent: false })
  })

  test('ohne die Capability fehlt das Häkchen, silent bleibt false', () => {
    renderModal(['manage_trainings'])
    openConfirm()
    expect(screen.getByLabelText(/Grund/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Ohne Benachrichtigung löschen')).not.toBeInTheDocument()
  })

  test('leeres Grundfeld sendet den leeren String', async () => {
    const onSaved = renderModal([])
    openConfirm()
    confirm()

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(JSON.parse(lastDelete().data)).toEqual({ reason: '', silent: false })
  })
})
