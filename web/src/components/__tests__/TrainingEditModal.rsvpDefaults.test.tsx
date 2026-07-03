import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import TrainingEditModal from '../TrainingEditModal'

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
  rsvp_default_players: 'none' as const,
  rsvp_default_extended: 'none' as const,
  rsvp_require_reason: 0,
}

beforeEach(() => {
  mock = new MockAdapter(api)
})
afterEach(() => {
  mock.restore()
})

describe('TrainingEditModal — RSVP-Voreinstellung', () => {
  test('rendert beide Radio-Gruppen mit je drei Optionen', () => {
    const { container } = render(
      <TrainingEditModal session={SESSION} onClose={() => {}} onSaved={() => {}} />,
    )
    expect(screen.getByText('RSVP-Voreinstellung')).toBeInTheDocument()
    expect(screen.getByText('Kader-Spieler')).toBeInTheDocument()
    expect(screen.getByText('Erweiterter Kader')).toBeInTheDocument()
    expect(container.querySelectorAll('input[name="rsvp-players"]')).toHaveLength(3)
    expect(container.querySelectorAll('input[name="rsvp-extended"]')).toHaveLength(3)
  })

  test('Auswahl „Standardmäßig abgesagt" für Erweiterten Kader landet im PUT-Payload', async () => {
    const onSaved = vi.fn()
    mock.onPut('/training-sessions/7').reply(204)
    const { container } = render(
      <TrainingEditModal session={SESSION} onClose={() => {}} onSaved={onSaved} />,
    )

    const extDeclined = container.querySelector<HTMLInputElement>('input[name="rsvp-extended"][value="declined"]')!
    fireEvent.click(extDeclined)
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    const body = JSON.parse(mock.history.put[0].data)
    expect(body.rsvp_default_extended).toBe('declined')
    expect(body.rsvp_default_players).toBe('none')
  })

  test('gesetzte Reason-Checkbox deaktiviert die „abgesagt"-Radios', () => {
    const { container } = render(
      <TrainingEditModal session={SESSION} onClose={() => {}} onSaved={() => {}} />,
    )
    fireEvent.click(screen.getByLabelText('Begründung bei Absage erforderlich'))

    const playersDeclined = container.querySelector<HTMLInputElement>('input[name="rsvp-players"][value="declined"]')!
    const extDeclined = container.querySelector<HTMLInputElement>('input[name="rsvp-extended"][value="declined"]')!
    expect(playersDeclined.disabled).toBe(true)
    expect(extDeclined.disabled).toBe(true)
  })

  test('gewählte „abgesagt"-Voreinstellung deaktiviert die Reason-Checkbox', () => {
    const { container } = render(
      <TrainingEditModal session={{ ...SESSION, rsvp_default_players: 'declined' }} onClose={() => {}} onSaved={() => {}} />,
    )
    const reason = screen.getByLabelText('Begründung bei Absage erforderlich') as HTMLInputElement
    expect(reason.disabled).toBe(true)
    // Nach Zurückschalten auf „zugesagt" ist die Checkbox wieder aktiv.
    const playersConfirmed = container.querySelector<HTMLInputElement>('input[name="rsvp-players"][value="confirmed"]')!
    fireEvent.click(playersConfirmed)
    expect((screen.getByLabelText('Begründung bei Absage erforderlich') as HTMLInputElement).disabled).toBe(false)
  })
})
