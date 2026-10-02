/**
 * chat-gruppe-aktualisieren: Die im Dialog „Neues Gespräch" gewählten
 * Standard-Gruppen-Kacheln gehen als `sources` an POST /chat/conversations —
 * sie sind die Herkunft, gegen die „Aktualisieren" später abgleicht.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

describe('NewConversationModal — Herkunft der Gruppe', () => {
  test('schickt die gewählten Kacheln als sources mit', async () => {
    renderAsPersona(<ChatPage />, 'trainer')
    await flushAsync()

    const mock = getApiMock()
    mock.reset()
    mock.onGet('/chat/team-groups').reply(200, [
      { groupType: 'team', teamId: 7, displayShort: 'mC1', kind: 'spieler', count: 1 },
      { groupType: 'team', teamId: 7, displayShort: 'mC1', kind: 'eltern', count: 1 },
      // Gleiche ID, andere Art: darf nicht mit der Mannschaft verwechselt werden.
      { groupType: 'practice', teamId: 7, displayShort: 'Torwart', kind: 'spieler', count: 1 },
    ])
    mock.onGet('/chat/team-groups/7/spieler/members').reply(200, [{ id: 11, name: 'Sven Spieler' }])
    mock.onGet('/chat/team-groups/7/eltern/members').reply(200, [{ id: 12, name: 'Elke Eltern' }])
    let payload: Record<string, unknown> | null = null
    mock.onPost('/chat/conversations').reply((config) => {
      payload = JSON.parse(config.data)
      return [201, { id: 99, type: 'group', name: 'mC1', members: [] }]
    })
    mock.onAny().reply(200, [])

    fireEvent.click(screen.getByText('Neues Gespräch'))
    await flushAsync()
    fireEvent.click(screen.getByText('Gruppe'))
    await flushAsync()
    fireEvent.change(screen.getByPlaceholderText('Gruppenname'), { target: { value: 'mC1' } })
    fireEvent.click(screen.getByText('Spieler mC1'))
    await flushAsync()
    fireEvent.click(screen.getByText('Eltern mC1'))
    await flushAsync()

    expect(screen.queryByText('Spieler Torwart'), 'Übungsgruppe mit gleicher ID bleibt wählbar').not.toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Gespräch starten' }))
    await flushAsync()

    expect(payload).not.toBeNull()
    expect(payload!.memberIds).toEqual([11, 12])
    expect(payload!.sources).toEqual([
      { groupType: 'team', refId: 7, kind: 'spieler' },
      { groupType: 'team', refId: 7, kind: 'eltern' },
    ])
  })
})
