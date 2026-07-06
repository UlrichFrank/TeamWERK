import { describe, test, expect, afterEach, vi } from 'vitest'
import type { AxiosAdapter, AxiosResponse } from 'axios'
import { AxiosError } from 'axios'
import { api, setMaintenanceHandler } from './api'

/**
 * Custom axios adapter: liefert eine feste Antwort. Ohne MSW/mock-adapter —
 * axios' `adapter`-Option ist genau dafür da, per-Request die Transport-
 * Schicht zu ersetzen. Der Interceptor sitzt im Response-Path, wird also
 * ausgelöst wie in Prod.
 */
function fixedAdapter(status: number, headers: Record<string, string> = {}): AxiosAdapter {
  return async (config) => {
    const response: AxiosResponse = {
      data: {},
      status,
      statusText: '',
      headers,
      config,
      request: {},
    }
    if (status >= 400) {
      throw new AxiosError(
        `Request failed with status ${status}`,
        String(status),
        config,
        {},
        response,
      )
    }
    return response
  }
}

afterEach(() => {
  setMaintenanceHandler(null)
})

describe('api interceptor — Maintenance-503', () => {
  test('503 mit X-Maintenance-Mode: 1 ruft den Handler auf, Promise rejectet', async () => {
    const handler = vi.fn()
    setMaintenanceHandler(handler)

    await expect(
      api.request({ url: '/games', method: 'POST', adapter: fixedAdapter(503, { 'x-maintenance-mode': '1' }) }),
    ).rejects.toBeDefined()

    expect(handler).toHaveBeenCalledTimes(1)
  })

  test('generischer 503 (ohne Header) ruft den Handler NICHT auf', async () => {
    const handler = vi.fn()
    setMaintenanceHandler(handler)

    await expect(
      api.request({ url: '/games', method: 'POST', adapter: fixedAdapter(503) }),
    ).rejects.toBeDefined()

    expect(handler).not.toHaveBeenCalled()
  })

  test('kein registrierter Handler ist unproblematisch', async () => {
    setMaintenanceHandler(null)
    await expect(
      api.request({ url: '/games', method: 'POST', adapter: fixedAdapter(503, { 'x-maintenance-mode': '1' }) }),
    ).rejects.toBeDefined()
  })

  test('erfolgreicher Request lässt den Handler in Ruhe', async () => {
    const handler = vi.fn()
    setMaintenanceHandler(handler)

    const res = await api.request({ url: '/games', method: 'GET', adapter: fixedAdapter(200) })
    expect(res.status).toBe(200)
    expect(handler).not.toHaveBeenCalled()
  })
})
