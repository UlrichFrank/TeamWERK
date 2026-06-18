/**
 * DutyPage Slot-Verwaltung: canManageDuties = manage_duties-Capability
 * (admin || vorstand || trainer || sportliche_leitung).
 * Steuert den Löschen-Button (Trash2-Icon) auf Duty-Slots.
 * Quelle: openspec/specs/me-capabilities/spec.md (Capability-Vokabular)
 *
 * Das frühere Designloch (vorstand sah keine Slot-Mutations) ist behoben:
 * vorstand ist jetzt — wie im Backend-Gate der duty-slots-Routen — eingeschlossen.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import DutyPage from '../DutyPage'
import { renderAsPersona } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const DUTY_BOARD_FIXTURE = [
  {
    game_id: null,
    team_id: 1,
    date: '2030-06-17',
    event_time: '10:00',
    opponent: null,
    event_type: 'generisch',
    team_name: 'Test Team',
    label: 'Testveranstaltung',
    past: false,
    slots: [
      {
        id: 1,
        duty_type: 'Einlass',
        event_time: '10:00',
        slots_total: 2,
        vacancies: 2,
        claimed_by_me: false,
      },
    ],
  },
]

// manage_duties = admin || vorstand || trainer || sportliche_leitung (inkl. Elternteil-Varianten)
const CAN_MANAGE_DUTIES_IDS = [
  'admin',
  'vorstand',
  'vorstand_elternteil',
  'trainer',
  'trainer_elternteil',
  'sportliche_leitung',
  'sportliche_leitung_elternteil',
]

describe('DutyPage — manage_duties-Gate: Slot-Löschen-Button', () => {
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<DutyPage />, persona.id, {
      mocks: [
        { url: /duty-board/, data: DUTY_BOARD_FIXTURE },
        { url: /teams/, data: [] },
        { url: /family\/proxy-accounts/, data: [] },
      ],
    })

    // Wait for slot data to load (duty_type renders as table cell text)
    await screen.findByText('Einlass')

    // "Slot löschen" aria-label ist nur gerendert wenn canEdit=true
    const deleteBtn = screen.queryByLabelText('Slot löschen')
    if (CAN_MANAGE_DUTIES_IDS.includes(persona.id)) {
      expect(
        deleteBtn,
        `Persona ${persona.id} (manage_duties): Löschen-Button muss vorhanden sein`,
      ).not.toBeNull()
    } else {
      expect(
        deleteBtn,
        `Persona ${persona.id} (kein manage_duties): kein Löschen-Button erwartet`,
      ).toBeNull()
    }
  })
})
