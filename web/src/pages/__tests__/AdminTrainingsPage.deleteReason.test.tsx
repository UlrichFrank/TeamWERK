import { describe, test, expect, vi, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import AdminTrainingsPage from '../AdminTrainingsPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

// useLiveUpdates öffnet eine EventSource (in jsdom nicht vorhanden) → mocken.
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const SERIES = [{
  id: 3, team_id: 1, season_id: 1, name: 'Dienstagstraining', venue: null,
  day_of_week: 1, start_time: '18:00', end_time: '20:00',
  valid_from: '2026-09-01', valid_until: '2027-05-31', note: '',
  team_name: 'Herren 1', session_count: 30,
  rsvp_default_players: 'none' as const, rsvp_default_extended: 'none' as const, rsvp_require_reason: 0,
}]

const SESSIONS = {
  items: [{
    id: 7, team_id: 1, date: '2026-09-10', start_time: '18:00', end_time: '20:00',
    venue: null, note: '', status: 'active', cancel_reason: '',
  }],
  total: 1,
}

async function renderPage(personaId: string) {
  const r = renderAsPersona(<AdminTrainingsPage />, personaId, {
    mocks: [
      { url: '/training-series', data: SERIES },
      { url: /\/training-sessions\?/, data: SESSIONS },
      { url: '/teams', data: [] },
      { url: '/seasons', data: [{ id: 1, name: '2026/27', is_active: true }] },
    ],
  })
  await flushAsync()
  return r
}

const openDelete = () => fireEvent.click(screen.getAllByLabelText('Löschen')[0])

// Die Trash-Buttons der Listen tragen aria-label „Löschen" — der Bestätigungs-Button
// heißt genauso. Deshalb gezielt innerhalb des Dialogs suchen.
const confirmDelete = () => {
  const dialog = screen.getByText(/^(Serie|Einzeltermin) löschen$/).closest('div.bg-white') as HTMLElement
  fireEvent.click(within(dialog).getByRole('button', { name: 'Löschen' }))
}
const lastDelete = () => getApiMock().history.delete[0]

beforeEach(() => {
  vi.clearAllMocks()
})

describe('AdminTrainingsPage — Löschgrund und Stummschaltung', () => {
  test('Serie: Grund und silent landen im DELETE-Body, Scope bleibt in der Query', async () => {
    await renderPage('vorstand')
    await screen.findByText('Dienstagstraining')

    openDelete()
    fireEvent.change(screen.getByLabelText(/Grund/), { target: { value: 'Halle gesperrt' } })
    fireEvent.click(screen.getByLabelText('Ohne Benachrichtigung löschen'))
    confirmDelete()

    await waitFor(() => expect(getApiMock().history.delete.length).toBe(1))
    expect(lastDelete().url).toBe('/training-series/3?scope=future')
    expect(JSON.parse(lastDelete().data)).toEqual({ reason: 'Halle gesperrt', silent: true })
  })

  test('Einzeltermin: Grund landet im DELETE-Body', async () => {
    await renderPage('vorstand')
    fireEvent.click(screen.getByRole('button', { name: 'Einzeltermine' }))
    await screen.findAllByLabelText('Löschen')

    openDelete()
    fireEvent.change(screen.getByLabelText(/Grund/), { target: { value: 'Trainer krank' } })
    confirmDelete()

    await waitFor(() => expect(getApiMock().history.delete.length).toBe(1))
    expect(lastDelete().url).toBe('/training-sessions/7')
    expect(JSON.parse(lastDelete().data)).toEqual({ reason: 'Trainer krank', silent: false })
  })

  test('Trainer sieht kein Stummschalt-Häkchen — silent bleibt false', async () => {
    await renderPage('trainer')
    await screen.findByText('Dienstagstraining')

    openDelete()
    expect(screen.getByLabelText(/Grund/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Ohne Benachrichtigung löschen')).not.toBeInTheDocument()

    confirmDelete()
    await waitFor(() => expect(getApiMock().history.delete.length).toBe(1))
    expect(JSON.parse(lastDelete().data)).toEqual({ reason: '', silent: false })
  })
})
