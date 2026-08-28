import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import AdminDutyTemplatesPage from './AdminDutyTemplatesPage'

const mockGet = vi.fn()
const mockPut = vi.fn()
vi.mock('../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    put: (...args: unknown[]) => mockPut(...args),
    post: vi.fn(),
    delete: vi.fn(),
  },
}))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const TEAMS = [
  { id: 13, age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 1 },
  { id: 15, age_class: 'B-Jugend', gender: 'm', team_number: 1, group_count: 1 },
]

/** Ein gespeicherter Eintrag mit bereits gesetzter Team-Einschränkung. */
function savedItem(teamIds: number[]) {
  return {
    duty_type_id: 8,
    anchor: 'start',
    offset_minutes: -60,
    slots_count: 1,
    audiences: [],
    team_ids: teamIds,
    rotation_enabled: true,
  }
}

function mockApi(templateType = 'heim', teamIds: number[] = [15]) {
  mockGet.mockImplementation((url: string) => {
    if (url === '/duty-templates') {
      return Promise.resolve({
        data: [{ id: 1, name: 'Heimspiel', template_type: templateType, duration_minutes: 90, item_count: 1 }],
      })
    }
    if (url === '/duty-templates/1') {
      return Promise.resolve({
        data: { id: 1, name: 'Heimspiel', template_type: templateType, duration_minutes: 90, items: [savedItem(teamIds)] },
      })
    }
    if (url === '/duty-types') {
      return Promise.resolve({
        data: [{ id: 8, name: 'Kasse', default_anchor: 'start', default_offset_minutes: -60, hours_value: 1, audiences: [] }],
      })
    }
    if (url === '/teams/names') return Promise.resolve({ data: TEAMS })
    return Promise.resolve({ data: [] })
  })
  mockPut.mockResolvedValue({ data: {} })
}

async function openEditModal() {
  render(<MemoryRouter><AdminDutyTemplatesPage /></MemoryRouter>)
  await waitFor(() => expect(screen.getByText('Heimspiel')).toBeTruthy())
  fireEvent.click(screen.getByLabelText(/Aktionen|Menü/i))
  fireEvent.click(await screen.findByText('Bearbeiten'))
  return waitFor(() => expect(screen.getByDisplayValue('Heimspiel')).toBeTruthy())
}

describe('AdminDutyTemplatesPage — Teamauswahl im Bearbeiten-Modal', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockReset()
  })

  test('bietet Kaderteams und Rotations-Schalter im Modal an', async () => {
    mockApi()
    await openEditModal()
    expect(screen.getByText(/Kaderteams/)).toBeTruthy()
    const rotation = screen.getByLabelText(/Bewirtungsrotation/) as HTMLInputElement
    expect(rotation.type).toBe('checkbox')
    expect(rotation.checked).toBe(true) // SAVED_ITEM.rotation_enabled
  })

  /**
   * Der Cap gehört seit bewirtung-cap-global in die Einstellungen. Der Modal-Editor
   * darf dafür kein eigenes Eingabefeld mehr anbieten — sonst gäbe es zwei Quellen
   * für denselben Wert.
   */
  test('bietet im Modal kein Cap-Eingabefeld mehr an, sondern verweist auf die Einstellungen', async () => {
    mockApi()
    await openEditModal()
    expect(screen.queryByLabelText(/Max\. Kuchen pro Mannschaft/)).toBeNull()
    expect(screen.getByText(/Einstellungen → Bewirtung/)).toBeTruthy()
  })

  test('zeigt die gespeicherte Team-Einschränkung als angehakt', async () => {
    mockApi()
    await openEditModal()
    const mB = screen.getByText('mB').closest('label') as HTMLElement
    const mA = screen.getByText('mA').closest('label') as HTMLElement
    expect(within(mB).getByRole('checkbox')).toBeChecked()
    expect(within(mA).getByRole('checkbox')).not.toBeChecked()
  })

  test('speichert eine geänderte Teamauswahl über die PUT-Route', async () => {
    mockApi()
    await openEditModal()
    const mA = screen.getByText('mA').closest('label') as HTMLElement
    fireEvent.click(within(mA).getByRole('checkbox'))
    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const [url, body] = mockPut.mock.calls[0]
    expect(url).toBe('/duty-templates/1')
    // Bestehende Auswahl bleibt erhalten, neue kommt dazu.
    expect(body.items[0].team_ids).toEqual([15, 13])
    // Der Rotations-Schalter darf beim Speichern nicht verloren gehen.
    expect(body.items[0].rotation_enabled).toBe(true)
  })

  test('blendet die Teamauswahl bei generischen Vorlagen aus', async () => {
    mockApi('generisch')
    await openEditModal()
    expect(screen.queryByText(/Kaderteams/)).toBeNull()
  })

  // Portiert vom entfernten Detailseiten-Editor (AdminDutyTemplateDetailPage.teamScope.test.tsx):
  // die umgekehrte Leer-Semantik zur Zielgruppe daneben war dort abgedeckt, hier bislang nicht.
  test('Hinweis nennt die umgekehrte Leer-Semantik gegenüber der Zielgruppe', async () => {
    mockApi()
    await openEditModal()
    expect(screen.getAllByText(/leer =/).some(el => /alle/.test(el.textContent ?? ''))).toBe(true)
    expect(screen.getAllByText(/leer = keine/).length).toBeGreaterThan(0)
  })

  test('Abhaken entfernt nur das eine Team aus team_ids', async () => {
    mockApi('heim', [15, 13])
    await openEditModal()
    const mB = screen.getByText('mB').closest('label') as HTMLElement
    expect(within(mB).getByRole('checkbox')).toBeChecked()
    fireEvent.click(within(mB).getByRole('checkbox'))
    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const body = mockPut.mock.calls[0][1]
    expect(body.items[0].team_ids).toEqual([13])
  })

  test('ein gespeichertes Team ohne aktuellen Kader-Eintrag überlebt das Speichern', async () => {
    // 99 steht in keiner /teams/names-Antwort (kein Kader in der aktiven Saison)
    // und hat deshalb keine Checkbox — darf beim Togglen anderer Teams aber nicht
    // aus dem Array fallen.
    mockApi('heim', [99])
    await openEditModal()
    expect(screen.queryByText('99')).toBeNull()
    const mA = screen.getByText('mA').closest('label') as HTMLElement
    fireEvent.click(within(mA).getByRole('checkbox'))
    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const body = mockPut.mock.calls[0][1]
    expect(body.items[0].team_ids).toEqual([99, 13])
  })
})
