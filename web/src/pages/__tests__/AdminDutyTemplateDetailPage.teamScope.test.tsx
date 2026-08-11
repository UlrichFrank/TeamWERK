/**
 * Team-Einschränkung eines Vorlagen-Eintrags im Dienstplan-Vorlagen-Editor.
 *
 * Quelle: openspec/changes/duty-template-team-scope/specs/duty-template-team-scope/spec.md
 * — Requirement "Vorlagen-Editor bietet nur Kaderteams der aktiven Saison zur Auswahl an".
 */
import { describe, test, expect, vi } from 'vitest'
import { Routes, Route } from 'react-router-dom'
import { screen, fireEvent, within } from '@testing-library/react'
import AdminDutyTemplateDetailPage from '../AdminDutyTemplateDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

// group_count = 2 → Kurzname trägt die Team-Nummer: id 7 → "mA1", id 8 → "mA2".
const TEAMS = [
  { id: 7, age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 2 },
  { id: 8, age_class: 'A-Jugend', gender: 'm', team_number: 2, group_count: 2 },
]

const DUTY_TYPES = [
  { id: 3, name: 'Kamera', default_anchor: 'start', default_offset_minutes: -60, audiences: [] },
]

/** teamIDs = bereits gespeicherte team_ids des einen Vorlagen-Items. */
function renderEditor(teamIDs: number[] | null) {
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
    items: [{
      duty_type_id: 3, anchor: 'start', offset_minutes: -60, slots_count: 1,
      audiences: [], ...(teamIDs ? { team_ids: teamIDs } : {}),
    }],
  })
  mock.onGet('/duty-types').reply(200, DUTY_TYPES)
  mock.onGet('/teams/names').reply(200, TEAMS)
  mock.onPut('/duty-templates/4').reply(204)
  mock.onAny().reply(200, [])
  return mock
}

/** Die Checkbox eines Kaderteams innerhalb der Kaderteams-Gruppe (Desktop-Layout). */
function teamCheckbox(name: string): HTMLInputElement {
  // Beide Layouts (Mobile + Desktop) sind im jsdom gerendert; der erste Treffer reicht,
  // beide hängen am selben Item-State.
  const label = screen.getAllByText(name)[0].closest('label') as HTMLLabelElement
  return within(label).getByRole('checkbox') as HTMLInputElement
}

describe('AdminDutyTemplateDetailPage — Kaderteams pro Eintrag', () => {
  test('bietet die Kaderteams der aktiven Saison mit Kurznamen an', async () => {
    renderEditor(null)
    await flushAsync()

    expect(screen.getAllByText('mA1').length).toBeGreaterThan(0)
    expect(screen.getAllByText('mA2').length).toBeGreaterThan(0)
    // Ohne gespeicherte Auswahl ist nichts angehakt (= gilt für alle Teams).
    expect(teamCheckbox('mA1').checked).toBe(false)
    expect(teamCheckbox('mA2').checked).toBe(false)
  })

  test('Hinweis nennt die umgekehrte Leer-Semantik gegenüber der Zielgruppe', async () => {
    renderEditor(null)
    await flushAsync()

    expect(screen.getAllByText(/leer =/).some(el => /alle/.test(el.textContent ?? ''))).toBe(true)
    expect(screen.getAllByText(/leer = keine/).length).toBeGreaterThan(0)
  })

  test('Anhaken schreibt team_ids in die gespeicherte Payload', async () => {
    const mock = renderEditor(null)
    await flushAsync()

    fireEvent.click(teamCheckbox('mA1'))
    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/duty-templates/4')
    expect(put).toBeTruthy()
    expect(JSON.parse(put!.data).items[0].team_ids).toEqual([7])
  })

  test('Abhaken entfernt nur das eine Team aus team_ids', async () => {
    const mock = renderEditor([7, 8])
    await flushAsync()

    expect(teamCheckbox('mA1').checked).toBe(true)
    fireEvent.click(teamCheckbox('mA1'))
    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/duty-templates/4')
    expect(JSON.parse(put!.data).items[0].team_ids).toEqual([8])
  })

  test('ein gespeichertes Team ohne aktuellen Kader-Eintrag überlebt das Speichern', async () => {
    // 99 steht in keiner /teams/names-Antwort (kein Kader in der aktiven Saison)
    // und hat deshalb keine Checkbox — darf beim Togglen anderer Teams aber nicht
    // aus dem Array fallen.
    const mock = renderEditor([99])
    await flushAsync()

    expect(screen.queryByText('99')).toBeNull()
    fireEvent.click(teamCheckbox('mA2'))
    fireEvent.click(screen.getByRole('button', { name: 'Vorlage speichern' }))
    await flushAsync()

    const put = mock.history.put.find(r => r.url === '/duty-templates/4')
    expect(JSON.parse(put!.data).items[0].team_ids).toEqual([99, 8])
  })
})

describe('AdminDutyTemplateDetailPage — generische Vorlagen', () => {
  test('bietet keine Kaderteams-Auswahl an (Regen ignoriert team_ids dort)', async () => {
    renderAsPersona(
      <Routes>
        <Route path="/dienstplan-vorlagen/:id" element={<AdminDutyTemplateDetailPage />} />
      </Routes>,
      'vorstand',
      { route: '/dienstplan-vorlagen/9' },
    )
    const mock = getApiMock()
    mock.reset()
    mock.onGet('/duty-templates/9').reply(200, {
      id: 9, name: 'Turnier', template_type: 'generisch', duration_minutes: 120,
      items: [{ duty_type_id: 3, anchor: 'start', offset_minutes: -60, slots_count: 1, audiences: [] }],
    })
    mock.onGet('/duty-types').reply(200, DUTY_TYPES)
    mock.onGet('/teams/names').reply(200, TEAMS)
    mock.onAny().reply(200, [])
    await flushAsync()

    expect(screen.queryByText(/Kaderteams/)).toBeNull()
    expect(screen.queryByText('mA1')).toBeNull()
    // Die Zielgruppen-Auswahl bleibt davon unberührt.
    expect(screen.getAllByText(/leer = keine/).length).toBeGreaterThan(0)
  })
})
