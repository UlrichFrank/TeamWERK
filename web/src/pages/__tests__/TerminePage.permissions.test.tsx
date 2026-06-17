/**
 * TerminePage inline gate: isTrainer = admin || trainer || sportliche_leitung
 * Wenn isTrainer = true, werden RSVP-Buttons pro Termin ausgeblendet.
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 *
 * Hinweis: Die TerminePage hat keinen "Training anlegen"-Button — die spec.md §design.md §5
 * nennt ihn als Ziel, er existiert aber in der aktuellen Codebasis noch nicht.
 * Stattdessen wird ein aktives Training gemockt und geprüft, ob RSVP-Buttons erscheinen.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import TerminePage from '../TerminePage'
import { renderAsPersona } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const SESSION_FIXTURE = [
  {
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
    my_rsvp: null,
    note: '',
    cancel_reason: '',
    rsvp_opt_out: 0,
    rsvp_require_reason: 0,
    // Eltern sehen RSVP-Buttons über children_rsvp
    children_rsvp: [{ member_id: 1, name: 'Kind A', rsvp: null }],
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

describe('TerminePage — isTrainer-Gate: RSVP-Buttons', () => {
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<TerminePage />, persona.id, {
      mocks: [
        // Regex matcht /training-sessions?from=...&to=... (Listenaufruf ohne ID)
        { url: /\/training-sessions\?/, data: SESSION_FIXTURE },
        { url: /\/games\/my/, data: [] },
        { url: /\/teams/, data: [] },
      ],
    })

    // Seite muss ohne Crash rendern (heading immer vorhanden)
    expect(screen.getByRole('heading', { name: /Termine/i })).toBeInTheDocument()

    // RSVP-Buttons (Zusagen) werden nur für nicht-isTrainer Personas gerendert
    // ({s.status === 'active' && !isTrainer && <RsvpButton .../>})
    if (IS_TRAINER_IDS.includes(persona.id)) {
      // Wait for session data to render, then verify no RSVP button
      await screen.findByText('18:00 – 19:30')
      expect(
        screen.queryByText('Zusagen'),
        `Persona ${persona.id} (isTrainer): kein RSVP-Button erwartet`,
      ).toBeNull()
    } else {
      // Wait for RSVP button to appear
      expect(
        await screen.findByText('Zusagen'),
        `Persona ${persona.id} (kein isTrainer): RSVP-Button erwartet`,
      ).not.toBeNull()
    }
  })
})
