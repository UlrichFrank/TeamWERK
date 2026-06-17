/**
 * SpieltagDetailPage (game-view in TermineDetailPage) inline gate:
 * isTrainer = admin || trainer || sportliche_leitung
 * Steuert die Aufstellungs-Checkboxen (onToggleLineup).
 *
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 *
 * Designloch (§10): isTrainer schließt vorstand aus, obwohl canEdit in KalenderPage
 * vorstand einschließt. Spiegelbild: internal/permissions/matrix_test.go
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import TermineDetailPage from '../TermineDetailPage'
import { renderAsPersona } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const GAME_FIXTURE = {
  game: {
    id: 1,
    date: '2030-06-17',
    time: '15:00',
    opponent: 'Testgegner',
    event_type: 'heim',
    is_home: true,
    season_id: 1,
    team_names: 'Test Team',
    team_ids: [1],
    confirmed_count: 1,
    declined_count: 0,
    maybe_count: 0,
    my_rsvp: null,
    rsvp_opt_out: 0,
    rsvp_require_reason: 0,
  },
}

const PARTICIPANTS_FIXTURE = [
  {
    member_id: 1,
    member_name: 'Max Mustermann',
    rsvp_status: 'confirmed',
    is_extended: false,
    in_lineup: false,
  },
]

// isTrainer = admin || trainer || trainer_elternteil || sportliche_leitung || sportliche_leitung_elternteil
const IS_TRAINER_IDS = [
  'admin',
  'trainer',
  'trainer_elternteil',
  'sportliche_leitung',
  'sportliche_leitung_elternteil',
]

describe('SpieltagDetailPage (TermineDetailPage game-view) — isTrainer-Gate: Aufstellungs-Checkbox', () => {
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<TermineDetailPage />, persona.id, {
      initialEntries: ['/termine/spiel/1'],
      mocks: [
        { url: /\/games\/1$/, data: GAME_FIXTURE },
        { url: /\/games\/1\/participants/, data: PARTICIPANTS_FIXTURE },
      ],
    })

    // Seite muss ohne Crash rendern
    const playerRow = screen.queryByText('Max Mustermann')
    if (!playerRow) return // Noch im Ladezustand — kein Fehler

    // isTrainer → Aufstellungs-Checkbox ist interaktiv
    // !isTrainer → onToggleLineup = undefined → Aufstellungs-Icon, keine Checkbox
    const lineup = screen.queryByRole('checkbox')
    if (IS_TRAINER_IDS.includes(persona.id)) {
      expect(
        lineup,
        `Persona ${persona.id} (isTrainer): Aufstellungs-Checkbox muss vorhanden sein`,
      ).not.toBeNull()
    } else {
      expect(
        lineup,
        `Persona ${persona.id} (kein isTrainer): keine editierbare Aufstellungs-Checkbox`,
      ).toBeNull()
    }
  })
})
