import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import AdminDutyTemplateDetailPage from './AdminDutyTemplateDetailPage'

const mockGet = vi.fn()
const mockPut = vi.fn()
vi.mock('../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    put: (...args: unknown[]) => mockPut(...args),
  },
}))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

// Prod-nahe Daten: Heim-Vorlage mit einem Eintrag, Kaderteams der aktiven Saison.
const TEAMS = [
  { id: 13, age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 2 },
  { id: 15, age_class: 'B-Jugend', gender: 'm', team_number: 1, group_count: 2 },
]

function mockApi(templateType: string) {
  mockGet.mockImplementation((url: string) => {
    if (url === '/duty-templates/1') {
      return Promise.resolve({
        data: {
          id: 1, name: 'Heimspiel', template_type: templateType, duration_minutes: 90,
          items: [{ duty_type_id: 8, anchor: 'start', offset_minutes: -60, slots_count: 1, audiences: [] }],
        },
      })
    }
    if (url === '/duty-types') {
      return Promise.resolve({
        data: [{ id: 8, name: 'Kasse', default_anchor: 'start', default_offset_minutes: -60, audiences: [] }],
      })
    }
    if (url === '/teams/names') return Promise.resolve({ data: TEAMS })
    return Promise.resolve({ data: [] })
  })
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/dienstplan-vorlagen/1']}>
      <Routes>
        <Route path="/dienstplan-vorlagen/:id" element={<AdminDutyTemplateDetailPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('AdminDutyTemplateDetailPage — Teamauswahl', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockReset()
  })

  test('zeigt die Kaderteam-Auswahl bei einer Heim-Vorlage', async () => {
    mockApi('heim')
    renderPage()
    await waitFor(() => expect(screen.getAllByText(/Kaderteams/).length).toBeGreaterThan(0))
    expect(screen.getAllByText('mA1').length).toBeGreaterThan(0)
  })

  test('blendet die Auswahl bei generischen Vorlagen aus', async () => {
    mockApi('generisch')
    renderPage()
    await waitFor(() => expect(screen.getByDisplayValue('Heimspiel')).toBeTruthy())
    expect(screen.queryByText(/Kaderteams/)).toBeNull()
  })
})
