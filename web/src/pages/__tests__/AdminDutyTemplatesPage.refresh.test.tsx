/**
 * „Aus Diensttypen auffrischen" im Vorlagen-Editor.
 *
 * Hintergrund: eine Vorlagen-Zeile ist nach dem Auswählen des Diensttyps
 * eigenständig (Copy-on-pick), und der Dienst-Regen liest ausschließlich die
 * Zeile. Ändert der Vorstand später die Dauer eines Diensttyps, bleibt die
 * Vorlage stehen — „Dienste aktualisieren" auf dem Kalender überträgt dann
 * weiterhin den alten Wert. Diese Aktion ist der Weg zurück.
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

/** itemHours = Stand der Vorlagen-Zeile, typeHours = aktueller Stand des Diensttyps. */
function mockApi(itemHours: number, typeHours: number) {
  mockGet.mockImplementation((url: string) => {
    if (url === '/duty-templates') {
      return Promise.resolve({
        data: [{ id: 1, name: 'Heimspiel', template_type: 'heim', duration_minutes: 75, item_count: 1 }],
      })
    }
    if (url === '/duty-templates/1') {
      return Promise.resolve({
        data: {
          id: 1, name: 'Heimspiel', template_type: 'heim', duration_minutes: 75,
          items: [{ duty_type_id: 3, anchor: 'start', offset_minutes: -60, hours_value: itemHours, slots_count: 1, audiences: [] }],
        },
      })
    }
    if (url === '/duty-types') {
      return Promise.resolve({
        data: [{ id: 3, name: 'Zeitnehmer', default_anchor: 'start', default_offset_minutes: -60, hours_value: typeHours, audiences: [] }],
      })
    }
    if (url === '/teams/names') return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
  mockPut.mockResolvedValue({ data: {} })
}

async function openEditModal() {
  render(<MemoryRouter><AdminDutyTemplatesPage /></MemoryRouter>)
  await waitFor(() => expect(screen.getByText('Heimspiel')).toBeTruthy())
  fireEvent.click(screen.getByLabelText(/Aktionen|Menü/i))
  fireEvent.click(await screen.findByText('Bearbeiten'))
  await waitFor(() => expect(screen.getByDisplayValue('Heimspiel')).toBeTruthy())
}

function openItemMenu() {
  fireEvent.click(screen.getByLabelText('Weitere Aktionen'))
}

function hoursInput(): HTMLInputElement {
  return screen.getByPlaceholderText('z.B. 1h 30min') as HTMLInputElement
}

describe('AdminDutyTemplatesPage — aus Diensttypen auffrischen', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockReset()
  })

  test('holt die geänderte Dauer des Diensttyps in die Zeile', async () => {
    mockApi(1, 1.5)
    await openEditModal()
    expect(hoursInput().value).toBe('1h')

    openItemMenu()
    fireEvent.click(screen.getByText('Aus Diensttypen auffrischen'))

    await waitFor(() => expect(hoursInput().value).toBe('1h 30min'))
    expect(screen.getByText(/1 Eintrag aus den Diensttypen aufgefrischt/)).toBeTruthy()
  })

  test('meldet, wenn schon alles übereinstimmt, und ändert nichts', async () => {
    mockApi(1.5, 1.5)
    await openEditModal()

    openItemMenu()
    fireEvent.click(screen.getByText('Aus Diensttypen auffrischen'))

    await waitFor(() => expect(screen.getByText(/stimmen bereits mit ihrem Diensttyp überein/)).toBeTruthy())
    expect(hoursInput().value).toBe('1h 30min')
  })

  test('persistiert nichts von allein — erst Speichern schickt die neuen Werte', async () => {
    mockApi(1, 1.5)
    await openEditModal()

    openItemMenu()
    fireEvent.click(screen.getByText('Aus Diensttypen auffrischen'))
    await waitFor(() => expect(hoursInput().value).toBe('1h 30min'))
    expect(mockPut).not.toHaveBeenCalled()

    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    expect(mockPut.mock.calls[0][1].items[0].hours_value).toBe(1.5)
  })
})
