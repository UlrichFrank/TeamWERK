/**
 * KalenderPage inline gates:
 *   canEdit = admin || vorstand || trainer || sportliche_leitung → "Event"-Button
 *   canCreateAbsence = spieler || isParent → "Abwesenheit"-Button
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import KalenderPage from '../KalenderPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
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
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<KalenderPage />, persona.id, {
      mocks: [
        { url: /\/games/, data: [] },
        { url: /\/training-sessions/, data: [] },
        { url: /\/teams/, data: [] },
        { url: /\/absences/, data: [] },
      ],
    })
    await flushAsync()

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
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<KalenderPage />, persona.id, {
      mocks: [
        { url: /\/games/, data: [] },
        { url: /\/training-sessions/, data: [] },
        { url: /\/teams/, data: [] },
        { url: /\/absences/, data: [] },
      ],
    })
    await flushAsync()

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

// canImportGames = import_games (nur Vorstand/Admin, enger als manage_games).
const CAN_IMPORT_IDS = ['admin', 'vorstand', 'vorstand_elternteil']

describe('KalenderPage — import_games-Gate: H4A-Import im Aktionsmenü', () => {
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<KalenderPage />, persona.id, {
      mocks: [
        { url: /\/games/, data: [] },
        { url: /\/training-sessions/, data: [] },
        { url: /\/teams/, data: [] },
        { url: /\/absences/, data: [] },
      ],
    })
    await flushAsync()

    const menuBtn = screen.queryByRole('button', { name: /Weitere Aktionen/i })
    if (!CAN_IMPORT_IDS.includes(persona.id)) {
      expect(
        menuBtn,
        `Persona ${persona.id}: ohne import_games kein Aktionsmenü erwartet`,
      ).toBeNull()
      return
    }

    expect(
      menuBtn,
      `Persona ${persona.id} (import_games): Aktionsmenü muss vorhanden sein`,
    ).not.toBeNull()

    // Der Import liegt hinter dem Dropdown — erst nach Klick sichtbar. Genau das
    // ist der Regressionsschutz: vorher war es ein eigener Button.
    expect(screen.queryByRole('menuitem', { name: /Handball4All/i })).toBeNull()
    await userEvent.click(menuBtn!)
    expect(
      screen.queryByRole('menuitem', { name: /Handball4All/i }),
      `Persona ${persona.id}: H4A-Import muss im geöffneten Menü stehen`,
    ).not.toBeNull()
  })
})
