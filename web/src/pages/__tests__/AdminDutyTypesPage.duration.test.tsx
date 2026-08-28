/**
 * Zeit-Modus-Maske im Diensttyp-Editor (openspec/changes/dienst-dauer-dynamisch
 * Aufgabe 5, erweitert um dienst-zeitmodus-strikt Aufgabe 4.4). Deckt ab:
 * (1) die End-Felder erscheinen nur im Modus „Startzeit + Endzeit" — und das
 * Dauer-Feld verschwindet dort, weil es dort nichts mehr bewirkt (der Rückfall
 * auf `hours_value` ist entfallen); (2) sowohl Anlegen als auch Bearbeiten
 * schicken alle drei Felder mit — `saveEdit` hatte sie beim ersten Durchgang
 * schlicht vergessen, obwohl `handleCreate` sie schon trug; (3) eine Spanne, die
 * nie positiv werden kann, blockiert das Speichern vor dem Request.
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

const EXISTING_TYPE = {
  id: 7,
  name: 'Zeitnehmer',
  hours_value: 1,
  default_anchor: 'start' as const,
  default_offset_minutes: -30,
  duration_mode: 'dynamisch' as const,
  end_anchor: 'start' as const,
  end_offset_minutes: 20,
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

/** OffsetInput übernimmt seinen Wert erst beim Blur — beide Schritte in einem Helfer. */
function setOffset(label: HTMLElement, value: string) {
  const input = label.closest('div')!.querySelector('input') as HTMLInputElement
  fireEvent.change(input, { target: { value } })
  fireEvent.blur(input)
}

describe('AdminDutyTypesPage — Zeit-Modus', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
  })

  test('End-Felder ersetzen im Modus „Startzeit + Endzeit" das Dauer-Feld', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    await screen.findByPlaceholderText('z.B. Kassierer')

    // Modus 1: Dauer, keine End-Felder.
    expect(screen.getByText('Dauer')).toBeTruthy()
    expect(screen.queryByText('End-Anker')).toBeNull()

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    expect(screen.getByText('End-Anker')).toBeTruthy()
    expect(screen.getByText('End-Versatz')).toBeTruthy()
    // Kein Rückfall mehr → kein Dauer-Feld, das die Endzeit stillschweigend ersetzt.
    expect(screen.queryByText('Dauer')).toBeNull()

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Dauer' }))
    expect(screen.queryByText('End-Anker')).toBeNull()
    expect(screen.getByText('Dauer')).toBeTruthy()
  })

  test('Start-Felder heißen Start-Anker und Start-Versatz', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    await screen.findByPlaceholderText('z.B. Kassierer')

    expect(screen.getByText('Start-Anker')).toBeTruthy()
    expect(screen.getByText('Start-Versatz')).toBeTruthy()
    expect(screen.queryByText('Standard-Anker')).toBeNull()
  })

  test('unmögliche Spanne blockiert das Speichern', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    fireEvent.change(await screen.findByPlaceholderText('z.B. Kassierer'), { target: { value: 'Halbzeit' } })

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    // Start und Ende am selben Anker, Ende NICHT dahinter → kann nie positiv werden.
    // Bei verschiedenen Ankern (Default) hinge die Dauer an der Spieldauer und wäre
    // hier gar nicht entscheidbar — deshalb muss der End-Anker mitgezogen werden.
    const endAnchor = screen.getByText('End-Anker').closest('div')!.querySelector('select') as HTMLSelectElement
    fireEvent.change(endAnchor, { target: { value: 'start' } })
    setOffset(screen.getByText('Start-Versatz'), '40')
    setOffset(screen.getByText('End-Versatz'), '25')

    expect(screen.getByText(/End-Versatz hinter dem Start-Versatz/)).toBeTruthy()
    const submit = screen.getByText('Anlegen') as HTMLButtonElement
    expect(submit.disabled).toBe(true)
    fireEvent.click(submit)
    expect(mockPost).not.toHaveBeenCalled()

    // Ende hinter den Start geschoben → wieder speicherbar.
    setOffset(screen.getByText('End-Versatz'), '55')
    expect(screen.queryByText(/End-Versatz hinter dem Start-Versatz/)).toBeNull()
    expect((screen.getByText('Anlegen') as HTMLButtonElement).disabled).toBe(false)
  })

  test('Anlegen schickt alle drei Felder', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    fireEvent.change(await screen.findByPlaceholderText('z.B. Kassierer'), { target: { value: 'Kamera' } })

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    const endAnchorSelect = screen.getByText('End-Anker').closest('div')!.querySelector('select') as HTMLSelectElement
    fireEvent.change(endAnchorSelect, { target: { value: 'start' } })
    setOffset(screen.getByText('End-Versatz'), '15')

    fireEvent.click(screen.getByText('Anlegen'))

    await waitFor(() => expect(mockPost).toHaveBeenCalled())
    const body = mockPost.mock.calls[0][1]
    expect(body.duration_mode).toBe('dynamisch')
    expect(body.end_anchor).toBe('start')
    expect(body.end_offset_minutes).toBe(15)
  })

  test('Bearbeiten schickt alle drei Felder mit (Regression: saveEdit vergaß sie)', async () => {
    mockApi([EXISTING_TYPE])
    render(<AdminDutyTypesPage />)
    await screen.findByText('Zeitnehmer')

    fireEvent.click(screen.getByLabelText('Aktionen'))
    fireEvent.click(await screen.findByText('Bearbeiten'))
    await screen.findByDisplayValue('Zeitnehmer')

    // Die Maske zeigt bereits den bestehenden dynamischen Zustand.
    expect(screen.getByText('End-Anker')).toBeTruthy()

    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const body = mockPut.mock.calls[0][1]
    expect(body.duration_mode).toBe('dynamisch')
    expect(body.end_anchor).toBe('start')
    expect(body.end_offset_minutes).toBe(20)
  })
})
