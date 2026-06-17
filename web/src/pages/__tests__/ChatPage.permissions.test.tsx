/**
 * ChatPage inline gates:
 *   canBroadcast = admin || vorstand || trainer || sportliche_leitung
 *   → "Mitteilung senden"-Button im Mitteilungen-Tab
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

// canBroadcast = admin || vorstand || vorstand_elternteil || trainer || trainer_elternteil
//              || sportliche_leitung || sportliche_leitung_elternteil
const CAN_BROADCAST_IDS = [
  'admin',
  'vorstand',
  'vorstand_elternteil',
  'trainer',
  'trainer_elternteil',
  'sportliche_leitung',
  'sportliche_leitung_elternteil',
]

describe('ChatPage — canBroadcast-Gate: "Mitteilung senden"-Button', () => {
  test.each(PERSONAS)('Persona $id', (persona) => {
    renderAsPersona(<ChatPage />, persona.id)

    // Mitteilungen-Tab anklicken (initial ist "Chats" aktiv)
    const mittelungenTab = screen.queryByText('Mitteilungen')
    if (mittelungenTab) {
      fireEvent.click(mittelungenTab)
    }

    const btn = screen.queryByText('Mitteilung senden')
    if (CAN_BROADCAST_IDS.includes(persona.id)) {
      expect(
        btn,
        `Persona ${persona.id} (canBroadcast): "Mitteilung senden" muss sichtbar sein`,
      ).not.toBeNull()
    } else {
      expect(
        btn,
        `Persona ${persona.id} (kein canBroadcast): "Mitteilung senden" darf nicht sichtbar sein`,
      ).toBeNull()
    }
  })
})
