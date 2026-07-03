import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import GameEditModal from '../GameEditModal'

let mock: MockAdapter

const GAME = {
  id: 9,
  date: '2026-09-10',
  time: '18:00',
  opponent: 'FC Test',
  event_type: 'heim',
  teams: [{ id: 1, name: 'Herren' }],
  rsvp_default_players: 'none' as const,
  rsvp_default_extended: 'none' as const,
  rsvp_require_reason: 0,
}

beforeEach(() => {
  mock = new MockAdapter(api)
  mock.onGet('/teams').reply(200, [])
  mock.onGet('/duty-templates').reply(200, [])
})
afterEach(() => {
  mock.restore()
})

describe('GameEditModal — RSVP-Voreinstellung', () => {
  test('rendert beide Radio-Gruppen', async () => {
    const { container } = render(
      <GameEditModal game={GAME} onClose={() => {}} onSaved={() => {}} />,
    )
    await screen.findByText('RSVP-Voreinstellung')
    expect(screen.getByText('Kader-Spieler')).toBeInTheDocument()
    expect(screen.getByText('Erweiterter Kader')).toBeInTheDocument()
    expect(container.querySelectorAll('input[name="rsvp-players"]')).toHaveLength(3)
    expect(container.querySelectorAll('input[name="rsvp-extended"]')).toHaveLength(3)
  })

  test('Auswahl landet im PUT-Payload', async () => {
    const onSaved = vi.fn()
    mock.onPut('/games/9').reply(200, {})
    const { container } = render(
      <GameEditModal game={GAME} onClose={() => {}} onSaved={onSaved} />,
    )
    await screen.findByText('RSVP-Voreinstellung')

    fireEvent.click(container.querySelector<HTMLInputElement>('input[name="rsvp-players"][value="confirmed"]')!)
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    const body = JSON.parse(mock.history.put[0].data)
    expect(body.rsvp_default_players).toBe('confirmed')
  })

  test('declined und Reason-Pflicht sind frei kombinierbar (keine Sperre)', async () => {
    const { container } = render(
      <GameEditModal game={GAME} onClose={() => {}} onSaved={() => {}} />,
    )
    await screen.findByText('RSVP-Voreinstellung')

    // Reason gesetzt → declined-Radios bleiben aktiv.
    fireEvent.click(screen.getByLabelText('Begründung bei Absage erforderlich'))
    expect(container.querySelector<HTMLInputElement>('input[name="rsvp-players"][value="declined"]')!.disabled).toBe(false)

    // declined wählen → Reason-Checkbox bleibt aktiv und gesetzt.
    fireEvent.click(container.querySelector<HTMLInputElement>('input[name="rsvp-extended"][value="declined"]')!)
    const reason = screen.getByLabelText('Begründung bei Absage erforderlich') as HTMLInputElement
    expect(reason.disabled).toBe(false)
    expect(reason.checked).toBe(true)
  })
})
