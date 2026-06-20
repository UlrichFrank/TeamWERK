import { describe, it, expect } from 'vitest'

// Regressions-Invarianten für die PWA-Icons (Change: android-pwa-icons).
// Das tatsächliche Android-Rendering ist nur manuell prüfbar; diese Tests
// sichern die statischen Voraussetzungen ab, damit sie nicht zurückfallen.
// Dateien werden Vite-nativ (?raw / import.meta.glob) eingelesen — kein node:fs,
// damit der Test auch unter `tsc -b` (ohne @types/node) typt.

const iconAssets = import.meta.glob(
  '../../public/icons/{icon-maskable-512,badge-96}.png',
  { eager: true, query: '?raw', import: 'default' },
) as Record<string, string>

const manifestJson = import.meta.glob('../../public/manifest.json', {
  eager: true,
  query: '?raw',
  import: 'default',
})

const byName = (suffix: string) =>
  Object.entries(iconAssets).find(([k]) => k.endsWith(suffix))?.[1]

describe('PWA icon assets', () => {
  it('maskable Icon existiert und ist nicht trivial', () => {
    const content = byName('icon-maskable-512.png')
    expect(content).toBeDefined()
    expect(content!.length).toBeGreaterThan(1024)
  })

  it('Notification-Badge existiert und ist nicht trivial', () => {
    const content = byName('badge-96.png')
    expect(content).toBeDefined()
    expect(content!.length).toBeGreaterThan(1024)
  })
})

describe('Manifest-Konsolidierung', () => {
  it('keine statische public/manifest.json mehr', () => {
    expect(Object.keys(manifestJson)).toHaveLength(0)
  })

  it('index.html enthält keinen manuellen Manifest-Link', async () => {
    const html = (await import('../../index.html?raw')).default
    expect(html).not.toMatch(/rel=["']manifest["']/)
  })
})

describe('vite.config Manifest-Purposes', () => {
  it('genau ein maskable-Eintrag, kein gemischtes "any maskable"', async () => {
    const cfg = (await import('../../vite.config.ts?raw')).default
    const maskable = cfg.match(/purpose:\s*['"]maskable['"]/g) ?? []
    expect(maskable).toHaveLength(1)
    expect(cfg).not.toMatch(/purpose:\s*['"]any maskable['"]/)
  })
})

describe('Service Worker Badge', () => {
  it('referenziert das eigene Badge-Icon', async () => {
    const sw = (await import('../sw.ts?raw')).default
    expect(sw).toMatch(/badge:\s*['"]\/icons\/badge-96\.png['"]/)
  })

  // Regression: iOS hat alle Pushes verworfen, weil rejecting setAppBadge/
  // clearAppBadge das event.waitUntil-Promise mit gerissen hat. Jeder Aufruf
  // muss .catch(...) tragen, damit ein Badge-Fehler die Notification nicht killt.
  it('setAppBadge/clearAppBadge-Aufrufe sind mit .catch abgesichert', async () => {
    const sw = (await import('../sw.ts?raw')).default
    const matches = sw.match(/(setAppBadge|clearAppBadge)\?\.\([^)]*\)/g) ?? []
    expect(matches.length).toBeGreaterThan(0)
    for (const call of matches) {
      const idx = sw.indexOf(call)
      const window = sw.slice(idx, idx + call.length + 200)
      expect(window, `${call} braucht .catch-Absicherung`).toMatch(/\.catch\(/)
    }
  })
})
