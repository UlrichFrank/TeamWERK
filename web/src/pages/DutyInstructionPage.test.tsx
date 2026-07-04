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
  beforeEach(() => {
    mockGet.mockReset()
    window.history.replaceState(null, '', '/') // fresh history state per test
  })

  test('renders instruction markdown from the detail path', async () => {
    mockGet.mockResolvedValue({
      data: { id: 42, name: 'Kasse', instruction_md: '## Ablauf\nKasse öffnen', instruction_updated_at: '2026-06-14T12:00:00Z' },
    })
    renderAt('/dienste/anleitung/42')
    await waitFor(() => expect(screen.getByText('Ablauf')).toBeTruthy())
    // Detail-Route, nicht die Liste.
    expect(mockGet).toHaveBeenCalledWith('/duty-types/42/instruction')
    expect(screen.getByText('Kasse öffnen')).toBeTruthy()
    expect(screen.getByText(/Anleitung: Kasse/)).toBeTruthy()
  })

  test('shows placeholder when instruction empty', async () => {
    mockGet.mockResolvedValue({ data: { id: 7, name: 'Aufbau', instruction_md: '' } })
    renderAt('/dienste/anleitung/7')
    await waitFor(() => expect(screen.getByText(/noch keine Anleitung/)).toBeTruthy())
  })

  test('shows not-found when detail path 404s', async () => {
    mockGet.mockRejectedValue(new Error('not found'))
    renderAt('/dienste/anleitung/999')
    await waitFor(() => expect(screen.getByText(/nicht gefunden/)).toBeTruthy())
  })

  test('renders fallback back-link on cold-start (history.state.idx === 0)', async () => {
    // Default JSDOM state: no idx set → coldStart true → fallback visible.
    mockGet.mockResolvedValue({
      data: { id: 42, name: 'Kasse', instruction_md: '## Foo' },
    })
    renderAt('/dienste/anleitung/42')
    await waitFor(() => expect(screen.getByText('Foo')).toBeTruthy())
    const link = screen.getByRole('link', { name: /Zur Dienstbörse/ })
    expect(link.getAttribute('href')).toBe('/dienste')
  })

  test('hides fallback back-link when history has depth', async () => {
    // Simulate that the user navigated in — React Router sets history.state.idx > 0.
    window.history.replaceState({ idx: 1 }, '', '/dienste/anleitung/42')
    mockGet.mockResolvedValue({
      data: { id: 42, name: 'Kasse', instruction_md: '## Foo' },
    })
    renderAt('/dienste/anleitung/42')
    await waitFor(() => expect(screen.getByText('Foo')).toBeTruthy())
    expect(screen.queryByRole('link', { name: /Zur Dienstbörse/ })).toBeNull()
  })
})
