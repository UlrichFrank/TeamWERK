/**
 * TerminePage: Trainer sehen RSVP-Buttons auf der Kartenliste, wenn sie Teilnehmer
 * des Termins sind (my_rsvp != null, z. B. Trainer-Default 'confirmed'). Für fremde
 * Termine (my_rsvp = null) bleiben die Buttons ausgeblendet.
 * Quelle: openspec/changes/termine-rsvp-nachbesserung/specs/trainer-rsvp/spec.md
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import TerminePage from '../TerminePage'
import { renderAsPersona } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const baseSession = {
  id: 1,
  date: '2030-06-17',
  start_time: '18:00',
  end_time: '19:30',
  status: 'active',
  team_id: 1,
  team_name: 'Test Team',
  confirmed_count: 0,
  declined_count: 0,
  maybe_count: 0,
  note: '',
  cancel_reason: '',
  rsvp_default_players: 'none',
  rsvp_default_extended: 'none',
  rsvp_require_reason: 0,
}

function renderTrainer(session: Record<string, unknown>) {
  renderAsPersona(<TerminePage />, 'trainer', {
    mocks: [
      { url: /\/training-sessions\?/, data: [session] },
      { url: /\/games\/my/, data: [] },
      { url: /\/teams/, data: [] },
    ],
  })
}

describe('TerminePage — Trainer-RSVP-Buttons', () => {
  test('Trainer eines Team-Termins (my_rsvp=confirmed) sieht Buttons', async () => {
    renderTrainer({ ...baseSession, my_rsvp: 'confirmed', my_rsvp_is_default: true })
    expect(await screen.findByText('Zusagen')).not.toBeNull()
  })

  test('Trainer ohne Teilnahme am Termin (my_rsvp=null) sieht keine Buttons', async () => {
    renderTrainer({ ...baseSession, my_rsvp: null })
    await screen.findByText('18:00 – 19:30')
    expect(screen.queryByText('Zusagen')).toBeNull()
  })
})
