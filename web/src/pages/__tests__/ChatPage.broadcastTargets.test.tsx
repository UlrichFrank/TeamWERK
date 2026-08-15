/**
 * Der Composer schickt genau eine der vier vereinsweiten Zielgruppen und meldet
 * die Empfängerzahl zurück.
 *
 * Der Empfängerzähler ist kein Komfort-Feature: ohne ihn sieht eine Mitteilung,
 * die niemanden erreicht, für den Absender exakt so aus wie eine zugestellte —
 * genau der stille Fehler, der diesen Change ausgelöst hat (targetType 'role'
 * löste gegen users.role auf und traf immer null Empfänger).
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

/** Öffnet den Composer als Vorstand und liefert die zuletzt gesendete Payload. */
async function openComposer(recipients: number) {
  renderAsPersona(<ChatPage />, 'vorstand')
  await flushAsync()

  // Der Catch-all aus setupApiMock ist zuerst registriert und gewinnt sonst —
  // axios-mock-adapter matcht in Registrierungsreihenfolge. Die Initial-Loads
  // sind durch das flushAsync oben schon durch.
  const mock = getApiMock()
  mock.reset()
  const sent: Record<string, unknown>[] = []
  mock.onPost('/chat/broadcasts').reply((config) => {
    sent.push(JSON.parse(config.data as string))
    return [201, { id: 1, recipients }]
  })
  mock.onAny().reply(200, [])

  fireEvent.click(screen.getByText('Mitteilungen'))
  await flushAsync()
  fireEvent.click(screen.getByText('Mitteilung senden'))
  await flushAsync()

  return sent
}

async function submitWith(target: string, recipients = 5) {
  const sent = await openComposer(recipients)

  fireEvent.change(screen.getByLabelText('Zielgruppe'), {
    target: { value: target },
  })
  fireEvent.change(screen.getByPlaceholderText('Deine Mitteilung…'), {
    target: { value: 'Testnachricht' },
  })
  // Der Submit-Button trägt denselben Text wie der Öffner — der letzte im DOM
  // ist der im Modal.
  const buttons = screen.getAllByText('Mitteilung senden')
  fireEvent.click(buttons[buttons.length - 1])
  await flushAsync()

  return sent
}

describe('BroadcastComposer — Zielgruppen-Payload', () => {
  beforeEach(() => {
    vi.useRealTimers()
  })

  test.each(['users', 'members', 'spieler', 'eltern'])(
    'Zielgruppe %s wird als targetType gesendet',
    async (target) => {
      const sent = await submitWith(target)

      expect(sent, `Zielgruppe ${target}: kein POST abgesetzt`).toHaveLength(1)
      expect(sent[0].targetType).toBe(target)
      // Die Alt-Felder sind ersatzlos entfallen und dürfen nicht mitreisen.
      expect(sent[0]).not.toHaveProperty('targetId')
      expect(sent[0]).not.toHaveProperty('targetRole')
    },
  )

  test('Erfolgsmeldung nennt die zurückgegebene Empfängerzahl', async () => {
    await submitWith('spieler', 183)

    expect(
      screen.queryByText(/183 Empfänger/),
      'Empfängerzahl aus der Server-Antwort fehlt in der Erfolgsmeldung',
    ).not.toBeNull()
  })

  test('Null Empfänger erzeugt einen sichtbaren Hinweis, keinen Fehler', async () => {
    await submitWith('eltern', 0)

    const hint = screen.queryByRole('status')
    expect(
      hint,
      'Leere Zielgruppe muss sichtbar gemeldet werden — sonst ist ein Fan-out ins Leere nicht erkennbar',
    ).not.toBeNull()
    expect(hint?.textContent).toMatch(/niemanden/)
  })
})
