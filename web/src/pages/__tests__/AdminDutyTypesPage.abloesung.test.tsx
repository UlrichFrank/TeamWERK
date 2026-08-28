/**
 * Ablösungs-Häkchen im Diensttyp-Editor (openspec/changes/dienst-abloesung Aufgabe 5.2).
 *
 * Das Kennzeichen ist bewusst kein dritter Zeit-Modus, sondern eine Kappung des im Modus
 * „Startzeit + Endzeit" definierten Endes — deshalb steht es unter den End-Feldern und
 * darf im Modus „Startzeit + Dauer" gar nicht erscheinen. Geprüft wird: (1) die
 * Sichtbarkeit hängt am Modus; (2) ein Moduswechsel hin und zurück verliert den Wert
 * nicht (er bleibt gespeichert, damit die Definition beim Zurückschalten wieder da ist);
 * (3) Anlegen und Bearbeiten schicken das Feld mit.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import AdminDutyTypesPage from '../AdminDutyTypesPage'

const mockGet = vi.fn()
const mockPost = vi.fn()
const mockPut = vi.fn()
vi.mock('../../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    put: (...args: unknown[]) => mockPut(...args),
    delete: vi.fn(),
  },
  getReference: (url: string) => mockGet(url).then((r: { data: unknown }) => r.data),
}))
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const KUCHEN = {
  id: 3,
  name: 'Kuchenverkauf',
  hours_value: 2,
  default_anchor: 'start' as const,
  default_offset_minutes: -30,
  duration_mode: 'dynamisch' as const,
  end_anchor: 'end' as const,
  end_offset_minutes: 30,
  end_at_next_duty: true,
  audiences: [],
}

function mockApi(types: unknown[] = []) {
  mockGet.mockImplementation((url: string) => {
    if (url === '/duty-types') return Promise.resolve({ data: types })
    return Promise.resolve({ data: [] })
  })
  mockPost.mockResolvedValue({ data: {} })
  mockPut.mockResolvedValue({ data: {} })
}

const CHECKBOX_NAME = /Endet spätestens bei Ablösung/

describe('AdminDutyTypesPage — Ablösung', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
  })

  test('Häkchen erscheint nur im Modus „Startzeit + Endzeit"', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    await screen.findByPlaceholderText('z.B. Kassierer')

    // Modus „Startzeit + Dauer": es gibt kein Ende, das gedeckelt werden könnte.
    expect(screen.queryByRole('checkbox', { name: CHECKBOX_NAME })).toBeNull()

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    expect(screen.getByRole('checkbox', { name: CHECKBOX_NAME })).toBeTruthy()

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Dauer' }))
    expect(screen.queryByRole('checkbox', { name: CHECKBOX_NAME })).toBeNull()
  })

  test('Moduswechsel hin und zurück verliert den Haken nicht', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    await screen.findByPlaceholderText('z.B. Kassierer')

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    const box = screen.getByRole('checkbox', { name: CHECKBOX_NAME }) as HTMLInputElement
    fireEvent.click(box)
    expect((screen.getByRole('checkbox', { name: CHECKBOX_NAME }) as HTMLInputElement).checked).toBe(true)

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Dauer' }))
    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    expect((screen.getByRole('checkbox', { name: CHECKBOX_NAME }) as HTMLInputElement).checked).toBe(true)
  })

  test('Anlegen schickt end_at_next_duty mit', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    fireEvent.change(await screen.findByPlaceholderText('z.B. Kassierer'), { target: { value: 'Kuchenverkauf' } })

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    fireEvent.click(screen.getByRole('checkbox', { name: CHECKBOX_NAME }))
    fireEvent.click(screen.getByText('Anlegen'))

    await waitFor(() => expect(mockPost).toHaveBeenCalled())
    expect(mockPost.mock.calls[0][1]).toMatchObject({
      name: 'Kuchenverkauf',
      duration_mode: 'dynamisch',
      end_at_next_duty: true,
    })
  })

  test('Bearbeiten lädt den Bestandswert und schickt die Änderung mit', async () => {
    mockApi([KUCHEN])
    render(<AdminDutyTypesPage />)
    await screen.findByText('Kuchenverkauf')
    fireEvent.click(screen.getByLabelText('Aktionen'))
    fireEvent.click(await screen.findByText('Bearbeiten'))
    await screen.findByDisplayValue('Kuchenverkauf')

    // Der gespeicherte Wert füllt das Häkchen vor.
    const box = await screen.findByRole('checkbox', { name: CHECKBOX_NAME })
    expect((box as HTMLInputElement).checked).toBe(true)

    fireEvent.click(box)
    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    expect(mockPut.mock.calls[0][1]).toMatchObject({ end_at_next_duty: false })
  })
})
