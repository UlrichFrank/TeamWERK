/**
 * Bewirtungsrotations-Schalter eines Vorlagen-Eintrags (kuchendienst-rotation,
 * bewirtung-cap-global) — portiert vom entfernten Detailseiten-Editor
 * (`AdminDutyTemplateDetailPage`) auf den Modal-Editor der Listenseite. Beide
 * Editoren teilen sich die Felder aus `components/DutyTemplateItemFields.tsx`.
 *
 * Quelle: openspec/changes/bewirtung-cap-global/specs/bewirtungsrotation/spec.md
 * — Requirement "Rotations-Schalter pro Vorlagen-Item",
 *   Requirement "Vorlagen-Editor zeigt den Rotations-Schalter statt eines Cap-Feldes".
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AxiosError } from 'axios'
import AdminDutyTemplatesPage from '../AdminDutyTemplatesPage'

function axiosErrorWith(status: number, data: unknown): AxiosError {
  const err = new AxiosError('request failed')
  err.response = { status, data, statusText: '', headers: {}, config: {} } as AxiosError['response']
  return err
}

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

/** rotationEnabled = bereits gespeicherter Schalter des einen Vorlagen-Items. */
function mockApi(rotationEnabled: boolean) {
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
          items: [{
            duty_type_id: 11, anchor: 'start', offset_minutes: -60, hours_value: 1, slots_count: 1,
            audiences: [], team_ids: [], rotation_enabled: rotationEnabled,
          }],
        },
      })
    }
    if (url === '/duty-types') {
      return Promise.resolve({
        data: [{ id: 11, name: 'Kuchendienst', default_anchor: 'start', default_offset_minutes: -60, hours_value: 1, audiences: [] }],
      })
    }
    if (url === '/teams/names') return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
}

async function openEditModal() {
  render(<MemoryRouter><AdminDutyTemplatesPage /></MemoryRouter>)
  await waitFor(() => expect(screen.getByText('Heimspiel Standard')).toBeTruthy())
  fireEvent.click(screen.getByLabelText(/Aktionen|Menü/i))
  fireEvent.click(await screen.findByText('Bearbeiten'))
  return waitFor(() => expect(screen.getByDisplayValue('Heimspiel Standard')).toBeTruthy())
}

function rotationCheckbox(): HTMLInputElement {
  return screen.getByLabelText(/Bewirtungsrotation/) as HTMLInputElement
}

function personenFeld(): HTMLInputElement {
  return screen.getByLabelText('Personen') as HTMLInputElement
}

describe('AdminDutyTemplatesPage — Bewirtungsrotations-Schalter im Modal', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockReset()
  })

  test('rendert ungesetzt, wenn die Rotation deaktiviert ist', async () => {
    mockApi(false)
    await openEditModal()

    expect(rotationCheckbox().checked).toBe(false)
  })

  test('rendert gesetzt, wenn die Rotation aktiviert ist', async () => {
    mockApi(true)
    await openEditModal()

    expect(rotationCheckbox().checked).toBe(true)
  })

  test('verweist für die Obergrenze auf die Einstellungen statt ein Zahlenfeld anzubieten', async () => {
    mockApi(true)
    await openEditModal()

    expect(screen.getAllByText(/Einstellungen → Bewirtung/).length).toBeGreaterThan(0)
    // Der Cap ist keine Item-Eigenschaft mehr — hier darf kein Eingabefeld dafür stehen.
    expect(screen.queryByLabelText(/Max\. Kuchen pro Mannschaft/)).toBeNull()
  })

  // bewirtung-kuchen-statt-slots: die Personenzahl eines Rotations-Slots kommt aus der
  // Zuteilung des Spieltags, slots_count der Vorlage bleibt wirkungslos.
  test('Personen-Feld ist bei aktiver Rotation deaktiviert und begründet', async () => {
    mockApi(true)
    await openEditModal()

    expect(personenFeld().disabled).toBe(true)
    expect(screen.getByText(/aus der Zuteilung des Spieltags/)).toBeTruthy()
  })

  test('Personen-Feld bleibt ohne Rotation bedienbar', async () => {
    mockApi(false)
    await openEditModal()

    expect(personenFeld().disabled).toBe(false)
    expect(screen.queryByText(/aus der Zuteilung des Spieltags/)).toBeNull()
  })

  test('Einschalten der Rotation deaktiviert das Personen-Feld sofort', async () => {
    mockApi(false)
    await openEditModal()

    fireEvent.click(rotationCheckbox())

    expect(personenFeld().disabled).toBe(true)
  })

  test('aus gelassen: rotation_enabled=false im Payload', async () => {
    mockApi(false)
    mockPut.mockResolvedValue({ data: {} })
    await openEditModal()

    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const [url, body] = mockPut.mock.calls[0]
    expect(url).toBe('/duty-templates/1')
    expect(body.items[0].rotation_enabled ?? false).toBe(false)
  })

  test('eingeschaltet: rotation_enabled=true im Payload', async () => {
    mockApi(false)
    mockPut.mockResolvedValue({ data: {} })
    await openEditModal()

    fireEvent.click(rotationCheckbox())
    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const body = mockPut.mock.calls[0][1]
    expect(body.items[0].rotation_enabled).toBe(true)
  })

  /**
   * Beim Entfernen der Detailseite wanderte deren Fehlercode-Übersetzung mit ins
   * Modal. Ohne sie hätte jeder abgelehnte Speichervorgang nur „Speichern
   * fehlgeschlagen" gezeigt, und der Vorstand müsste raten, welche Zeile das
   * Backend beanstandet hat.
   */
  test('Server-Fehlercode wird in einen lesbaren Hinweis übersetzt', async () => {
    mockApi(true)
    // Echter AxiosError, kein nacktes Objekt: errorData() prüft isAxiosError.
    mockPut.mockRejectedValue(axiosErrorWith(400, { error: 'rotation_requires_normal_behavior' }))
    await openEditModal()

    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(screen.getByText(/Bewirtungsrotation erfordert/)).toBeTruthy())
    expect(screen.queryByText('Speichern fehlgeschlagen.')).toBeNull()
  })

  test('unbekannter Fehlercode fällt auf den generischen Hinweis zurück', async () => {
    mockApi(true)
    mockPut.mockRejectedValue(axiosErrorWith(500, {}))
    await openEditModal()

    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(screen.getByText('Speichern fehlgeschlagen.')).toBeTruthy())
  })
})
