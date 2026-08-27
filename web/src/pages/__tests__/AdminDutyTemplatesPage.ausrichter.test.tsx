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

const AUSRICHTER = [
  { id: 1, name: 'Ausrichter A', aktiv: true, is_default: true, sort_order: 1 },
  { id: 2, name: 'Ausrichter B', aktiv: true, is_default: false, sort_order: 2 },
  { id: 3, name: 'Inaktiver', aktiv: false, is_default: false, sort_order: 3 },
]

const SAVED_ITEM_WITH_AUSRICHTER = {
  duty_type_id: 8,
  anchor: 'start',
  offset_minutes: -60,
  hours_value: 1,
  slots_count: 1,
  audiences: [],
  ausrichter_id: 1,
}

const SAVED_ITEM_WITHOUT_AUSRICHTER = {
  duty_type_id: 8,
  anchor: 'start',
  offset_minutes: -60,
  hours_value: 1,
  slots_count: 1,
  audiences: [],
}

function mockApi(templateType = 'heim', itemWithAusrichter = true) {
  mockGet.mockImplementation((url: string) => {
    if (url === '/duty-templates') {
      return Promise.resolve({
        data: [{ id: 1, name: 'Heimspiel', template_type: templateType, duration_minutes: 90, item_count: 1 }],
      })
    }
    if (url === '/duty-templates/1') {
      return Promise.resolve({
        data: {
          id: 1,
          name: 'Heimspiel',
          template_type: templateType,
          duration_minutes: 90,
          items: [itemWithAusrichter ? SAVED_ITEM_WITH_AUSRICHTER : SAVED_ITEM_WITHOUT_AUSRICHTER],
        },
      })
    }
    if (url === '/duty-types') {
      return Promise.resolve({
        data: [{ id: 8, name: 'Kasse', hours_value: 1, default_anchor: 'start', default_offset_minutes: -60, audiences: [] }],
      })
    }
    if (url === '/teams/names') return Promise.resolve({ data: [] })
    if (url === '/ausrichter') {
      return Promise.resolve({
        data: { items: AUSRICHTER },
      })
    }
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

describe('AdminDutyTemplatesPage — Ausrichter-Auswahl im Bearbeiten-Modal', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockReset()
  })

  test('bietet Ausrichter-Auswahl bei Heim-Vorlagen an', async () => {
    mockApi('heim')
    await openEditModal()
    expect(screen.getByLabelText(/Ausrichter/)).toBeTruthy()
    expect(screen.getByText('Ausrichter A')).toBeTruthy()
    expect(screen.getByText('Ausrichter B')).toBeTruthy()
    // Inaktive Ausrichter werden nicht angeboten
    expect(screen.queryByText('Inaktiver')).toBeNull()
  })

  test('blendet die Ausrichter-Auswahl bei Auswärts-Vorlagen aus', async () => {
    mockApi('auswärts')
    await openEditModal()
    expect(screen.queryByLabelText(/Ausrichter/)).toBeNull()
  })

  test('blendet die Ausrichter-Auswahl bei generischen Vorlagen aus', async () => {
    mockApi('generisch')
    await openEditModal()
    expect(screen.queryByLabelText(/Ausrichter/)).toBeNull()
  })

  test('zeigt die gespeicherte Ausrichter-Auswahl an', async () => {
    mockApi('heim', true)
    await openEditModal()
    const select = screen.getByLabelText(/Ausrichter/) as HTMLSelectElement
    expect(select.value).toBe('1')
  })

  test('speichert eine geänderte Ausrichter-Auswahl über die PUT-Route', async () => {
    mockApi('heim', false)
    await openEditModal()
    const select = screen.getByLabelText(/Ausrichter/) as HTMLSelectElement
    fireEvent.change(select, { target: { value: '2' } })
    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const [url, body] = mockPut.mock.calls[0]
    expect(url).toBe('/duty-templates/1')
    expect(body.items[0].ausrichter_id).toBe(2)
  })

  test('speichert null/leer als Leer-Option', async () => {
    mockApi('heim', true)
    await openEditModal()
    const select = screen.getByLabelText(/Ausrichter/) as HTMLSelectElement
    fireEvent.change(select, { target: { value: '' } })
    fireEvent.click(screen.getByText('Speichern'))

    await waitFor(() => expect(mockPut).toHaveBeenCalled())
    const [url, body] = mockPut.mock.calls[0]
    expect(url).toBe('/duty-templates/1')
    expect(body.items[0].ausrichter_id).toBeNull()
  })

  test('bietet die Leer-Option mit verständlicher Beschriftung an', async () => {
    mockApi('heim')
    await openEditModal()
    const select = screen.getByLabelText(/Ausrichter/) as HTMLSelectElement
    const emptyOption = Array.from(select.options).find(opt => opt.value === '')
    expect(emptyOption?.textContent).toContain('Gilt immer')
    expect(emptyOption?.textContent).toContain('unabhängig vom Ausrichter')
  })
})
