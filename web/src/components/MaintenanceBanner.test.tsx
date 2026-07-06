import { describe, test, expect, afterEach, vi } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import MaintenanceBanner from './MaintenanceBanner'

// Wir mocken den Hook, damit der Test unabhängig vom Fetch- und SSE-Verhalten
// den Enabled-Zustand direkt steuert.
const mockUseMaintenanceStatus = vi.fn()
vi.mock('../hooks/useMaintenanceStatus', () => ({
  useMaintenanceStatus: () => mockUseMaintenanceStatus(),
}))

afterEach(() => {
  cleanup()
  mockUseMaintenanceStatus.mockReset()
})

describe('MaintenanceBanner', () => {
  test('rendert Banner mit Text und Warn-Icon, wenn enabled=true', () => {
    mockUseMaintenanceStatus.mockReturnValue({ enabled: true, loading: false })
    render(<MaintenanceBanner />)
    expect(screen.getByRole('status')).toBeTruthy()
    expect(screen.getByText(/Wartungsmodus aktiv/i)).toBeTruthy()
  })

  test('rendert null, wenn enabled=false', () => {
    mockUseMaintenanceStatus.mockReturnValue({ enabled: false, loading: false })
    const { container } = render(<MaintenanceBanner />)
    expect(container.firstChild).toBeNull()
  })

  test('rendert null, während loading (initialer Fetch) und enabled=false', () => {
    mockUseMaintenanceStatus.mockReturnValue({ enabled: false, loading: true })
    const { container } = render(<MaintenanceBanner />)
    expect(container.firstChild).toBeNull()
  })
})
