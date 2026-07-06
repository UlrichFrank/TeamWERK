import { describe, test, expect, afterEach, vi } from 'vitest'
import { render, screen, cleanup, waitFor, fireEvent } from '@testing-library/react'
import WartungsmodusPage from './WartungsmodusPage'

const mockGet = vi.fn()
const mockPost = vi.fn()
vi.mock('../../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
  },
}))

afterEach(() => {
  cleanup()
  mockGet.mockReset()
  mockPost.mockReset()
})

describe('WartungsmodusPage', () => {
  test('rendert den Ist-Zustand aus der API', async () => {
    mockGet.mockResolvedValue({
      data: { enabled: true, updated_at: '2026-07-05T10:00:00Z', updated_by_name: 'admin@ts.de' },
    })
    render(<WartungsmodusPage />)
    await waitFor(() => expect(screen.getByText(/Zustand:/i)).toBeTruthy())
    expect(screen.getByText(/Ein/)).toBeTruthy()
    expect(screen.getByText(/admin@ts\.de/)).toBeTruthy()
  })

  test('Toggle-Klick sendet POST mit invertiertem Wert und lädt neu', async () => {
    mockGet
      .mockResolvedValueOnce({ data: { enabled: false } })
      .mockResolvedValueOnce({ data: { enabled: true } })
    mockPost.mockResolvedValue({ data: {} })

    render(<WartungsmodusPage />)
    await waitFor(() => expect(screen.getByRole('button')).toBeTruthy())

    const button = screen.getByRole('button')
    expect(button.textContent).toMatch(/einschalten/i)
    fireEvent.click(button)

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith('/admin/maintenance-mode', { enabled: true })
    })
    await waitFor(() => {
      expect(screen.getByText(/Ein/)).toBeTruthy()
    })
  })

  test('Fehler beim Laden wird angezeigt', async () => {
    mockGet.mockRejectedValue(new Error('boom'))
    render(<WartungsmodusPage />)
    await waitFor(() =>
      expect(screen.getByText(/Zustand konnte nicht geladen werden/i)).toBeTruthy(),
    )
  })
})
