/**
 * Dauer-Modus-Maske im Diensttyp-Editor (openspec/changes/dienst-dauer-dynamisch,
 * Aufgabe 5). Deckt zwei Dinge ab, die beim ersten Bauen leicht auseinanderlaufen:
 * (1) die End-Felder erscheinen nur im Modus „dynamisch", (2) sowohl Anlegen als
 * auch Bearbeiten schicken alle drei neuen Felder mit — `saveEdit` hatte sie beim
 * ersten Durchgang schlicht vergessen, obwohl `handleCreate` sie schon trug.
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

describe('AdminDutyTypesPage — Dauer-Modus', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
  })

  test('End-Felder erscheinen nur im dynamischen Modus', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    await screen.findByPlaceholderText('z.B. Kassierer')

    expect(screen.queryByText('End-Anker')).toBeNull()

    fireEvent.click(screen.getByRole('radio', { name: /Dynamisch/ }))
    expect(screen.getByText('End-Anker')).toBeTruthy()
    expect(screen.getByText('End-Versatz')).toBeTruthy()

    fireEvent.click(screen.getByRole('radio', { name: 'Absolut' }))
    expect(screen.queryByText('End-Anker')).toBeNull()
  })

  test('Anlegen schickt alle drei Felder', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    fireEvent.change(await screen.findByPlaceholderText('z.B. Kassierer'), { target: { value: 'Kamera' } })

    fireEvent.click(screen.getByRole('radio', { name: /Dynamisch/ }))
    const endAnchorSelect = screen.getByText('End-Anker').closest('div')!.querySelector('select') as HTMLSelectElement
    fireEvent.change(endAnchorSelect, { target: { value: 'start' } })
    const endOffsetInput = screen.getByText('End-Versatz').closest('div')!.querySelector('input') as HTMLInputElement
    fireEvent.change(endOffsetInput, { target: { value: '15' } })
    fireEvent.blur(endOffsetInput)

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
