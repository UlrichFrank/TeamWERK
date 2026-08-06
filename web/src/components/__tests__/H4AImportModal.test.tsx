/**
 * H4A-Import-Modal: Diff-Darstellung, Template-Batch vs. selektive Wahl,
 * Sperre nicht zugeordneter Zeilen.
 * Quelle: openspec/changes/h4a-import/specs/h4a-game-import/spec.md
 */
import { describe, test, expect } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import H4AImportModal, { type H4APlanGame } from '../H4AImportModal'
import { renderAsPersonaNoRouter, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

const TEAMS = [
  { id: 1, name: 'Männliche C', is_active: true },
  { id: 2, name: 'Weibliche D', is_active: true },
]

const TEMPLATES = [
  { id: 10, name: 'Heimspiel Standard', template_type: 'heim' },
  { id: 11, name: 'Heimspiel groß', template_type: 'heim' },
  { id: 12, name: 'Auswärts Standard', template_type: 'auswärts' },
]

function planGame(over: Partial<H4APlanGame>): H4APlanGame {
  return {
    game_no: '905996',
    staffel: 'mC-OL-3-BW',
    club_alias: 'Team Stuttgart',
    opponent: 'Bregenz Handb.',
    date: '2026-09-26',
    time: '14:45',
    is_home: true,
    event_type: 'heim',
    hall_number: '3059',
    team_id: 1,
    team_name: 'Männliche C',
    venue_id: 5,
    venue_name: 'Sporthalle Fixture',
    status: 'new',
    ...over,
  }
}

// Eine kombinierte Preview-Antwort: Schritt 1 (Perioden) und Schritt 2 (Plan)
// lesen daraus jeweils ihren Teil. Der Mock-Adapter kann pro URL nur eine
// statische Antwort liefern; die Trennung der beiden Aufrufe ist im Go-Test
// abgedeckt (internal/games/h4aimport_handler_test.go).
const PREVIEW = {
  needs_period: true,
  periods: [{ ID: '142', Name: 'Hallenrunde 26/27' }],
  new: [
    planGame({}),
    // Zeile ohne Mannschaftszuordnung → nicht bestätigbar.
    planGame({
      game_no: '211004', staffel: 'xY-UNBEKANNT', team_id: null, team_name: '',
      venue_id: null, venue_name: '', hall_number: '9999', is_home: false,
      event_type: 'auswärts', opponent: 'HSC Schm/Oeff',
      warnings: ['Mannschaft nicht zugeordnet', 'Halle 9999 unbekannt'],
    }),
  ],
  changed: [
    planGame({
      game_no: '214102', status: 'changed', existing_game_id: 77,
      changes: [{ field: 'time', old: '10:00', new: '11:45' }],
    }),
  ],
  unchanged: [planGame({ game_no: '905992', status: 'unchanged' })],
}

const BASE_MOCKS = [
  { method: 'get' as const, url: '/teams', data: TEAMS },
  { method: 'get' as const, url: '/duty-templates', data: TEMPLATES },
  { method: 'any' as const, url: '/games/import/h4a/preview', data: PREVIEW },
  { method: 'any' as const, url: '/games/import/h4a/apply', data: { imported: 2, updated: 0, skipped: 0 } },
]

function renderModal(onImported: (r: { imported: number; updated: number; skipped: number }) => void = () => {}) {
  return renderAsPersonaNoRouter(
    <H4AImportModal isOpen onClose={() => {}} onImported={onImported} />,
    'vorstand',
    { mocks: BASE_MOCKS },
  )
}

// Bringt das Modal von der Credentials-Eingabe in die Diff-Ansicht.
async function openDiff() {
  fireEvent.change(screen.getByLabelText('H4A-Benutzername'), { target: { value: 'v_109' } })
  fireEvent.change(screen.getByLabelText('H4A-Passwort'), { target: { value: 'geheim' } })
  fireEvent.click(screen.getByRole('button', { name: 'Verbinden' }))
  await flushAsync()
  fireEvent.click(screen.getByRole('button', { name: 'Spielplan laden' }))
  await flushAsync()
}

describe('H4AImportModal', () => {
  test('rendert die Diff-Abschnitte mit Alt→Neu und blendet Unverändert erst auf Klick ein', async () => {
    renderModal()
    await openDiff()

    expect(screen.getByText('Neu (2)')).toBeInTheDocument()
    expect(screen.getByText('Geändert (1)')).toBeInTheDocument()

    // Feld-Änderung mit Alt- und Neu-Wert.
    expect(screen.getByText('10:00')).toBeInTheDocument()
    expect(screen.getByText('11:45')).toBeInTheDocument()

    // Unveränderte Spiele sind zunächst zugeklappt.
    expect(screen.queryByText(/Nr\. 905992/)).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Unverändert \(1\)/ }))
    await flushAsync()
    expect(screen.getByText(/Nr\. 905992/)).toBeInTheDocument()
  })

  test('Zeile ohne Mannschaftszuordnung ist nicht bestätigbar', async () => {
    renderModal()
    await openDiff()

    const unresolved = screen.getByLabelText('Spiel 211004 übernehmen') as HTMLInputElement
    expect(unresolved.disabled).toBe(true)
    expect(unresolved.checked).toBe(false)
    expect(screen.getByText('Mannschaft nicht zugeordnet')).toBeInTheDocument()
    expect(screen.getByText('Halle 9999 unbekannt')).toBeInTheDocument()

    // Zugeordnete Zeilen sind vorausgewählt: 2 von 3 Zeilen importierbar.
    expect(screen.getByRole('button', { name: '2 Spiele importieren' })).toBeInTheDocument()
  })

  test('nachträgliche Mannschaftszuordnung schaltet die Zeile frei', async () => {
    renderModal()
    await openDiff()

    fireEvent.change(document.querySelector('#h4a-team-211004')!, { target: { value: '2' } })
    await flushAsync()

    const unresolved = screen.getByLabelText('Spiel 211004 übernehmen') as HTMLInputElement
    expect(unresolved.disabled).toBe(false)
    expect(screen.getByRole('button', { name: '3 Spiele importieren' })).toBeInTheDocument()
  })

  test('Batch-Template setzt alle Spiele eines Typs, selektive Wahl überschreibt', async () => {
    renderModal()
    await openDiff()

    // Batch für alle Heimspiele.
    fireEvent.change(screen.getByLabelText('Dienst-Vorlage für alle Heimspiele'), {
      target: { value: '10' },
    })
    await flushAsync()
    // Selektiv für ein einzelnes Heimspiel abweichend.
    fireEvent.change(document.querySelector('#h4a-tpl-214102')!, { target: { value: '11' } })
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: /Spiele importieren/ }))

    const mock = getApiMock()
    await waitFor(() => {
      expect(mock.history.post.some(r => r.url === '/games/import/h4a/apply')).toBe(true)
    })
    const applyReq = mock.history.post.find(r => r.url === '/games/import/h4a/apply')!
    const decisions = JSON.parse(applyReq.data as string).decisions as
      { game_no: string; template_id: number | null; team_id: number }[]

    const byNo = Object.fromEntries(decisions.map(d => [d.game_no, d.template_id]))
    expect(byNo['905996']).toBe(10) // Batch greift
    expect(byNo['214102']).toBe(11) // selektive Wahl überschreibt den Batch
    // Die nicht zugeordnete Zeile wird gar nicht erst mitgeschickt.
    expect(byNo['211004']).toBeUndefined()
  })

  test('meldet das Import-Ergebnis an die aufrufende Seite', async () => {
    const seen: { imported: number; updated: number; skipped: number }[] = []
    renderModal(r => seen.push(r))
    await openDiff()

    fireEvent.click(screen.getByRole('button', { name: /Spiele importieren/ }))
    await waitFor(() => expect(seen).toHaveLength(1))
    expect(seen[0].imported).toBe(2)
  })

  test('Login-Fehler zeigt eine generische Meldung ohne H4A-Interna', async () => {
    renderModal()
    // Handler in der richtigen Reihenfolge neu setzen: der Catch-all aus
    // setupApiMock würde sonst gewinnen (Registrierungsreihenfolge zählt).
    const mock = getApiMock()
    mock.reset()
    mock.onPost('/games/import/h4a/preview').reply(502, { error: 'h4a_login_failed' })
    mock.onAny().reply(200, [])

    fireEvent.change(screen.getByLabelText('H4A-Benutzername'), { target: { value: 'v_109' } })
    fireEvent.change(screen.getByLabelText('H4A-Passwort'), { target: { value: 'falsch' } })
    fireEvent.click(screen.getByRole('button', { name: 'Verbinden' }))
    await flushAsync()

    expect(screen.getByText(/Anmeldung bei Handball4All fehlgeschlagen/)).toBeInTheDocument()
    // Das Passwort darf nirgends als Text im DOM landen (nur im Passwortfeld).
    expect(screen.queryByText('falsch')).not.toBeInTheDocument()
  })

  test('rendert nichts, wenn isOpen false ist', () => {
    const { container } = renderAsPersonaNoRouter(
      <H4AImportModal isOpen={false} onClose={() => {}} onImported={() => {}} />,
      'vorstand',
      { mocks: BASE_MOCKS },
    )
    expect(container).toBeEmptyDOMElement()
  })
})
