/**
 * Dienst-Slots werden ausschließlich über das Kalender-Modal (SpieltagDetailModal,
 * "Dienst bearbeiten"-Dialog) gelöscht — /dienste bietet absichtlich KEINEN
 * Löschen-Button mehr, damit hier nicht versehentlich gelöscht werden kann
 * (unabhängig von der manage_duties-Capability). DutySlotList selbst trägt
 * seit diesem Change keinen Löschpfad mehr.
 * Quelle: openspec/specs/me-capabilities/spec.md (Capability-Vokabular)
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
    team_ids: [1],
    date: '2030-06-17',
    event_time: '10:00',
    opponent: null,
    event_type: 'generisch',
    team_names: ['Test Team'],
    label: 'Testveranstaltung',
    past: false,
    slots: [
      {
        id: 1,
        duty_type: 'Einlass',
        event_time: '10:00',
        hours_value: 1,
        slots_total: 2,
        vacancies: 2,
        claimed_by_me: false,
      },
    ],
  },
]

describe('DutyPage — kein Löschen-Button (Löschen läuft nur über den Kalender)', () => {
  test.each(PERSONAS)('Persona $id sieht keinen Slot-Löschen-Button', async (persona) => {
    renderAsPersona(<DutyPage />, persona.id, {
      mocks: [
        { url: /duty-board/, data: DUTY_BOARD_FIXTURE },
        { url: /teams/, data: [] },
        { url: /family\/proxy-accounts/, data: [] },
      ],
    })

    await screen.findByText('Einlass')

    expect(screen.queryByLabelText('Slot löschen')).toBeNull()
    expect(screen.queryByText('Löschen')).toBeNull()
  })
})
