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

function renderTrainerElternteil(session: Record<string, unknown>) {
  renderAsPersona(<TerminePage />, 'trainer_elternteil', {
    mocks: [
      { url: /\/training-sessions\?/, data: [session] },
      { url: /\/games\/my/, data: [] },
      { url: /\/teams/, data: [] },
    ],
  })
}

describe('TerminePage — Trainer-RSVP-Buttons', () => {
  test('Trainer eines Team-Termins (am_i_participant=true) sieht Buttons', async () => {
    renderTrainer({ ...baseSession, my_rsvp: 'confirmed', my_rsvp_is_default: true, am_i_participant: true })
    expect(await screen.findByText('Zusagen')).not.toBeNull()
  })

  test('Trainer ohne Teilnahme am Termin (am_i_participant=false) sieht keine Buttons', async () => {
    renderTrainer({ ...baseSession, my_rsvp: null, am_i_participant: false })
    await screen.findByText('18:00 – 19:30')
    expect(screen.queryByText('Zusagen')).toBeNull()
  })

  test('Trainer-Elternteil mit Kind im Team sieht eigene UND Kind-Buttons', async () => {
    renderTrainerElternteil({
      ...baseSession,
      my_rsvp: 'confirmed',
      my_rsvp_is_default: true,
      am_i_participant: true,
      children_rsvp: [{ member_id: 99, name: 'Lias Muster', rsvp: null }],
    })
    // Eigene Zeile hat Label "Ich" und Buttons
    expect(await screen.findByText('Ich')).not.toBeNull()
    expect(screen.getByText('Lias Muster')).not.toBeNull()
    // Beide Zeilen → je 1x "Zusagen" pro Zeile
    expect(screen.getAllByText('Zusagen').length).toBe(2)
  })

  test('Trainer-Elternteil ohne Kind im Team sieht trotzdem eigene Buttons', async () => {
    renderTrainerElternteil({
      ...baseSession,
      my_rsvp: 'confirmed',
      my_rsvp_is_default: true,
      am_i_participant: true,
      // kein children_rsvp → Kind nicht im Team
    })
    expect(await screen.findByText('Zusagen')).not.toBeNull()
    // Kein "Ich"-Label, da keine Kind-Zeile daneben steht
    expect(screen.queryByText('Ich')).toBeNull()
  })
})
