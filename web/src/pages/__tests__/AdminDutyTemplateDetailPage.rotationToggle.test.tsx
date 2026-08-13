/**
 * Bewirtungsrotations-Schalter eines Vorlagen-Eintrags (kuchendienst-rotation,
 * bewirtung-cap-global).
 *
 * Quelle: openspec/changes/bewirtung-cap-global/specs/bewirtungsrotation/spec.md
 * — Requirement "Rotations-Schalter pro Vorlagen-Item",
 *   Requirement "Vorlagen-Editor zeigt den Rotations-Schalter statt eines Cap-Feldes".
 */
import { describe, test, expect, vi } from 'vitest'
import { Routes, Route } from 'react-router-dom'
import { screen, fireEvent } from '@testing-library/react'
import AdminDutyTemplateDetailPage from '../AdminDutyTemplateDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const DUTY_TYPES = [
  { id: 11, name: 'Kuchendienst', default_anchor: 'start', default_offset_minutes: -60, audiences: [] },
]

/**
 * rotationEnabled = bereits gespeicherter Schalter des einen Vorlagen-Items.
 * `putReply` MUSS vor dem `onAny()`-Catch-all registriert werden — axios-mock-adapter
 * matched Handler in Registrierungsreihenfolge, ein später hinzugefügter `onPut`
 * würde vom bereits registrierten Catch-all maskiert (falscher Grün-Test).
 */
function renderEditor(rotationEnabled: boolean, putReply?: [number, unknown]) {
  renderAsPersona(
    <Routes>
      <Route path="/dienstplan-vorlagen/:id" element={<AdminDutyTemplateDetailPage />} />
    </Routes>,
    'vorstand',
    { route: '/dienstplan-vorlagen/5' },
  )
  const mock = getApiMock()
  mock.reset()
  mock.onGet('/duty-templates/5').reply(200, {
    id: 5,
    name: 'Heimspiel Standard',
    template_type: 'heim',
    duration_minutes: 75,
    items: [{
      duty_type_id: 11, anchor: 'start', offset_minutes: -60, slots_count: 1,
      audiences: [], team_ids: [], rotation_enabled: rotationEnabled,
    }],
  })
  mock.onGet('/duty-types').reply(200, DUTY_TYPES)
  mock.onGet('/teams/names').reply(200, [])
  if (putReply) mock.onPut('/duty-templates/5').reply(...putReply)
  mock.onAny().reply(200, [])
  return mock
}

/** Die Seite rendert Mobile- und Desktop-Variante parallel — [0] genügt. */
function rotationCheckbox(): HTMLInputElement {
  return screen.getAllByLabelText(/Bewirtungsrotation/)[0] as HTMLInputElement
}

function personenFelder(): HTMLInputElement[] {
  return screen.getAllByLabelText('Personen') as HTMLInputElement[]
}

describe('AdminDutyTemplateDetailPage — Bewirtungsrotations-Schalter', () => {
  test('rendert ungesetzt, wenn die Rotation deaktiviert ist', async () => {
    renderEditor(false)
    await flushAsync()

    expect(rotationCheckbox().checked).toBe(false)
  })

  test('rendert gesetzt, wenn die Rotation aktiviert ist', async () => {
    renderEditor(true)
    await flushAsync()

    expect(rotationCheckbox().checked).toBe(true)
  })

  test('verweist für die Obergrenze auf die Einstellungen statt ein Zahlenfeld anzubieten', async () => {
    renderEditor(true)
    await flushAsync()

    expect(screen.getAllByText(/Einstellungen → Bewirtung/).length).toBeGreaterThan(0)
    // Der Cap ist keine Item-Eigenschaft mehr — hier darf kein Eingabefeld dafür stehen.
    expect(screen.queryByLabelText(/Max\. Kuchen pro Mannschaft/)).toBeNull()
  })

  // bewirtung-kuchen-statt-slots: die Personenzahl eines Rotations-Slots kommt aus der
  // Zuteilung des Spieltags, slots_count der Vorlage bleibt wirkungslos.
  test('Personen-Feld ist bei aktiver Rotation deaktiviert und begründet', async () => {
    renderEditor(true)
    await flushAsync()

    const felder = personenFelder()
    expect(felder.length).toBeGreaterThan(0)
    for (const f of felder) expect(f.disabled).toBe(true)
    expect(screen.getAllByText(/aus der Zuteilung des Spieltags/).length).toBeGreaterThan(0)
  })

  test('Personen-Feld bleibt ohne Rotation bedienbar', async () => {
    renderEditor(false)
    await flushAsync()

    for (const f of personenFelder()) expect(f.disabled).toBe(false)
    expect(screen.queryByText(/aus der Zuteilung des Spieltags/)).toBeNull()
  })

  test('Einschalten der Rotation deaktiviert das Personen-Feld sofort', async () => {
    renderEditor(false)
    await flushAsync()

    fireEvent.click(rotationCheckbox())

    for (const f of personenFelder()) expect(f.disabled).toBe(true)
  })

  test('aus gelassen: rotation_enabled=false im Payload', async () => {
    const mock = renderEditor(false, [204, undefined])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/duty-templates/5')
    expect(put).toBeTruthy()
    const body = JSON.parse(put!.data)
    expect(body.items[0].rotation_enabled ?? false).toBe(false)
  })

  test('eingeschaltet: rotation_enabled=true im Payload', async () => {
    const mock = renderEditor(false, [204, undefined])
    await flushAsync()

    fireEvent.click(rotationCheckbox())
    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/duty-templates/5')
    const body = JSON.parse(put!.data)
    expect(body.items[0].rotation_enabled).toBe(true)
  })

  test('Server-Fehler rotation_requires_normal_behavior wird lesbar angezeigt', async () => {
    renderEditor(true, [400, { error: 'rotation_requires_normal_behavior' }])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    expect(screen.getByText(/Bewirtungsrotation erfordert/)).toBeTruthy()
  })
})
