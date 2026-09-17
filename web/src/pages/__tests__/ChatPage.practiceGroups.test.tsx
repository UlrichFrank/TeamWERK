/**
 * Regression: Übungsgruppen tauchen in /chat/team-groups als groupType
 * "practice" auf (kader.id statt teams.id, siehe internal/chat/practice_groups.go),
 * müssen für die Mitgliederauflösung aber über /chat/practice-groups/{id}/{kind}
 * laufen — die Route /chat/team-groups/{teamId}/{kind} joint auf `teams` und
 * kennt Übungsgruppen strukturell nicht (kein teams-Zwilling), sonst bekommt
 * ein Trainer beim Auswählen seiner eigenen Übungsgruppe "forbidden".
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

describe('NewConversationModal — Übungsgruppen-Auflösung', () => {
  test('löst eine Übungsgruppen-Kachel über /chat/practice-groups auf, nicht /chat/team-groups', async () => {
    renderAsPersona(<ChatPage />, 'trainer')
    await flushAsync()

    const mock = getApiMock()
    mock.reset()
    mock.onGet('/chat/team-groups').reply(200, [
      {
        groupType: 'practice',
        teamId: 5,
        displayShort: 'Torwarttraining',
        kind: 'spieler',
        count: 3,
      },
    ])
    mock
      .onGet('/chat/practice-groups/5/spieler/members')
      .reply(200, [{ id: 42, name: 'Erika Musterspielerin' }])
    // Die falsche (Team-)Route existiert für eine Übungsgruppe nicht — bliebe
    // addTeamGroup fälschlich dabei, würde dieser 403 die Auflösung sichtbar
    // scheitern lassen.
    mock
      .onGet('/chat/team-groups/5/spieler/members')
      .reply(403, { error: 'forbidden' })
    // Catch-all zuletzt registriert — axios-mock-adapter matcht in
    // Registrierungsreihenfolge, ein früher registrierter Catch-all würde die
    // spezifischeren Handler oben sonst nie erreichen lassen.
    mock.onAny().reply(200, [])

    fireEvent.click(screen.getByText('Neues Gespräch'))
    await flushAsync()
    fireEvent.click(screen.getByText('Gruppe'))
    await flushAsync()

    fireEvent.click(screen.getByText(/Torwarttraining/))
    await flushAsync()

    expect(
      screen.queryByText('Erika Musterspielerin'),
      'Übungsgruppen-Mitglied muss nach Auswahl der Kachel als Chip erscheinen',
    ).not.toBeNull()
    expect(
      screen.queryByText(/konnte nicht aufgelöst werden/),
      'die Auflösung darf nicht über die Team-Route (403) laufen',
    ).toBeNull()
  })
})
