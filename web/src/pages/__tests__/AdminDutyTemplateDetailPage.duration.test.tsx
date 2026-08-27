/**
 * Dauer je Vorlagen-Zeile (openspec/changes/dienst-dauer).
 *
 * Quelle: openspec/changes/dienst-dauer/specs/duties/spec.md — Requirement
 * "Vorlagen-Zeile trägt eine eigene Dauer (Copy-on-pick)".
 */
import { describe, test, expect, vi } from 'vitest'
import { Routes, Route } from 'react-router-dom'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import AdminDutyTemplateDetailPage from '../AdminDutyTemplateDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const DUTY_TYPES = [
  { id: 3, name: 'Kamera', hours_value: 1.5, default_anchor: 'start', default_offset_minutes: -60, audiences: [] },
]

function renderEditor() {
  renderAsPersona(
    <Routes>
      <Route path="/dienstplan-vorlagen/:id" element={<AdminDutyTemplateDetailPage />} />
    </Routes>,
    'vorstand',
    { route: '/dienstplan-vorlagen/4' },
  )
  const mock = getApiMock()
  mock.reset()
  mock.onGet('/duty-templates/4').reply(200, {
    id: 4,
    name: 'Heimspiel Standard',
    template_type: 'heim',
    duration_minutes: 75,
    items: [{ duty_type_id: 0, anchor: 'start', offset_minutes: 0, hours_value: 1, slots_count: 1, audiences: [] }],
  })
  mock.onGet('/duty-types').reply(200, DUTY_TYPES)
  mock.onGet('/teams/names').reply(200, [])
  mock.onPut('/duty-templates/4').reply(204)
  mock.onAny().reply(200, [])
  return mock
}

/** Beide Layouts (Mobile + Desktop) hängen am selben Item-State — [0] genügt. */
function hoursInput(): HTMLInputElement {
  return screen.getAllByPlaceholderText('z.B. 1h 30min')[0] as HTMLInputElement
}

describe('AdminDutyTemplateDetailPage — Dauer je Zeile', () => {
  test('Diensttyp-Auswahl füllt die Dauer der Zeile aus dem Typ-Wert', async () => {
    renderEditor()
    await flushAsync()

    const select = screen.getAllByDisplayValue('Auswählen…')[0]
    fireEvent.change(select, { target: { value: '3' } })

    expect(hoursInput().value).toBe('1h 30min')
  })

  test('eine abweichend gesetzte Dauer überlebt die Auswahl eines anderen Feldes', async () => {
    const mock = renderEditor()
    await flushAsync()

    const select = screen.getAllByDisplayValue('Auswählen…')[0]
    fireEvent.change(select, { target: { value: '3' } })
    expect(hoursInput().value).toBe('1h 30min')

    // Dauer manuell abweichend setzen.
    fireEvent.change(hoursInput(), { target: { value: '2h' } })
    fireEvent.blur(hoursInput())
    expect(hoursInput().value).toBe('2h')

    // Ein anderes Feld ändern (Zielgruppe) löst einen erneuten Render der Zeile
    // aus, ohne dass die Diensttyp-Auswahl erneut greift.
    fireEvent.click(screen.getAllByText('Spieler')[0])
    expect(hoursInput().value).toBe('2h')

    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await waitFor(() => expect(mock.history.put.find(r => r.url === '/duty-templates/4')).toBeTruthy())
    const put = mock.history.put.find(r => r.url === '/duty-templates/4')
    expect(JSON.parse(put!.data).items[0].hours_value).toBe(2)
  })
})
