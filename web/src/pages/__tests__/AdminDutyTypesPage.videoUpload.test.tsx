/**
 * Video-Upload-Häkchen im Diensttyp-Editor (video-download-duty-upload).
 *
 * Anders als das Ablösungs-Häkchen ist dieses Kennzeichen unabhängig vom
 * Zeit-Modus immer sichtbar — es beschreibt eine Berechtigung, keine
 * Zeitberechnung. Geprüft wird: (1) der Bestandswert füllt das Häkchen vor;
 * (2) Anlegen und Bearbeiten schicken das Feld mit.
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

const VIDEO_DUTY = {
  id: 4,
  name: 'Video',
  hours_value: 2,
  default_anchor: 'start' as const,
  default_offset_minutes: -30,
  duration_mode: 'absolut' as const,
  end_anchor: 'end' as const,
  end_offset_minutes: 0,
  audiences: [],
  grants_video_upload: true,
}

function mockApi(types: unknown[] = []) {
  mockGet.mockImplementation((url: string) => {
    if (url === '/duty-types') return Promise.resolve({ data: types })
    return Promise.resolve({ data: [] })
  })
  mockPost.mockResolvedValue({ data: {} })
  mockPut.mockResolvedValue({ data: {} })
}

const CHECKBOX_NAME = /Berechtigt zum Video-Upload/

describe('AdminDutyTypesPage — Video-Upload-Berechtigung', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
  })

  test('Häkchen ist unabhängig vom Zeit-Modus sichtbar', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    await screen.findByPlaceholderText('z.B. Kassierer')

    // Default-Modus „Startzeit + Dauer"
    expect(screen.getByRole('checkbox', { name: CHECKBOX_NAME })).toBeTruthy()

    fireEvent.click(screen.getByRole('radio', { name: 'Startzeit + Endzeit' }))
    expect(screen.getByRole('checkbox', { name: CHECKBOX_NAME })).toBeTruthy()
  })

  test('Anlegen schickt grants_video_upload mit', async () => {
    mockApi([])
    render(<AdminDutyTypesPage />)
    fireEvent.click(await screen.findByText('+ Diensttyp'))
    fireEvent.change(await screen.findByPlaceholderText('z.B. Kassierer'), { target: { value: 'Video' } })

    fireEvent.click(screen.getByRole('checkbox', { name: CHECKBOX_NAME }))
    fireEvent.click(screen.getByText('Anlegen'))

    await waitFor(() => expect(mockPost).toHaveBeenCalled())
    expect(mockPost.mock.calls[0][1]).toMatchObject({
      name: 'Video',
      grants_video_upload: true,
    })
  })

  test('Bearbeiten lädt den Bestandswert und schickt die Änderung mit', async () => {
    mockApi([VIDEO_DUTY])
    render(<AdminDutyTypesPage />)
    await screen.findByText('Video')
    fireEvent.click(screen.getByLabelText('Aktionen'))
    fireEvent.click(await screen.findByText('Bearbeiten'))
    await screen.findByDisplayValue('Video')

    const box = await screen.findByRole('checkbox', { name: CHECKBOX_NAME })
    expect((box as HTMLInputElement).checked).toBe(true)

    fireEvent.click(box)
    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    expect(mockPut.mock.calls[0][1]).toMatchObject({ grants_video_upload: false })
  })
})
