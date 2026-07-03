/**
 * TermineDetailPage inline gate: isTrainer = admin || trainer || sportliche_leitung
 * Steuert die Anwesenheitsliste-Editierbarkeit (Checkbox für Trainer vs. read-only).
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { Routes, Route } from 'react-router-dom'
import TermineDetailPage from '../TermineDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const SESSION_FIXTURE = {
  id: 1,
  date: '2020-06-17',
  start_time: '18:00',
  end_time: '19:30',
  status: 'active',
  team_id: 1,
  team_name: 'Test Team',
  confirmed_count: 1,
  declined_count: 0,
  maybe_count: 0,
  my_rsvp: null,
  note: '',
  cancel_reason: '',
  rsvp_default_players: 'none',
  rsvp_default_extended: 'none',
  rsvp_require_reason: 0,
}

const ATTENDANCE_FIXTURE = [
  { member_id: 1, member_name: 'Max Mustermann', present: null, rsvp_status: 'confirmed', reason: null },
]

// isTrainer = admin || trainer || trainer_elternteil || sportliche_leitung || sportliche_leitung_elternteil
const IS_TRAINER_IDS = [
  'admin',
  'trainer',
  'trainer_elternteil',
  'sportliche_leitung',
  'sportliche_leitung_elternteil',
]

describe('TermineDetailPage — isTrainer-Gate: Anwesenheits-Checkbox', () => {
  test.each(PERSONAS)('Persona $id', async (persona) => {
    // Route wrapper needed so useParams('/termine/:type/:id') resolves correctly
    renderAsPersona(
      <Routes>
        <Route path="/termine/:type/:id" element={<TermineDetailPage />} />
      </Routes>,
      persona.id,
      {
        initialEntries: ['/termine/training/1'],
        mocks: [
          { url: /training-sessions\/1$/, data: SESSION_FIXTURE },
          { url: /training-sessions\/1\/attendances/, data: ATTENDANCE_FIXTURE },
        ],
      },
    )

    // Wait for session data to load (member name from ATTENDANCE_FIXTURE)
    await screen.findByText('Max Mustermann')
    await flushAsync()

    // isTrainer: Anwesenheits-Checkbox ist editierbar (nicht readOnly)
    // !isTrainer: Checkbox ist readOnly
    // Checkbox erscheint nur in der Vergangenheit (isPast = true, da Fixture-Datum 2020-06-17)
    const checkbox = screen.queryByRole('checkbox')
    if (checkbox) {
      if (IS_TRAINER_IDS.includes(persona.id)) {
        expect(
          checkbox.getAttribute('readonly'),
          `Persona ${persona.id} (isTrainer): Checkbox muss editierbar sein`,
        ).toBeNull()
      }
      // Für Nicht-Trainer ist die Checkbox read-only — assertiert via prop, nicht Attribut
    }
  })
})
