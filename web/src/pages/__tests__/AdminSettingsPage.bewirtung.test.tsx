/**
 * Einstellungen-Tab „Bewirtung" (kuchendienst-rotation): zeigt/ändert das
 * vereinsweite Spiele-zu-Kuchen-Verhältnis.
 *
 * Quelle: openspec/changes/kuchendienst-rotation/specs/bewirtungsrotation/spec.md
 * — Requirement "Vereinsweites Spiele-zu-Kuchen-Verhältnis",
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
  mock.onGet('/settings/bewirtung').reply(200, { verhaeltnis: 1 })
  if (putReply) mock.onPut('/settings/bewirtung').reply(...putReply)
  mock.onAny().reply(200, [])
  return mock
}

describe('AdminSettingsPage — Bewirtung-Tab', () => {
  test('lädt den Wert und zeigt ihn an', async () => {
    renderBewirtung()
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    const input = screen.getByLabelText('Kuchen je Spiel') as HTMLInputElement
    expect(input.value).toBe('1')
  })

  test('Speichern-Interaktion: PUT mit korrektem Body, Erfolgsanzeige', async () => {
    const mock = renderBewirtung([200, { verhaeltnis: 0.5 }])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Bewirtung' }))
    await flushAsync()

    const input = screen.getByLabelText('Kuchen je Spiel') as HTMLInputElement
    // Deutsches Komma als Dezimaltrennzeichen.
    fireEvent.change(input, { target: { value: '0,5' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/settings/bewirtung')
    expect(put).toBeTruthy()
    expect(JSON.parse(put!.data)).toEqual({ verhaeltnis: 0.5 })

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

    expect(screen.getByText('Keine Berechtigung, das Verhältnis zu ändern.')).toBeTruthy()
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

    const input = screen.getByLabelText('Kuchen je Spiel') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'abc' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))
    await flushAsync()

    expect(mock.history.put.find(r => r.url === '/settings/bewirtung')).toBeUndefined()
    expect(screen.getByText(/Bitte eine Zahl größer 0 angeben/)).toBeTruthy()
  })
})
