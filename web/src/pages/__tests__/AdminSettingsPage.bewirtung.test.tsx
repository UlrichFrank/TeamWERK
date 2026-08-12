/**
 * Einstellungen-Tab „Bewirtung": zeigt/ändert die beiden vereinsweiten Werte —
 * Spiele-zu-Kuchen-Verhältnis und Obergrenze pro Mannschaft.
 *
 * Quelle: openspec/changes/bewirtung-cap-global/specs/bewirtungsrotation/spec.md
 * — Requirement "Vereinsweites Spiele-zu-Kuchen-Verhältnis",
 *   Requirement "Vereinsweiter Cap „Max. Kuchen pro Mannschaft"",
 *   Requirement "Einstellungen-UI Tab „Bewirtung"".
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import AdminSettingsPage from '../AdminSettingsPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

/**
 * `putReply` MUSS vor dem `onAny()`-Catch-all registriert werden — axios-mock-adapter
 * matched Handler in Registrierungsreihenfolge, ein später hinzugefügter `onPut`
 * würde vom bereits registrierten Catch-all maskiert (falscher Grün-Test).
 */
function renderBewirtung(putReply?: [number, unknown]) {
  renderAsPersona(<AdminSettingsPage />, 'vorstand', { route: '/einstellungen?tab=bewirtung' })
  const mock = getApiMock()
  mock.reset()
  mock.onGet('/club').reply(200, {})
  mock.onGet('/seasons').reply(200, [])
  mock.onGet('/fee-rates').reply(200, { items: [] })
  mock.onGet('/age-class-rules').reply(200, [])
  mock.onGet('/stammvereine').reply(200, { items: [] })
  mock.onGet('/settings/bewirtung').reply(200, { verhaeltnis: 1, max_per_team: 2 })
  if (putReply) mock.onPut('/settings/bewirtung').reply(...putReply)
  mock.onAny().reply(200, [])
  return mock
}

describe('AdminSettingsPage — Bewirtung-Tab', () => {
  test('lädt beide Werte und zeigt sie an', async () => {
    renderBewirtung()
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    expect((screen.getByLabelText('Kuchen je Spiel') as HTMLInputElement).value).toBe('1')
    expect((screen.getByLabelText('Max. Kuchen pro Mannschaft') as HTMLInputElement).value).toBe('2')
  })

  test('benennt den Wirkungsbereich (Dienst-Generierung bei Heimspielen)', async () => {
    renderBewirtung()
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    expect(screen.getByText(/Dienst-Generierung bei/)).toBeTruthy()
    expect(screen.getByText('Heimspielen')).toBeTruthy()
  })

  test('Speichern-Interaktion: PUT mit korrektem Body, Erfolgsanzeige', async () => {
    const mock = renderBewirtung([200, { verhaeltnis: 0.5, max_per_team: 3 }])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    // Deutsches Komma als Dezimaltrennzeichen.
    fireEvent.change(screen.getByLabelText('Kuchen je Spiel'), { target: { value: '0,5' } })
    fireEvent.change(screen.getByLabelText('Max. Kuchen pro Mannschaft'), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/settings/bewirtung')
    expect(put).toBeTruthy()
    // Beide Werte gehen gemeinsam raus — der Tab hat ein Formular, nicht zwei.
    expect(JSON.parse(put!.data)).toEqual({ verhaeltnis: 0.5, max_per_team: 3 })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Gespeichert/ })).toBeTruthy()
    })
  })

  test('zeigt Fehlermeldung bei 403 (kein Vorstand)', async () => {
    renderBewirtung([403, { error: 'forbidden' }])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))
    await flushAsync()

    expect(screen.getByText('Keine Berechtigung, die Bewirtungs-Einstellungen zu ändern.')).toBeTruthy()
  })

  test('zeigt Fehlermeldung bei 400 (ungültiger Wert vom Server abgelehnt)', async () => {
    renderBewirtung([400, { error: 'invalid_verhaeltnis' }])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))
    await flushAsync()

    expect(screen.getByText('Speichern fehlgeschlagen – bitte Eingabe prüfen.')).toBeTruthy()
  })

  test('lehnt eine nicht-numerische Eingabe bereits clientseitig ab (kein PUT)', async () => {
    const mock = renderBewirtung()
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    fireEvent.change(screen.getByLabelText('Kuchen je Spiel'), { target: { value: 'abc' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))
    await flushAsync()

    expect(mock.history.put.find(r => r.url === '/settings/bewirtung')).toBeUndefined()
    expect(screen.getByText(/Kuchen je Spiel: bitte eine Zahl größer 0 angeben/)).toBeTruthy()
  })

  /**
   * Geleerter Cap. Werte <= 0 fängt schon `min={1}` des number-Inputs ab (Browser
   * UND jsdom blocken den Submit dann), ein LEERES Feld kommt dagegen bis in
   * handleSubmit durch — es ist der real erreichbare Ungültig-Zustand, und ohne
   * eigene Prüfung ginge `max_per_team: NaN` an den Server.
   */
  test('lehnt einen leeren Cap clientseitig ab (kein PUT, auch nicht für das gültige Verhältnis)', async () => {
    const mock = renderBewirtung()
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    fireEvent.change(screen.getByLabelText('Max. Kuchen pro Mannschaft'), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))
    await flushAsync()

    expect(mock.history.put.find(r => r.url === '/settings/bewirtung')).toBeUndefined()
    expect(screen.getByText(/Max. Kuchen pro Mannschaft: bitte eine ganze Zahl größer 0 angeben/)).toBeTruthy()
  })
})
