/**
 * Bewirtungsrotations-Cap eines Vorlagen-Eintrags (kuchendienst-rotation).
 *
 * Quelle: openspec/changes/kuchendienst-rotation/specs/bewirtungsrotation/spec.md
 * — Requirement "Max-Kuchen-pro-Team-Cap pro Vorlagen-Item".
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
 * rotationMaxPerTeam = bereits gespeicherter Wert des einen Vorlagen-Items.
 * `putReply` MUSS vor dem `onAny()`-Catch-all registriert werden — axios-mock-adapter
 * matched Handler in Registrierungsreihenfolge, ein später hinzugefügter `onPut`
 * würde vom bereits registrierten Catch-all maskiert (falscher Grün-Test).
 */
function renderEditor(rotationMaxPerTeam: number | null, putReply?: [number, unknown]) {
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
      audiences: [], team_ids: [], rotation_max_per_team: rotationMaxPerTeam,
    }],
  })
  mock.onGet('/duty-types').reply(200, DUTY_TYPES)
  mock.onGet('/teams/names').reply(200, [])
  if (putReply) mock.onPut('/duty-templates/5').reply(...putReply)
  mock.onAny().reply(200, [])
  return mock
}

function capInput(): HTMLInputElement {
  return screen.getAllByLabelText(/Max\. Kuchen pro Mannschaft/)[0] as HTMLInputElement
}

describe('AdminDutyTemplateDetailPage — Bewirtungsrotations-Cap', () => {
  test('rendert leer, wenn kein Cap gespeichert ist', async () => {
    renderEditor(null)
    await flushAsync()

    expect(capInput().value).toBe('')
  })

  test('rendert den gespeicherten Cap-Wert', async () => {
    renderEditor(2)
    await flushAsync()

    expect(capInput().value).toBe('2')
  })

  test('leer gelassen: kein rotation_max_per_team im Payload', async () => {
    const mock = renderEditor(null, [204, undefined])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/duty-templates/5')
    expect(put).toBeTruthy()
    const body = JSON.parse(put!.data)
    expect(body.items[0].rotation_max_per_team ?? null).toBeNull()
  })

  test('gesetzt: rotation_max_per_team steht im Payload', async () => {
    const mock = renderEditor(null, [204, undefined])
    await flushAsync()

    fireEvent.change(capInput(), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/duty-templates/5')
    const body = JSON.parse(put!.data)
    expect(body.items[0].rotation_max_per_team).toBe(3)
  })

  test('Server-Fehler rotation_requires_normal_behavior wird lesbar angezeigt', async () => {
    renderEditor(2, [400, { error: 'rotation_requires_normal_behavior' }])
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    expect(screen.getByText(/Rotations-Cap erfordert/)).toBeTruthy()
  })
})
