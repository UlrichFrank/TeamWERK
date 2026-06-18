import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reloadWithSwActivation } from './reload'

describe('reloadWithSwActivation fallback cache cleanup', () => {
  let deleted: string[]
  let reloadSpy: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.useFakeTimers()
    deleted = []

    const cacheNames = new Set([
      'api-cache',
      'app-shell',
      'workbox-precache-v2-https://intern.team-stuttgart.org/',
      'google-fonts-cache',
      'google-fonts-static-cache',
    ])

    vi.stubGlobal('caches', {
      keys: async () => [...cacheNames],
      delete: async (name: string) => {
        deleted.push(name)
        return cacheNames.delete(name)
      },
    })

    // Registration that never produces a waiting SW → forces the fallback path.
    const reg = { waiting: null, update: vi.fn(async () => {}) }
    vi.stubGlobal('navigator', {
      serviceWorker: {
        getRegistration: async () => reg,
        addEventListener: vi.fn(),
      },
    })

    reloadSpy = vi.fn()
    vi.stubGlobal('location', { reload: reloadSpy })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('deletes api-cache, app-shell and workbox-precache, leaves fonts, then reloads', async () => {
    const p = reloadWithSwActivation()
    // Exhaust the 5s waitForWaiting poll so the fallback path runs.
    await vi.advanceTimersByTimeAsync(6000)
    await p

    expect(deleted).toContain('api-cache')
    expect(deleted).toContain('app-shell')
    expect(deleted).toContain('workbox-precache-v2-https://intern.team-stuttgart.org/')
    expect(deleted).toHaveLength(3)
    expect(deleted).not.toContain('google-fonts-cache')
    expect(deleted).not.toContain('google-fonts-static-cache')
    expect(reloadSpy).toHaveBeenCalledTimes(1)
  })
})
