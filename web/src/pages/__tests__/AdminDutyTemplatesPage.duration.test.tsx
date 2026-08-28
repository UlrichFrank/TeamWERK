/**
 * Dauer je Vorlagen-Zeile (openspec/changes/dienst-dauer) — portiert vom
 * entfernten Detailseiten-Editor (`AdminDutyTemplateDetailPage`) auf den
 * Modal-Editor der Listenseite. `HoursInput` ist unverändert dieselbe Komponente.
 *
 * Quelle: openspec/changes/dienst-dauer/specs/duties/spec.md — Requirement
 * "Vorlagen-Zeile trägt eine eigene Dauer (Copy-on-pick)".
 *
 * Seit dienst-zeitmodus-strikt gibt es das Dauer-Feld nur noch im Modus
 * „Startzeit + Dauer" — die Copy-on-pick-Tests der Dauer arbeiten deshalb mit
 * einem absoluten Diensttyp (id 4), die Modus-Tests mit dem dynamischen (id 3).
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
          {
            id: 3, name: 'Kamera', default_anchor: 'start', default_offset_minutes: -60, hours_value: 1.5,
            duration_mode: 'dynamisch', end_anchor: 'start', end_offset_minutes: 20,
            audiences: [],
          },
          {
            id: 4, name: 'Kuchen', default_anchor: 'start', default_offset_minutes: -60, hours_value: 1.5,
            duration_mode: 'absolut', end_anchor: 'end', end_offset_minutes: 0,
            audiences: [],
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

function hoursInput(): HTMLInputElement {
  return screen.getByPlaceholderText('z.B. 1h 30min') as HTMLInputElement
}

describe('AdminDutyTemplatesPage — Dauer je Zeile im Modal', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockReset()
  })

  test('Diensttyp-Auswahl füllt die Dauer der Zeile aus dem Typ-Wert', async () => {
    mockApi()
    await openEditModal()

    const select = screen.getByDisplayValue('Auswählen…')
    fireEvent.change(select, { target: { value: '4' } })

    expect(hoursInput().value).toBe('1h 30min')
  })

  test('eine abweichend gesetzte Dauer überlebt die Auswahl eines anderen Feldes', async () => {
    mockApi()
    await openEditModal()

    const select = screen.getByDisplayValue('Auswählen…')
    fireEvent.change(select, { target: { value: '4' } })
    expect(hoursInput().value).toBe('1h 30min')

    // Dauer manuell abweichend setzen.
    fireEvent.change(hoursInput(), { target: { value: '2h' } })
    fireEvent.blur(hoursInput())
    expect(hoursInput().value).toBe('2h')

    // Ein anderes Feld ändern (Zielgruppe) löst einen erneuten Render der Zeile
    // aus, ohne dass die Diensttyp-Auswahl erneut greift.
    fireEvent.click(screen.getByText('Spieler'))
    expect(hoursInput().value).toBe('2h')

    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const body = mockPut.mock.calls[0][1]
    expect(body.items[0].hours_value).toBe(2)
  })

  // openspec/changes/dienst-dauer-dynamisch, Aufgabe 6.4: der Copy-on-pick-Handler
  // kopiert heute anchor/offset_minutes/hours_value/audiences — muss um die drei
  // neuen Felder erweitert sein, sonst bleibt eine neu ausgewählte Zeile 'absolut',
  // obwohl der Diensttyp 'dynamisch' ist.
  test('Diensttyp-Auswahl überträgt auch den Dauer-Modus', async () => {
    mockApi()
    await openEditModal()

    const select = screen.getByDisplayValue('Auswählen…')
    fireEvent.change(select, { target: { value: '3' } })

    // End-Felder erscheinen nur im dynamischen Modus — sichtbarer Beleg, dass
    // der Modus mitkopiert wurde.
    expect(screen.getByText('End-Anker')).toBeTruthy()

    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const body = mockPut.mock.calls[0][1]
    expect(body.items[0]).toMatchObject({ duration_mode: 'dynamisch', end_anchor: 'start', end_offset_minutes: 20 })
  })

  // dienst-zeitmodus-strikt, Aufgabe 5.3: im Modus „Startzeit + Endzeit" gibt es kein
  // Dauer-Feld mehr — es wäre seit dem Wegfall des Rückfalls ein Feld ohne Wirkung.
  test('kein Dauer-Feld im Modus „Startzeit + Endzeit"', async () => {
    mockApi()
    await openEditModal()

    fireEvent.change(screen.getByDisplayValue('Auswählen…'), { target: { value: '4' } })
    expect(screen.queryByPlaceholderText('z.B. 1h 30min')).toBeTruthy()

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    expect(screen.queryByPlaceholderText('z.B. 1h 30min')).toBeNull()
    expect(screen.getByText('End-Anker')).toBeTruthy()
  })

  // dienst-zeitmodus-strikt, Aufgabe 5.3: eine Spanne, die nie positiv werden kann,
  // darf nicht in den Request laufen — der Server verwürfe die ganze Vorlage mit 400.
  test('unmögliche Spanne blockiert das Speichern', async () => {
    mockApi()
    await openEditModal()

    // Diensttyp 3 ist dynamisch mit Start-Anker start/−60 und End-Anker start/+20.
    fireEvent.change(screen.getByDisplayValue('Auswählen…'), { target: { value: '3' } })

    // End-Versatz vor den Start-Versatz ziehen → gleicher Anker, nie positiv.
    const endOffset = screen.getByText('End-Versatz').closest('div')!.querySelector('input') as HTMLInputElement
    fireEvent.change(endOffset, { target: { value: '-90min' } })
    fireEvent.blur(endOffset)

    expect(screen.getByText(/End-Versatz hinter dem Start-Versatz/)).toBeTruthy()

    fireEvent.click(screen.getByText('Speichern'))
    await waitFor(() => expect(screen.getByText(/^Eintrag 1:/)).toBeTruthy())
    expect(mockPut).not.toHaveBeenCalled()
  })
})
