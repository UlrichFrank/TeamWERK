// Schlanker Wrapper um Matomos globales `_paq`-Array.
// Wird in main.tsx einmal initialisiert; AppShell ruft die Helper bei Routenwechseln auf.
// Ohne Konfiguration (leere URL / SiteID) sind alle Aufrufe No-Ops — Dev-Builds und
// nicht konfigurierte Umgebungen senden nichts.

type PaqCommand = readonly unknown[]

declare global {
  interface Window {
    _paq?: PaqCommand[]
  }
  interface Navigator {
    standalone?: boolean
  }
}

const DIM_CHANNEL = 1
const DIM_TEAM_SLUG = 2
const DIM_ROLE = 3

let enabled = false

function push(cmd: PaqCommand): void {
  if (!enabled) return
  window._paq = window._paq ?? []
  window._paq.push(cmd)
}

export function initTelemetry(url: string | undefined, siteId: number | undefined): void {
  if (!url || !siteId || Number.isNaN(siteId)) {
    enabled = false
    return
  }

  const trackerUrl = url.endsWith('/') ? url : `${url}/`
  window._paq = window._paq ?? []
  window._paq.push(['disableCookies'])
  window._paq.push(['setSecureCookie', true])
  window._paq.push(['setRequestMethod', 'POST'])
  window._paq.push(['setTrackerUrl', `${trackerUrl}matomo.php`])
  window._paq.push(['setSiteId', String(siteId)])
  enabled = true

  // matomo.js asynchron nachladen — Browser cached die Datei.
  if (typeof document !== 'undefined') {
    const existing = document.querySelector<HTMLScriptElement>('script[data-matomo]')
    if (!existing) {
      const script = document.createElement('script')
      script.async = true
      script.src = `${trackerUrl}matomo.js`
      script.dataset.matomo = 'true'
      document.head.appendChild(script)
    }
  }
}

export function isTelemetryEnabled(): boolean {
  return enabled
}

export function detectChannel(): 'pwa' | 'browser' {
  if (typeof window === 'undefined') return 'browser'
  const standaloneMedia = window.matchMedia?.('(display-mode: standalone)').matches === true
  const iosStandalone = window.navigator?.standalone === true
  return standaloneMedia || iosStandalone ? 'pwa' : 'browser'
}

// Slugifiziert einen Teamnamen für die team_slug Custom Dimension.
// Beispiele: "H1" → "h1", "F-Jugend" → "f-jugend", "Männer Ü40" → "maenner-ue40".
export function slugifyTeam(name: string): string {
  return name
    .toLowerCase()
    .replace(/ä/g, 'ae')
    .replace(/ö/g, 'oe')
    .replace(/ü/g, 'ue')
    .replace(/ß/g, 'ss')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

export function setChannelDimension(channel: 'pwa' | 'browser' = detectChannel()): void {
  push(['setCustomDimension', DIM_CHANNEL, channel])
}

export function setTeamSlugDimension(slug: string): void {
  push(['setCustomDimension', DIM_TEAM_SLUG, slug])
}

export function setRoleDimension(role: string): void {
  // Wir senden nur 'admin' oder 'standard' — alles andere wird zu 'standard' normalisiert.
  push(['setCustomDimension', DIM_ROLE, role === 'admin' ? 'admin' : 'standard'])
}

export function trackPageview(href: string, title: string): void {
  push(['setCustomUrl', href])
  push(['setDocumentTitle', title])
  push(['trackPageView'])
}

// Nur für Tests: erzwingt einen sauberen Zustand zwischen Testfällen.
export function __resetTelemetryForTests(): void {
  enabled = false
  if (typeof window !== 'undefined') {
    window._paq = undefined
  }
}
