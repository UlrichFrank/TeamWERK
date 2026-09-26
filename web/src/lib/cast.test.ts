import { describe, test, expect, afterEach, vi } from 'vitest'

// loadCastSDK unterscheidet 'unavailable' (Browser kann nicht casten) von
// 'load_failed' (Skript kam nicht an). Beides als `false` zusammenzufassen hat
// einen CSP-Block monatelang als „Browser ohne Cast" getarnt.

type W = { chrome?: unknown; cast?: unknown }

async function freshModule() {
  vi.resetModules()
  return import('./cast')
}

function injectedScript(): HTMLScriptElement {
  const s = document.head.querySelector<HTMLScriptElement>('script[src*="cast_sender.js"]')
  if (!s) throw new Error('Cast-Skript nicht eingehängt')
  return s
}

afterEach(() => {
  vi.useRealTimers()
  document.head.querySelectorAll('script').forEach((s) => s.remove())
  delete (window as unknown as W).chrome
  delete (window as unknown as W).cast
  delete window.__onGCastApiAvailable
})

describe('loadCastSDK', () => {
  test('ohne crossorigin-Attribut (gstatic sendet kein CORS)', async () => {
    const { loadCastSDK } = await freshModule()
    void loadCastSDK()
    expect(injectedScript().hasAttribute('crossorigin')).toBe(false)
  })

  test('Callback true mit Framework: available', async () => {
    const { loadCastSDK } = await freshModule()
    const p = loadCastSDK()
    ;(window as unknown as W).chrome = { cast: {} }
    ;(window as unknown as W).cast = { framework: {} }
    window.__onGCastApiAvailable?.(true)
    await expect(p).resolves.toBe('available')
  })

  test('Callback false: unavailable', async () => {
    const { loadCastSDK } = await freshModule()
    const p = loadCastSDK()
    window.__onGCastApiAvailable?.(false)
    await expect(p).resolves.toBe('unavailable')
  })

  test('Callback true, aber Framework blockiert: load_failed, nicht unavailable', async () => {
    const { loadCastSDK } = await freshModule()
    const p = loadCastSDK()
    window.__onGCastApiAvailable?.(true)
    await expect(p).resolves.toBe('load_failed')
  })

  test('Skript-Fehler: load_failed, erneuter Aufruf lädt neu', async () => {
    const { loadCastSDK } = await freshModule()
    const p = loadCastSDK()
    const first = injectedScript()
    first.dispatchEvent(new Event('error'))
    await expect(p).resolves.toBe('load_failed')
    expect(first.isConnected).toBe(false)

    void loadCastSDK()
    expect(injectedScript()).not.toBe(first)
  })

  test('kein Callback (Nachlade-Skript still blockiert): load_failed nach Timeout', async () => {
    vi.useFakeTimers()
    const { loadCastSDK, CAST_LOAD_TIMEOUT_MS } = await freshModule()
    const p = loadCastSDK()
    vi.advanceTimersByTime(CAST_LOAD_TIMEOUT_MS)
    await expect(p).resolves.toBe('load_failed')
  })
})
