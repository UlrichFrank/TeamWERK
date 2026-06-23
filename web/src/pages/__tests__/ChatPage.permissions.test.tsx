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

describe('BroadcastModal — Team-Dropdown sichtbar für Trainer', () => {
  test('Trainer ohne broadcast_all sieht Team-Auswahl direkt nach Modal-Öffnung', async () => {
    renderAsPersona(<ChatPage />, 'trainer', {
      mocks: [{ url: '/teams', data: [
        { id: 7, name: 'mA1', age_class: 'mA', gender: 'm', team_number: 1, group_count: 1 },
      ] }],
    })
    await flushAsync()

    fireEvent.click(screen.getByText('Mitteilungen'))
    await flushAsync()
    fireEvent.click(screen.getByText('Mitteilung senden'))
    await flushAsync()

    // Modal ist offen; das zweite <select> mit „Team wählen…" muss vorhanden sein.
    expect(
      screen.queryByRole('option', { name: 'Team wählen…' }),
      'Trainer-Modal: Team-Dropdown fehlt — vermutlich falscher targetType-Default',
    ).not.toBeNull()
  })

  test('Admin sieht „Alle Mitglieder" als Default-Zielgruppe (kein Team-Dropdown)', async () => {
    renderAsPersona(<ChatPage />, 'admin')
    await flushAsync()

    fireEvent.click(screen.getByText('Mitteilungen'))
    await flushAsync()
    fireEvent.click(screen.getByText('Mitteilung senden'))
    await flushAsync()

    expect(screen.queryByRole('option', { name: 'Team wählen…' })).toBeNull()
    expect(screen.queryByRole('option', { name: 'Alle Mitglieder' })).not.toBeNull()
  })
})
