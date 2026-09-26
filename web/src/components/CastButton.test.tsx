import { describe, test, expect, afterEach, vi } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import CastButton from './CastButton'

// Die Cast-API hängt an window.chrome.cast/window.cast.framework — beide fehlen
// in jsdom. Der Button-Render-Kontrakt: sichtbar, sobald die API verfügbar ist
// (dann trägt der User seinen Wurf initiativ aus); versteckt, sobald ein
// Ladeversuch fehlgeschlagen ist. In der jsdom-Default-Umgebung ist die API
// noch nicht da, aber wir haben nicht geklickt — der Button bleibt sichtbar.

const CHROME_MAC =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36'
const SAFARI_IPHONE =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1'

function setUserAgent(ua: string) {
  vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue(ua)
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  // window.chrome kann per test injiziert worden sein — zurücksetzen.
  delete (window as { chrome?: unknown }).chrome
  delete (window as { cast?: unknown }).cast
})

describe('CastButton', () => {
  test('renders when Cast API is already injected before mount', () => {
    ;(window as unknown as { chrome: unknown; cast: unknown }).chrome = {
      cast: { media: {}, AutoJoinPolicy: {} },
    }
    ;(window as unknown as { chrome: unknown; cast: unknown }).cast = {
      framework: { CastContext: { getInstance: vi.fn() } },
    }
    render(<CastButton masterURL="/api/videos/1/hls/master.m3u8?st=abc" />)
    const btn = screen.queryByRole('button', { name: /Chromecast/i })
    expect(btn).not.toBeNull()
  })

  test('renders optimistically without Cast API (button stays until first click fails)', () => {
    // Kein window.chrome, kein window.cast — API nicht verfügbar, aber Chrome.
    setUserAgent(CHROME_MAC)
    render(<CastButton masterURL="/api/videos/1/hls/master.m3u8?st=abc" />)
    // Vor Klick: Button ist sichtbar (optimistisches Rendering für Chrome/Android).
    const btn = screen.queryByRole('button', { name: /Chromecast/i })
    expect(btn).not.toBeNull()
  })

  test('Safari auf dem iPhone: kein Button, kein Ladeversuch', () => {
    setUserAgent(SAFARI_IPHONE)
    render(<CastButton masterURL="/api/videos/1/hls/master.m3u8?st=abc" />)
    expect(screen.queryByRole('button', { name: /Chromecast/i })).toBeNull()
    expect(document.head.querySelector('script[src*="cast_sender.js"]')).toBeNull()
  })
})
