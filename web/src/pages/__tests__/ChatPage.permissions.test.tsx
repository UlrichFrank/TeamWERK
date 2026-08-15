/**
 * ChatPage inline gates:
 *   canBroadcast = admin || vorstand || trainer || sportliche_leitung
 *   → "Mitteilung senden"-Button im Mitteilungen-Tab
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

// canBroadcast = admin || vorstand || vorstand_elternteil
//              || sportliche_leitung || sportliche_leitung_elternteil
// Trainer sind bewusst NICHT dabei: der Empfängerkreis eines Teams ist über die
// Team-Standardgruppen des Chats erreichbar — mit Rückkanal.
const CAN_BROADCAST_IDS = [
  'admin',
  'vorstand',
  'vorstand_elternteil',
  'sportliche_leitung',
  'sportliche_leitung_elternteil',
]

describe('ChatPage — canBroadcast-Gate: "Mitteilung senden"-Button', () => {
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<ChatPage />, persona.id)
    await flushAsync()

    // Mitteilungen-Tab anklicken (initial ist "Chats" aktiv)
    const mittelungenTab = screen.queryByText('Mitteilungen')
    if (mittelungenTab) {
      fireEvent.click(mittelungenTab)
    }
    await flushAsync()

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

describe('BroadcastModal — Zielgruppen', () => {
  test('Reiner Trainer sieht den Button „Mitteilung senden" nicht', async () => {
    renderAsPersona(<ChatPage />, 'trainer')
    await flushAsync()

    fireEvent.click(screen.getByText('Mitteilungen'))
    await flushAsync()

    expect(
      screen.queryByText('Mitteilung senden'),
      'Trainer haben das Mitteilungsrecht verloren — Team-Ansagen laufen über die Team-Standardgruppen im Chat',
    ).toBeNull()
  })

  test('Sportliche Leitung sieht alle vier Zielgruppen', async () => {
    renderAsPersona(<ChatPage />, 'sportliche_leitung')
    await flushAsync()

    fireEvent.click(screen.getByText('Mitteilungen'))
    await flushAsync()
    fireEvent.click(screen.getByText('Mitteilung senden'))
    await flushAsync()

    for (const label of [
      'Alle Nutzer',
      'Alle Mitglieder',
      'Alle Spieler',
      'Alle Eltern',
    ]) {
      expect(
        screen.queryByRole('option', { name: label }),
        `Zielgruppe "${label}" fehlt im Composer`,
      ).not.toBeNull()
    }
    // Die Alt-Zielgruppe ist ersatzlos entfallen.
    expect(screen.queryByRole('option', { name: 'Team wählen…' })).toBeNull()
  })
})
