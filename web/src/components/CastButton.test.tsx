import { describe, test, expect, afterEach, vi } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import CastButton from './CastButton'

// Die Cast-API hängt an window.chrome.cast/window.cast.framework — beide fehlen
// in jsdom. Der Button-Render-Kontrakt: sichtbar, sobald die API verfügbar ist
// (dann trägt der User seinen Wurf initiativ aus); versteckt, sobald ein
// Ladeversuch fehlgeschlagen ist. In der jsdom-Default-Umgebung ist die API
// noch nicht da, aber wir haben nicht geklickt — der Button bleibt sichtbar.

afterEach(() => {
  cleanup()
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
    // Kein window.chrome, kein window.cast — API nicht verfügbar.
    render(<CastButton masterURL="/api/videos/1/hls/master.m3u8?st=abc" />)
    // Vor Klick: Button ist sichtbar (optimistisches Rendering für Chrome/Android).
    const btn = screen.queryByRole('button', { name: /Chromecast/i })
    expect(btn).not.toBeNull()
  })
})
