import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  initTelemetry,
  isTelemetryEnabled,
  detectChannel,
  slugifyTeam,
  setChannelDimension,
  setTeamSlugDimension,
  setRoleDimension,
  trackPageview,
  __resetTelemetryForTests,
} from '../telemetry'

function setMatchMedia(matches: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
      onchange: null,
    })),
  })
}

function clearScripts() {
  document.head.querySelectorAll('script[data-matomo]').forEach((s) => s.remove())
}

describe('telemetry — detectChannel', () => {
  beforeEach(() => {
    setMatchMedia(false)
    Object.defineProperty(window.navigator, 'standalone', { value: undefined, configurable: true })
  })

  test('display-mode standalone wird als pwa erkannt', () => {
    setMatchMedia(true)
    expect(detectChannel()).toBe('pwa')
  })

  test('iOS navigator.standalone === true wird als pwa erkannt', () => {
    Object.defineProperty(window.navigator, 'standalone', { value: true, configurable: true })
    expect(detectChannel()).toBe('pwa')
  })

  test('normaler Browser wird als browser erkannt', () => {
    expect(detectChannel()).toBe('browser')
  })
})

describe('telemetry — slugifyTeam', () => {
  test('einfacher Name → lowercase', () => {
    expect(slugifyTeam('H1')).toBe('h1')
  })
  test('mit Bindestrich', () => {
    expect(slugifyTeam('F-Jugend')).toBe('f-jugend')
  })
  test('mit Umlauten und Leerzeichen', () => {
    expect(slugifyTeam('Männer Ü40')).toBe('maenner-ue40')
  })
  test('mit ß und Sonderzeichen', () => {
    expect(slugifyTeam('Straßen-Team!')).toBe('strassen-team')
  })
})

describe('telemetry — disabled state', () => {
  beforeEach(() => {
    __resetTelemetryForTests()
    clearScripts()
  })
  afterEach(() => {
    __resetTelemetryForTests()
    clearScripts()
  })

  test('leere URL → deaktiviert, kein _paq, kein script', () => {
    initTelemetry('', 1)
    expect(isTelemetryEnabled()).toBe(false)
    expect(document.querySelector('script[data-matomo]')).toBeNull()
  })

  test('fehlende siteId → deaktiviert', () => {
    initTelemetry('https://matomo.example.com', NaN)
    expect(isTelemetryEnabled()).toBe(false)
  })

  test('dimension/track-Calls im disabled state sind No-Ops', () => {
    initTelemetry(undefined, undefined)
    setChannelDimension('pwa')
    setTeamSlugDimension('h1')
    setRoleDimension('admin')
    trackPageview('/x', 'X')
    expect(window._paq).toBeUndefined()
  })
})

describe('telemetry — enabled state', () => {
  beforeEach(() => {
    __resetTelemetryForTests()
    clearScripts()
  })
  afterEach(() => {
    __resetTelemetryForTests()
    clearScripts()
  })

  test('gültige Konfig → enabled, _paq mit Init-Befehlen, script geladen', () => {
    initTelemetry('https://matomo.example.com', 7)
    expect(isTelemetryEnabled()).toBe(true)
    expect(window._paq).toBeDefined()
    const cmds = window._paq!
    expect(cmds).toEqual(
      expect.arrayContaining([
        ['disableCookies'],
        ['setSecureCookie', true],
        ['setRequestMethod', 'POST'],
        ['setTrackerUrl', 'https://matomo.example.com/matomo.php'],
        ['setSiteId', '7'],
      ]),
    )
    expect(document.querySelector('script[data-matomo]')).not.toBeNull()
  })

  test('URL ohne trailing slash wird normalisiert', () => {
    initTelemetry('https://matomo.example.com', 7)
    expect(window._paq).toEqual(
      expect.arrayContaining([['setTrackerUrl', 'https://matomo.example.com/matomo.php']]),
    )
  })

  test('setChannelDimension pusht Custom Dim 1', () => {
    initTelemetry('https://matomo.example.com', 7)
    setChannelDimension('pwa')
    expect(window._paq).toEqual(
      expect.arrayContaining([['setCustomDimension', 1, 'pwa']]),
    )
  })

  test('setTeamSlugDimension pusht Custom Dim 2', () => {
    initTelemetry('https://matomo.example.com', 7)
    setTeamSlugDimension('h1')
    expect(window._paq).toEqual(
      expect.arrayContaining([['setCustomDimension', 2, 'h1']]),
    )
  })

  test('setRoleDimension normalisiert Nicht-admin zu standard', () => {
    initTelemetry('https://matomo.example.com', 7)
    setRoleDimension('admin')
    setRoleDimension('standard')
    setRoleDimension('weird-other-value')
    expect(window._paq).toEqual(
      expect.arrayContaining([
        ['setCustomDimension', 3, 'admin'],
        ['setCustomDimension', 3, 'standard'],
      ]),
    )
    // 'weird-other-value' → 'standard'
    const stdCount = window._paq!.filter(c => c[0] === 'setCustomDimension' && c[1] === 3 && c[2] === 'standard').length
    expect(stdCount).toBe(2)
  })

  test('trackPageview pusht URL, Title, trackPageView', () => {
    initTelemetry('https://matomo.example.com', 7)
    trackPageview('https://app.example/dienste', 'Dienste')
    const cmds = window._paq!
    expect(cmds).toEqual(
      expect.arrayContaining([
        ['setCustomUrl', 'https://app.example/dienste'],
        ['setDocumentTitle', 'Dienste'],
        ['trackPageView'],
      ]),
    )
  })

  test('script-Tag wird bei zweitem Init nicht dupliziert', () => {
    initTelemetry('https://matomo.example.com', 7)
    initTelemetry('https://matomo.example.com', 7)
    expect(document.querySelectorAll('script[data-matomo]').length).toBe(1)
  })
})
