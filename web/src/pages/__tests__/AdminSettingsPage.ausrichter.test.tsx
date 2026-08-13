/**
 * Einstellungen-Tab „Heimspieltage": vormals „Bewirtung", jetzt zwei Kacheln
 * — „Bewirtung" (unverändert, siehe AdminSettingsPage.bewirtung.test.tsx) und
 * „Ausrichter" (neu, diese Datei). Deckt außerdem den alten Query-Parameter-
 * Alias `?tab=bewirtung` ab, der weiterhin auf dem umbenannten Tab landen muss.
 *
 * Quelle: openspec/changes/heimspieltag-ausrichter/specs/bewirtungsrotation/spec.md
 * — Requirement "Einstellungen-UI Tab „Bewirtung"" (MODIFIED),
 *   openspec/changes/heimspieltag-ausrichter/specs/heimspieltag-ausrichter/spec.md
 * — Requirement "Vereinsweite Ausrichter-Liste mit genau einem Default",
 *   Requirement "Löschen eines Ausrichters entkoppelt Spieltage, löscht aber
 *   gebundene Vorlagen-Zeilen".
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import AdminSettingsPage from '../AdminSettingsPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const AUSRICHTER = [
  { id: 1, name: 'TV Ötlingen', aktiv: true, is_default: true, sort_order: 0 },
  { id: 2, name: 'SV Fellbach', aktiv: true, is_default: false, sort_order: 1 },
]

/**
 * Registriert Basis-Mocks für den Tab und rendert die Seite direkt auf der
 * Zielroute (Standard: `?tab=heimspieltage`). `configure` hängt zusätzliche
 * Handler (z.B. für `PUT`/`DELETE /ausrichter/{id}`) an, BEVOR der
 * `onAny()`-Catch-all registriert wird — axios-mock-adapter matched Handler
 * in Registrierungsreihenfolge, ein später hinzugefügter `onPut`/`onGet`
 * würde vom bereits registrierten Catch-all maskiert (Vorbild
 * AdminSettingsPage.bewirtung.test.tsx).
 */
function renderHeimspieltage(route = '/einstellungen?tab=heimspieltage', configure?: (mock: ReturnType<typeof getApiMock>) => void) {
  renderAsPersona(<AdminSettingsPage />, 'vorstand', { route })
  const mock = getApiMock()
  mock.reset()
  mock.onGet('/club').reply(200, {})
  mock.onGet('/seasons').reply(200, [])
  mock.onGet('/fee-rates').reply(200, { items: [] })
  mock.onGet('/age-class-rules').reply(200, [])
  mock.onGet('/stammvereine').reply(200, { items: [] })
  mock.onGet('/settings/bewirtung').reply(200, { verhaeltnis: 1, max_per_team: 2 })
  mock.onGet('/ausrichter?include_inactive=1').reply(200, { items: AUSRICHTER })
  configure?.(mock)
  mock.onAny().reply(200, [])
  return mock
}

/**
 * Findet die Desktop-Tabellenzeile zu einem Ausrichter-Namen. Mobile-Card und
 * Desktop-Tabelle liegen in jsdom (keine CSS-Auswertung) beide gleichzeitig
 * im DOM — `getByText` allein wäre mehrdeutig, deshalb gezielt auf `<tr>`
 * eingrenzen.
 */
function ausrichterRow(name: string): HTMLElement {
  const cells = screen.getAllByText(name)
  const row = cells.map(c => c.closest('tr')).find((r): r is HTMLTableRowElement => r !== null)
  if (!row) throw new Error(`Keine Tabellenzeile für "${name}" gefunden`)
  return row
}

describe('AdminSettingsPage — Heimspieltage-Tab', () => {
  test('rendert beide Kacheln (Bewirtung + Ausrichter)', async () => {
    renderHeimspieltage()
    await flushAsync()

    expect(screen.getByRole('heading', { name: 'Bewirtung' })).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Ausrichter' })).toBeTruthy()
    expect(screen.getByLabelText('Kuchen je Spiel')).toBeTruthy()
    // Mobile-Card und Desktop-Tabelle rendern in jsdom (keine CSS-Auswertung)
    // beide gleichzeitig — deshalb getAllByText statt getByText.
    expect(screen.getAllByText('TV Ötlingen').length).toBeGreaterThan(0)
    expect(screen.getAllByText('SV Fellbach').length).toBeGreaterThan(0)
  })

  test('benennt den Wirkungsbereich (Dienst-Generierung bei Heimspielen)', async () => {
    renderHeimspieltage()
    await flushAsync()

    expect(screen.getByText(/Dienst-Generierung bei/)).toBeTruthy()
    expect(screen.getByText('Heimspielen')).toBeTruthy()
  })

  test('alter Query-Parameter ?tab=bewirtung landet auf dem umbenannten Tab', async () => {
    renderHeimspieltage('/einstellungen?tab=bewirtung')
    await flushAsync()

    // Der Tab-Button trägt jetzt das neue Label — der alte Parameterwert
    // wurde intern auf 'heimspieltage' aufgelöst.
    const tabButton = screen.getByRole('button', { name: 'Heimspieltage' })
    expect(tabButton.className).toContain('border-brand-yellow')
    expect(screen.getByRole('heading', { name: 'Ausrichter' })).toBeTruthy()
  })

  test('Ausrichter anlegen sendet POST /ausrichter mit dem Namen', async () => {
    const mock = renderHeimspieltage()
    await flushAsync()

    fireEvent.change(screen.getByPlaceholderText('Name des Ausrichters'), { target: { value: 'TSV Neuhausen' } })
    fireEvent.click(screen.getByRole('button', { name: 'Hinzufügen' }))
    await flushAsync()

    const post = mock.history.post.find(r => r.url === '/ausrichter')
    expect(post).toBeTruthy()
    expect(JSON.parse(post!.data)).toEqual({ name: 'TSV Neuhausen' })
  })

  test('Ausrichter umbenennen sendet PUT /ausrichter/{id} mit dem neuen Namen', async () => {
    const mock = renderHeimspieltage()
    await flushAsync()

    const row = ausrichterRow('SV Fellbach')
    fireEvent.click(within(row).getByRole('button', { name: 'Umbenennen' }))
    await flushAsync()

    // Die Mobile-Card teilt sich den `editId`-State und zeigt ebenfalls ein
    // Eingabefeld — beide sind in jsdom gleichzeitig im DOM, deshalb auf die
    // (weiterhin existierende) Desktop-Zeile scopen.
    const input = within(row).getByRole('textbox')
    fireEvent.change(input, { target: { value: 'SV Fellbach e.V.' } })
    fireEvent.click(within(row).getByRole('button', { name: 'Speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/ausrichter/2')
    expect(put).toBeTruthy()
    expect(JSON.parse(put!.data)).toEqual({ name: 'SV Fellbach e.V.' })
  })

  test('Default-Wechsel sendet is_default: true für den neu gewählten Eintrag', async () => {
    const mock = renderHeimspieltage()
    await flushAsync()

    const row = ausrichterRow('SV Fellbach')
    fireEvent.click(within(row).getByRole('button', { name: 'Als Default festlegen' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/ausrichter/2')
    expect(put).toBeTruthy()
    expect(JSON.parse(put!.data)).toEqual({ is_default: true })
  })

  test('409 beim Abwählen des Defaults zeigt eine verständliche Meldung', async () => {
    renderHeimspieltage(undefined, mock => {
      mock.onPut('/ausrichter/1').reply(409, { error: 'default_required' })
    })
    await flushAsync()

    const row = ausrichterRow('TV Ötlingen')
    fireEvent.click(within(row).getByRole('button', { name: /Ist Default-Ausrichter/ }))
    await flushAsync()

    expect(screen.getByText('Erst einen anderen Eintrag zum Default machen.')).toBeTruthy()
  })

  test('409 beim Deaktivieren des Defaults zeigt dieselbe Meldung', async () => {
    renderHeimspieltage(undefined, mock => {
      mock.onPut('/ausrichter/1').reply(409, { error: 'default_required' })
    })
    await flushAsync()

    const row = ausrichterRow('TV Ötlingen')
    fireEvent.click(within(row).getByRole('button', { name: 'Deaktivieren' }))
    await flushAsync()

    expect(screen.getByText('Erst einen anderen Eintrag zum Default machen.')).toBeTruthy()
  })

  test('Löschen ruft vorher /usage ab und zeigt die betroffenen Vorlagen-Zeilen im Dialog', async () => {
    const mock = renderHeimspieltage(undefined, mock => {
      mock.onGet('/ausrichter/2/usage').reply(200, {
        game_days: [{ date: '2026-09-14T00:00:00Z', season_id: 3, season_name: 'Saison 25/26' }],
        template_items: [{ id: 10, template_id: 5, template_name: 'Heim-Standard', duty_type_name: 'Kuchendienst' }],
      })
    })
    await flushAsync()

    const row = ausrichterRow('SV Fellbach')
    fireEvent.click(within(row).getByRole('button', { name: 'SV Fellbach löschen' }))
    await flushAsync()

    const usageGet = mock.history.get.find(r => r.url === '/ausrichter/2/usage')
    expect(usageGet).toBeTruthy()

    expect(screen.getByText(/werden mitgelöscht/)).toBeTruthy()
    // Exakter Text der Vorlagen-Zeile (Template-Name + Duty-Type) — als
    // Substring würde "Kuchendienst" auch in "Bewirtungs-/Kuchendienste" aus
    // dem Tab-Hinweis matchen.
    expect(screen.getByText('Heim-Standard – Kuchendienst')).toBeTruthy()
    expect(screen.getByText(/fallen auf den Default-Ausrichter zurück/)).toBeTruthy()
    expect(screen.getByText(/2026-09-14/)).toBeTruthy()
  })

  test('Löschen bestätigt: DELETE-Request geht raus, Dialog schließt, Liste lädt neu', async () => {
    const mock = renderHeimspieltage(undefined, mock => {
      mock.onGet('/ausrichter/2/usage').reply(200, { game_days: [], template_items: [] })
      mock.onDelete('/ausrichter/2').reply(200, { deleted: true })
    })
    await flushAsync()

    const row = ausrichterRow('SV Fellbach')
    fireEvent.click(within(row).getByRole('button', { name: 'SV Fellbach löschen' }))
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Endgültig löschen' }))
    await waitFor(() => {
      expect(mock.history.delete.find(r => r.url === '/ausrichter/2')).toBeTruthy()
    })
    await flushAsync()

    expect(screen.queryByRole('heading', { name: 'Ausrichter löschen?' })).toBeNull()
  })

  test('der Default-Eintrag lässt sich in der Tabelle nicht löschen (Button deaktiviert)', async () => {
    renderHeimspieltage()
    await flushAsync()

    const row = ausrichterRow('TV Ötlingen')
    const deleteButton = within(row).getByRole('button', { name: 'TV Ötlingen löschen' })
    expect(deleteButton).toBeDisabled()
  })
})
