/**
 * Ablösungs-Häkchen je Vorlagen-Zeile (openspec/changes/dienst-abloesung Aufgabe 6.3).
 *
 * Wie in der Diensttyp-Maske ist das Kennzeichen eine Kappung des im Modus
 * „Startzeit + Endzeit" definierten Endes, kein dritter Modus — es erscheint deshalb nur
 * dort. Zusätzlich zur Sichtbarkeit prüft dieser Test den Copy-on-pick: wählt der
 * Vorstand einen Diensttyp aus, muss dessen Kennzeichen in die Zeile wandern, sonst
 * bliebe eine frisch angelegte Vorlage stumm beim alten Verhalten.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import AdminDutyTemplatesPage from '../AdminDutyTemplatesPage'

const mockGet = vi.fn()
const mockPut = vi.fn()
vi.mock('../../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    put: (...args: unknown[]) => mockPut(...args),
    post: vi.fn(),
    delete: vi.fn(),
  },
}))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function mockApi() {
  mockGet.mockImplementation((url: string) => {
    if (url === '/duty-templates') {
      return Promise.resolve({
        data: [{ id: 1, name: 'Heimspiel Standard', template_type: 'heim', duration_minutes: 75, item_count: 1 }],
      })
    }
    if (url === '/duty-templates/1') {
      return Promise.resolve({
        data: {
          id: 1,
          name: 'Heimspiel Standard',
          template_type: 'heim',
          duration_minutes: 75,
          items: [{ duty_type_id: 0, anchor: 'start', offset_minutes: 0, hours_value: 1, slots_count: 1, audiences: [] }],
        },
      })
    }
    if (url === '/duty-types') {
      return Promise.resolve({
        data: [
          // Kuchenverkauf: dynamisch UND mit Ablösung.
          {
            id: 3, name: 'Kuchenverkauf', default_anchor: 'start', default_offset_minutes: -30, hours_value: 2,
            duration_mode: 'dynamisch', end_anchor: 'end', end_offset_minutes: 30,
            end_at_next_duty: true, audiences: [],
          },
          // Zeitnehmer: dynamisch, aber ohne Ablösung — er endet mit seinem Spiel.
          {
            id: 4, name: 'Zeitnehmer', default_anchor: 'start', default_offset_minutes: -15, hours_value: 1.5,
            duration_mode: 'dynamisch', end_anchor: 'end', end_offset_minutes: 15,
            end_at_next_duty: false, audiences: [],
          },
          // Aufbau: absoluter Modus — dort gibt es kein Ende zum Deckeln.
          {
            id: 5, name: 'Aufbau', default_anchor: 'start', default_offset_minutes: -60, hours_value: 1,
            duration_mode: 'absolut', end_anchor: 'end', end_offset_minutes: 0,
            end_at_next_duty: false, audiences: [],
          },
        ],
      })
    }
    if (url === '/teams/names') return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
  mockPut.mockResolvedValue({ data: {} })
}

async function openEditModal() {
  render(<MemoryRouter><AdminDutyTemplatesPage /></MemoryRouter>)
  await waitFor(() => expect(screen.getByText('Heimspiel Standard')).toBeTruthy())
  fireEvent.click(screen.getByLabelText(/Aktionen|Menü/i))
  fireEvent.click(await screen.findByText('Bearbeiten'))
  return waitFor(() => expect(screen.getByDisplayValue('Heimspiel Standard')).toBeTruthy())
}

const CHECKBOX_NAME = /Endet spätestens bei Ablösung/

describe('AdminDutyTemplatesPage — Ablösung je Zeile', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockReset()
  })

  test('Diensttyp-Auswahl überträgt das Ablösungs-Kennzeichen', async () => {
    mockApi()
    await openEditModal()

    fireEvent.change(screen.getByDisplayValue('Auswählen…'), { target: { value: '3' } })

    const box = screen.getByRole('checkbox', { name: CHECKBOX_NAME }) as HTMLInputElement
    expect(box.checked).toBe(true)

    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    expect(mockPut.mock.calls[0][1].items[0]).toMatchObject({
      duration_mode: 'dynamisch',
      end_at_next_duty: true,
    })
  })

  test('ein Diensttyp ohne Ablösung lässt das Häkchen leer', async () => {
    mockApi()
    await openEditModal()

    fireEvent.change(screen.getByDisplayValue('Auswählen…'), { target: { value: '4' } })

    expect((screen.getByRole('checkbox', { name: CHECKBOX_NAME }) as HTMLInputElement).checked).toBe(false)
  })

  test('im Modus „Startzeit + Dauer" gibt es kein Häkchen', async () => {
    mockApi()
    await openEditModal()

    fireEvent.change(screen.getByDisplayValue('Auswählen…'), { target: { value: '5' } })

    expect(screen.queryByRole('checkbox', { name: CHECKBOX_NAME })).toBeNull()
  })

  test('das Häkchen lässt sich je Zeile abweichend vom Diensttyp setzen', async () => {
    mockApi()
    await openEditModal()

    // Zeitnehmer bringt end_at_next_duty=false mit — die Zeile darf es überschreiben.
    fireEvent.change(screen.getByDisplayValue('Auswählen…'), { target: { value: '4' } })
    fireEvent.click(screen.getByRole('checkbox', { name: CHECKBOX_NAME }))

    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    expect(mockPut.mock.calls[0][1].items[0].end_at_next_duty).toBe(true)
  })
})
