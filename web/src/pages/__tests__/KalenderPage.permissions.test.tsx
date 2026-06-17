/**
 * KalenderPage inline gates:
 *   canEdit = admin || vorstand || trainer || sportliche_leitung → "Event"-Button
 *   canCreateAbsence = spieler || isParent → "Abwesenheit"-Button
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import KalenderPage from '../KalenderPage'
import { renderAsPersona } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

// canEdit = admin || vorstand(alle) || trainer(alle) || sportliche_leitung(alle)
const CAN_EDIT_IDS = [
  'admin',
  'vorstand',
  'vorstand_elternteil',
  'trainer',
  'trainer_elternteil',
  'sportliche_leitung',
  'sportliche_leitung_elternteil',
]

// canCreateAbsence = spieler || isParent
// isParent: vorstand_elternteil, trainer_elternteil, sportliche_leitung_elternteil, elternteil
// spieler: spieler
const CAN_CREATE_ABSENCE_IDS = [
  'vorstand_elternteil',
  'trainer_elternteil',
  'sportliche_leitung_elternteil',
  'spieler',
  'elternteil',
]

describe('KalenderPage — canEdit-Gate: "Event"-Button', () => {
  test.each(PERSONAS)('Persona $id', (persona) => {
    renderAsPersona(<KalenderPage />, persona.id, {
      mocks: [
        { url: /\/games/, data: [] },
        { url: /\/training-sessions/, data: [] },
        { url: /\/teams/, data: [] },
        { url: /\/absences/, data: [] },
      ],
    })

    // canEdit → aria-label="Event"
    const eventBtn = screen.queryByRole('button', { name: /^Event$/i })
    if (CAN_EDIT_IDS.includes(persona.id)) {
      expect(
        eventBtn,
        `Persona ${persona.id} (canEdit): "Event"-Button muss vorhanden sein`,
      ).not.toBeNull()
    } else if (!CAN_CREATE_ABSENCE_IDS.includes(persona.id)) {
      // Weder canEdit noch canCreateAbsence → gar kein Plus-Button
      expect(
        eventBtn,
        `Persona ${persona.id}: kein "Event"-Button erwartet`,
      ).toBeNull()
    }
  })
})

describe('KalenderPage — canCreateAbsence-Gate: "Abwesenheit"-Button', () => {
  test.each(PERSONAS)('Persona $id', (persona) => {
    renderAsPersona(<KalenderPage />, persona.id, {
      mocks: [
        { url: /\/games/, data: [] },
        { url: /\/training-sessions/, data: [] },
        { url: /\/teams/, data: [] },
        { url: /\/absences/, data: [] },
      ],
    })

    // !canEdit && canCreateAbsence → aria-label="Abwesenheit"
    const absBtn = screen.queryByRole('button', { name: /^Abwesenheit$/i })
    const hasOnlyAbsence = !CAN_EDIT_IDS.includes(persona.id) && CAN_CREATE_ABSENCE_IDS.includes(persona.id)
    if (hasOnlyAbsence) {
      expect(
        absBtn,
        `Persona ${persona.id} (nur canCreateAbsence): "Abwesenheit"-Button muss vorhanden sein`,
      ).not.toBeNull()
    } else if (!CAN_EDIT_IDS.includes(persona.id) && !CAN_CREATE_ABSENCE_IDS.includes(persona.id)) {
      // Weder canEdit noch canCreateAbsence → kein Button
      expect(
        absBtn,
        `Persona ${persona.id}: kein "Abwesenheit"-Button erwartet`,
      ).toBeNull()
    }
  })
})
