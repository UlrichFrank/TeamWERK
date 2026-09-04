/**
 * Der Composer schickt die angekreuzten Ziele als targets-Array und meldet die
 * Empfängerzahl zurück.
 *
 * Der Empfängerzähler ist kein Komfort-Feature: ohne ihn sieht eine Mitteilung,
 * die niemanden erreicht, für den Absender exakt so aus wie eine zugestellte —
 * genau der stille Fehler, der mitteilung-zielgruppen ausgelöst hat (targetType
 * 'role' löste gegen users.role auf und traf immer null Empfänger).
 *
 * Die Ziel-Liste kommt vom Server: welche Teams ein Trainer betreut, steht
 * nicht im JWT, und /chat/team-groups zeigt bewusst mehr.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

const CLUB_WIDE = [
  { kind: 'users', teamId: null, label: 'Alle Nutzer', count: 183 },
  { kind: 'members', teamId: null, label: 'Alle Mitglieder', count: 141 },
  { kind: 'spieler', teamId: null, label: 'Alle Spieler', count: 98 },
  { kind: 'eltern', teamId: null, label: 'Alle Eltern', count: 62 },
]

const TEAM_GROUPS = [
  { kind: 'alle_trainer', teamId: null, label: 'Alle Trainer', count: 22 },
  { kind: 'team_spieler', teamId: 7, label: 'Spieler mB1', count: 14 },
  { kind: 'team_eltern', teamId: 7, label: 'Eltern mB1', count: 11 },
]

/**
 * Öffnet den Composer für eine Persona und liefert die zuletzt gesendete
 * Payload.
 */
async function openComposer(
  recipients: number,
  persona: 'vorstand' | 'trainer' = 'vorstand',
  targets: unknown[] = [...CLUB_WIDE, ...TEAM_GROUPS],
) {
  renderAsPersona(<ChatPage />, persona)
  await flushAsync()

  // Der Catch-all aus setupApiMock ist zuerst registriert und gewinnt sonst —
  // axios-mock-adapter matcht in Registrierungsreihenfolge. Die Initial-Loads
  // sind durch das flushAsync oben schon durch.
  const mock = getApiMock()
  mock.reset()
  const sent: Record<string, unknown>[] = []
  mock.onGet('/chat/broadcast-targets').reply(200, targets)
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

/** Kreuzt die Ziele an, tippt einen Text und sendet. */
async function submitWith(
  labels: string[],
  recipients = 5,
  persona: 'vorstand' | 'trainer' = 'vorstand',
) {
  const sent = await openComposer(recipients, persona)

  for (const label of labels) {
    fireEvent.click(screen.getByLabelText(label, { exact: false }))
  }
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

  test.each([
    ['Alle Nutzer', 'users'],
    ['Alle Mitglieder', 'members'],
    ['Alle Spieler', 'spieler'],
    ['Alle Eltern', 'eltern'],
  ])('Zielgruppe %s wird als targets-Eintrag gesendet', async (label, kind) => {
    const sent = await submitWith([label])

    expect(sent, `Zielgruppe ${kind}: kein POST abgesetzt`).toHaveLength(1)
    expect(sent[0].targets).toEqual([{ kind, teamId: null }])
    // Das Alt-Feld ist ersatzlos entfallen und darf nicht mitreisen.
    expect(sent[0]).not.toHaveProperty('targetType')
  })

  test('Mehrfachauswahl landet vollständig im Request', async () => {
    const sent = await submitWith(['Spieler mB1', 'Eltern mB1'])

    expect(sent).toHaveLength(1)
    expect(sent[0].targets).toEqual([
      { kind: 'team_spieler', teamId: 7 },
      { kind: 'team_eltern', teamId: 7 },
    ])
  })

  test('Team-Gruppen stehen getrennt von den vereinsweiten Zielen', async () => {
    await openComposer(5)

    expect(screen.queryByText('Vereinsweit')).not.toBeNull()
    expect(screen.queryByText('Gruppen')).not.toBeNull()
  })

  test('Ohne angekreuztes Ziel bleibt der Senden-Button gesperrt', async () => {
    await openComposer(5)

    fireEvent.change(screen.getByPlaceholderText('Deine Mitteilung…'), {
      target: { value: 'Testnachricht' },
    })
    const buttons = screen.getAllByText(
      'Mitteilung senden',
    ) as HTMLButtonElement[]
    expect(
      buttons[buttons.length - 1].disabled,
      'ohne Ziel darf nicht gesendet werden können',
    ).toBe(true)
  })

  test('Leere Ziel-Liste zeigt einen Hinweis statt einer leeren Auswahl', async () => {
    await openComposer(5, 'trainer', [])

    expect(
      screen.queryByText(/keine Gruppen/i),
      'ein Trainer ohne Kader braucht eine Erklärung, keinen leeren Kasten',
    ).not.toBeNull()
  })

  test('Erfolgsmeldung nennt die zurückgegebene Empfängerzahl', async () => {
    await submitWith(['Alle Spieler'], 183)

    expect(
      screen.queryByText(/183 Empfänger/),
      'Empfängerzahl aus der Server-Antwort fehlt in der Erfolgsmeldung',
    ).not.toBeNull()
  })

  test('Null Empfänger erzeugt einen sichtbaren Hinweis, keinen Fehler', async () => {
    await submitWith(['Alle Eltern'], 0)

    const hint = screen.queryByRole('status')
    expect(
      hint,
      'Leere Zielgruppe muss sichtbar gemeldet werden — sonst ist ein Fan-out ins Leere nicht erkennbar',
    ).not.toBeNull()
    expect(hint?.textContent).toMatch(/niemanden/)
  })
})
