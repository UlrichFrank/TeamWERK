import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import DutyInstructionPage from './DutyInstructionPage'

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args) } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/dienste/anleitung/:typeId" element={<DutyInstructionPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('DutyInstructionPage', () => {
  beforeEach(() => { mockGet.mockReset() })

  test('renders instruction markdown', async () => {
    mockGet.mockResolvedValue({
      data: [
        { id: 42, name: 'Kasse', instruction_md: '## Ablauf\nKasse öffnen', instruction_updated_at: '2026-06-14T12:00:00Z' },
      ],
    })
    renderAt('/dienste/anleitung/42')
    await waitFor(() => expect(screen.getByText('Ablauf')).toBeTruthy())
    expect(screen.getByText('Kasse öffnen')).toBeTruthy()
    expect(screen.getByText(/Anleitung: Kasse/)).toBeTruthy()
  })

  test('shows placeholder when instruction empty', async () => {
    mockGet.mockResolvedValue({
      data: [{ id: 7, name: 'Aufbau', instruction_md: '' }],
    })
    renderAt('/dienste/anleitung/7')
    await waitFor(() => expect(screen.getByText(/noch keine Anleitung/)).toBeTruthy())
  })

  test('shows not-found when id missing', async () => {
    mockGet.mockResolvedValue({ data: [{ id: 1, name: 'X' }] })
    renderAt('/dienste/anleitung/999')
    await waitFor(() => expect(screen.getByText(/nicht gefunden/)).toBeTruthy())
  })
})
