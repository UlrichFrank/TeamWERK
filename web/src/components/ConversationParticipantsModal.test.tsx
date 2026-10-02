import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import ConversationParticipantsModal from './ConversationParticipantsModal'
import { setupApiMock } from '../test/apiMock'

let currentUserId = 1
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: currentUserId } }),
}))

const members = [{ id: 1, name: 'Ersteller' }, { id: 2, name: 'Mitglied' }]

function renderModal() {
  return render(
    <ConversationParticipantsModal
      convId={5}
      initialName="mC1"
      createdBy={1}
      members={members}
      onClose={() => {}}
      onChanged={() => {}}
    />,
  )
}

describe('ConversationParticipantsModal — Aktualisieren', () => {
  beforeEach(() => {
    const mock = setupApiMock()
    mock.reset()
    mock.onPost('/chat/conversations/5/sync/preview').reply(200, {
      sources: [], add: [], remove: [], blocked: false, suggestions: [],
    })
  })

  test('Nicht-Ersteller sieht keinen Knopf', () => {
    currentUserId = 2
    renderModal()
    expect(screen.queryByLabelText('Aktualisieren')).toBeNull()
  })

  test('Ersteller öffnet das Abgleich-Modal', async () => {
    currentUserId = 1
    renderModal()
    fireEvent.click(screen.getByLabelText('Aktualisieren'))
    expect(await screen.findByText('Teilnehmer aktualisieren')).toBeTruthy()
  })
})
