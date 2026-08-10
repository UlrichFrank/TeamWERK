import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import GameEditModal from '../GameEditModal'
import { WithCapabilities } from '../../test/authCtx'

let mock: MockAdapter

const GAME = {
  id: 9,
  date: '2026-09-10',
  time: '18:00',
  opponent: 'HSG Ostfildern',
  event_type: 'heim',
  teams: [{ id: 1, name: 'Herren' }],
}

function renderModal(caps: string[], onDeleted = vi.fn()) {
  render(
    <WithCapabilities caps={caps}>
      <GameEditModal game={GAME} onClose={() => {}} onSaved={() => {}} onDeleted={onDeleted} />
    </WithCapabilities>,
  )
  return onDeleted
}

const openConfirm = () => fireEvent.click(screen.getByLabelText('Event löschen'))
const confirm = () => fireEvent.click(screen.getByRole('button', { name: 'Ja, löschen' }))
const deleteBody = () => JSON.parse(mock.history.delete[0].data)

beforeEach(() => {
  mock = new MockAdapter(api)
  mock.onGet('/teams').reply(200, [])
  mock.onGet('/teams/names').reply(200, [])
  mock.onGet('/duty-templates').reply(200, [])
  mock.onDelete('/games/9').reply(200, {})
})
afterEach(() => {
  mock.restore()
})

describe('GameEditModal — Löschgrund und Stummschaltung', () => {
  test('Grundfeld erscheint erst im Bestätigungsblock', async () => {
    renderModal([])
    await screen.findByLabelText('Event löschen')
    expect(screen.queryByLabelText(/Grund/)).not.toBeInTheDocument()

    openConfirm()
    expect(screen.getByLabelText(/Grund/)).toBeInTheDocument()
  })

  test('Grund landet im DELETE-Body', async () => {
    const onDeleted = renderModal([])
    await screen.findByLabelText('Event löschen')
    openConfirm()

    fireEvent.change(screen.getByLabelText(/Grund/), { target: { value: 'Halle gesperrt' } })
    confirm()

    await waitFor(() => expect(onDeleted).toHaveBeenCalled())
    expect(deleteBody()).toEqual({ reason: 'Halle gesperrt', silent: false })
  })

  test('silent landet im DELETE-Body, wenn der Vorstand das Häkchen setzt', async () => {
    const onDeleted = renderModal(['suppress_event_notification'])
    await screen.findByLabelText('Event löschen')
    openConfirm()

    fireEvent.change(screen.getByLabelText(/Grund/), { target: { value: 'Import-Dublette' } })
    fireEvent.click(screen.getByLabelText('Ohne Benachrichtigung löschen'))
    confirm()

    await waitFor(() => expect(onDeleted).toHaveBeenCalled())
    expect(deleteBody()).toEqual({ reason: 'Import-Dublette', silent: true })
  })

  test('ohne die Capability gibt es kein Häkchen — silent bleibt false', async () => {
    const onDeleted = renderModal(['manage_games'])
    await screen.findByLabelText('Event löschen')
    openConfirm()

    expect(screen.queryByLabelText('Ohne Benachrichtigung löschen')).not.toBeInTheDocument()
    confirm()

    await waitFor(() => expect(onDeleted).toHaveBeenCalled())
    expect(deleteBody().silent).toBe(false)
  })

  test('leeres Grundfeld sendet den leeren String', async () => {
    const onDeleted = renderModal([])
    await screen.findByLabelText('Event löschen')
    openConfirm()
    confirm()

    await waitFor(() => expect(onDeleted).toHaveBeenCalled())
    expect(deleteBody()).toEqual({ reason: '', silent: false })
  })
})
