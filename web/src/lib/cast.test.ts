import { describe, test, expect, afterEach, vi } from 'vitest'
import { browserCanCast } from './cast'

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

describe('browserCanCast', () => {
  const UA = {
    chromeMac:
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36',
    chromeAndroid:
      'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Mobile Safari/537.36',
    edgeWin:
      'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36 Edg/140.0.0.0',
    safariIPhone:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1',
    chromeIPhone:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/140.0.0.0 Mobile/15E148 Safari/604.1',
    safariMac:
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15',
    firefoxMac: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:140.0) Gecko/20100101 Firefox/140.0',
  }

  test.each([
    ['Chrome macOS', UA.chromeMac, 0, true],
    ['Chrome Android', UA.chromeAndroid, 5, true],
    ['Edge Windows', UA.edgeWin, 0, true],
    ['Safari iPhone', UA.safariIPhone, 5, false],
    ['Chrome iPhone (WebKit)', UA.chromeIPhone, 5, false],
    ['Safari iPad (meldet sich als Mac)', UA.safariMac, 5, false],
    ['Safari macOS', UA.safariMac, 0, false],
    ['Firefox macOS', UA.firefoxMac, 0, false],
  ])('%s → %s', (_name, ua, touch, want) => {
    expect(browserCanCast(ua as string, touch as number)).toBe(want)
  })
})
