import { describe, test, expect, afterEach, vi } from 'vitest'
import { renderHook, waitFor, cleanup } from '@testing-library/react'
import { useMaintenanceStatus } from './useMaintenanceStatus'

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({
  api: { get: (...args: unknown[]) => mockGet(...args) },
}))

// useLiveUpdates: wir verwahren den Handler-Ref lokal, damit Tests SSE-Events
// direkt auslösen können.
let liveHandler: ((event: string) => void) | null = null
vi.mock('./useLiveUpdates', () => ({
  useLiveUpdates: (h: (event: string) => void) => {
    liveHandler = h
  },
}))

afterEach(() => {
  cleanup()
  mockGet.mockReset()
  liveHandler = null
})

describe('useMaintenanceStatus', () => {
  test('initialer Fetch schreibt enabled aus der API-Antwort in den State', async () => {
    mockGet.mockResolvedValueOnce({ data: { enabled: true } })
    const { result } = renderHook(() => useMaintenanceStatus())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.enabled).toBe(true)
    expect(mockGet).toHaveBeenCalledWith('/maintenance-status')
  })

  test('fail-open: bei Fehler bleibt enabled=false', async () => {
    mockGet.mockRejectedValueOnce(new Error('network'))
    const { result } = renderHook(() => useMaintenanceStatus())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.enabled).toBe(false)
  })

  test('settings-changed-Event triggert Refetch', async () => {
    mockGet.mockResolvedValueOnce({ data: { enabled: false } })
    const { result } = renderHook(() => useMaintenanceStatus())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.enabled).toBe(false)

    mockGet.mockResolvedValueOnce({ data: { enabled: true } })
    liveHandler?.('settings-changed')
    await waitFor(() => expect(result.current.enabled).toBe(true))
    expect(mockGet).toHaveBeenCalledTimes(2)
  })

  test('andere SSE-Events lösen KEINEN Refetch aus', async () => {
    mockGet.mockResolvedValueOnce({ data: { enabled: false } })
    const { result } = renderHook(() => useMaintenanceStatus())
    await waitFor(() => expect(result.current.loading).toBe(false))
    liveHandler?.('games-updated')
    // Nur der initiale Call zählt.
    expect(mockGet).toHaveBeenCalledTimes(1)
  })
})
