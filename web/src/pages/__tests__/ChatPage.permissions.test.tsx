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

// canBroadcast = admin || vorstand || trainer || sportliche_leitung (je + Elternteil)
//
// Trainer sind seit mitteilung-team-gruppen wieder dabei. Die Capability ist
// grob und sagt nur "darf senden"; an welche Gruppen, entscheidet der Server
// anhand der Kader (chat.allowedTargets) — der Composer holt die Liste über
// GET /chat/broadcast-targets und zeigt einen Hinweis, wenn sie leer ist.
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

describe('BroadcastModal — Zielgruppen', () => {
  test('Reiner Trainer sieht den Button „Mitteilung senden"', async () => {
    renderAsPersona(<ChatPage />, 'trainer')
    await flushAsync()

    fireEvent.click(screen.getByText('Mitteilungen'))
    await flushAsync()

    expect(
      screen.queryByText('Mitteilung senden'),
      'Trainer dürfen seit mitteilung-team-gruppen an die Gruppen ihrer eigenen Kader senden',
    ).not.toBeNull()
  })

  test('Der Composer zeigt die Ziele, die der Server liefert', async () => {
    renderAsPersona(<ChatPage />, 'sportliche_leitung', {
      mocks: [
        {
          method: 'get',
          url: '/chat/broadcast-targets',
          data: [
            { kind: 'users', teamId: null, label: 'Alle Nutzer', count: 183 },
            {
              kind: 'members',
              teamId: null,
              label: 'Alle Mitglieder',
              count: 141,
            },
            { kind: 'spieler', teamId: null, label: 'Alle Spieler', count: 98 },
            { kind: 'eltern', teamId: null, label: 'Alle Eltern', count: 62 },
          ],
        },
      ],
    })
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
        screen.queryByLabelText(label, { exact: false }),
        `Zielgruppe "${label}" fehlt im Composer`,
      ).not.toBeNull()
    }
  })
})
