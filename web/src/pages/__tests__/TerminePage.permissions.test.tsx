/**
 * TerminePage inline gate: isTrainer = admin || trainer || sportliche_leitung
 * Wenn isTrainer = true UND weder eigener my_rsvp gesetzt NOCH Elternteil,
 * werden RSVP-Buttons pro Termin ausgeblendet. Trainer-Elternteile sehen
 * die Kind-Buttons (Fix „Trainer-Eltern sehen für Kind nichts").
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
    rsvp_default_players: 'none',
    rsvp_default_extended: 'none',
    rsvp_require_reason: 0,
    // Eltern sehen RSVP-Buttons über children_rsvp
    children_rsvp: [{ member_id: 1, name: 'Kind A', rsvp: null }],
  },
]

// Fixture: Nutzer ist nicht selbst Teilnehmer (my_rsvp=null), aber es gibt
// ein Kind. Buttons erscheinen daher nur für Elternteile (Kind-Zeile).
describe('TerminePage — RSVP-Buttons-Sichtbarkeit', () => {
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

    if (persona.isParent) {
      // Elternteile sehen den Kind-Button
      expect(
        await screen.findByText('Zusagen'),
        `Persona ${persona.id} (Elternteil): Kind-RSVP-Button erwartet`,
      ).not.toBeNull()
    } else {
      // Nicht-Elternteile ohne eigene Teilnahme (my_rsvp=null) sehen keine Buttons
      await screen.findByText('18:00 – 19:30')
      expect(
        screen.queryByText('Zusagen'),
        `Persona ${persona.id} (kein Elternteil, my_rsvp=null): kein Button erwartet`,
      ).toBeNull()
    }
  })
})
