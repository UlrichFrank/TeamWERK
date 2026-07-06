import axios, { AxiosRequestConfig } from 'axios'
import { API_CACHE_NAME } from './reload'

let accessToken: string | null = null
let refreshPromise: Promise<string> | null = null
let maintenanceHandler: (() => void) | null = null

export const api = axios.create({ baseURL: '/api', withCredentials: true })

// uid-Claim aus dem Access-Token (ohne Verifikation — nur zur Erkennung eines
// Identitätswechsels für die Cache-Invalidierung).
function tokenUid(token: string | null): string | null {
  if (!token) return null
  try {
    const payload = JSON.parse(atob(token.split('.')[1])) as { uid?: number }
    return payload.uid != null ? String(payload.uid) : null
  } catch {
    return null
  }
}

/**
 * Registriert einen Callback, der bei einer Maintenance-503-Antwort
 * (Status 503 + Header `X-Maintenance-Mode: 1`) aufgerufen wird. Der Aufrufer
 * (typischerweise der `AppShell`) zeigt darauf einen freundlichen Toast/Dialog.
 * Reguläre 503-Responses (Upstream-Timeout, LB-Fehler) triggern den Callback
 * NICHT. Übergabe von `null` löscht die Registrierung (für Tests).
 */
export function setMaintenanceHandler(fn: (() => void) | null) {
  maintenanceHandler = fn
}

export function setAccessToken(token: string | null) {
  // Identitätswechsel (Logout, Login, Impersonation) leert den Referenz-Cache,
  // damit nie nutzergefilterte Daten (z. B. /teams) eines anderen Nutzers aus
  // dem Cache bedient werden. Ein Token-Refresh desselben Nutzers behält ihn.
  if (tokenUid(token) !== tokenUid(accessToken)) {
    clearReferenceCache()
    // Zusätzlich den geräteweiten SW-`api-cache` (NetworkFirst) leeren: er hält
    // nutzerspezifische Antworten (z. B. /api/teams, /api/members) und würde
    // sonst auf einem geteilten Gerät nach Nutzerwechsel bei langsamem Netz/
    // Offline die Daten des Vor-Nutzers ausliefern (Cross-User-Leak). Der SW-
    // `api-reference-cache` bleibt bewusst unangetastet: er enthält nur
    // club-weit für ALLE Nutzer identische Referenzrouten (kein Nutzerbezug).
    // Fire-and-forget, damit setAccessToken synchron bleibt (Aufrufer erwarten das).
    if (typeof caches !== 'undefined') {
      caches.delete(API_CACHE_NAME).catch(() => {})
    }
  }
  accessToken = token
}

export function getAccessToken(): string | null {
  return accessToken
}

// Single-Flight-Refresh: erneuert den Access-Token über /api/auth/refresh und
// speichert ihn im Store. Wird EIN Refresh bereits ausgeführt, warten alle
// weiteren Aufrufer auf dieselbe Promise — verhindert parallele Refresh-Calls
// (Axios-Interceptor + tus-Upload-Hook), die sonst den serverseitig rotierenden
// Refresh-Token gegenseitig invalidieren würden. Nach Abschluss (Erfolg wie
// Fehler) wird die Promise zurückgesetzt, damit ein späterer 401 neu refresht.
export function refreshAccessToken(): Promise<string> {
  if (!refreshPromise) {
    refreshPromise = axios
      .post('/api/auth/refresh', {}, { withCredentials: true })
      .then(res => {
        const t = res.data.access_token as string
        setAccessToken(t)
        return t
      })
      .finally(() => { refreshPromise = null })
  }
  return refreshPromise
}

// Aktuell laufender Refresh (oder null). Der tus-Upload-Hook (onBeforeRequest)
// wartet hierauf, damit ein durch 401 ausgelöster Retry erst nach Abschluss des
// Refreshs feuert und den neuen Token trägt.
export function getPendingRefresh(): Promise<string> | null {
  return refreshPromise
}

api.interceptors.request.use(config => {
  if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`
  return config
})

api.interceptors.response.use(
  res => res,
  async err => {
    const original = err.config
    if (err.response?.status === 401 && !original._retry) {
      original._retry = true
      try {
        const newToken = await refreshAccessToken()
        original.headers.Authorization = `Bearer ${newToken}`
        return api(original)
      } catch {
        setAccessToken(null)
        window.location.href = '/login'
      }
    }
    // Wartungsmodus-503 vom Server: Trennt sich vom generischen 503 durch den
    // Header `X-Maintenance-Mode: 1`. Der Handler wird bewusst NICHT
    // aufgerufen, wenn er nicht gesetzt ist — dann fällt der Fehler wie
    // gewohnt an den Caller durch (der ihn typischerweise als Toast zeigt).
    if (err.response?.status === 503 && err.response?.headers?.['x-maintenance-mode'] === '1') {
      if (maintenanceHandler) maintenanceHandler()
    }
    return Promise.reject(err)
  },
)

// ── Client-TTL-Cache + Single-Flight für Referenzdaten ───────────────────────
//
// Quasi-statische Referenzrouten (Teams, Saisons, Venues, …) werden bei jedem
// Seiten-Mount neu geladen. Statt eines State-Managers (react-query/SWR — RAM-/
// Bundle-Budget, VPS 1 GB) hält dieser dünne In-Memory-Cache pro URL das
// letzte Ergebnis für eine kurze, routen-spezifische TTL. Parallele Abrufe
// derselben URL teilen sich EINEN HTTP-Request (Single-Flight). Der Cache greift
// NUR für die Allowlist unten; alle anderen Calls laufen unverändert über `api`.
//
// Komplementär zum Server-`ETag`/`304`: die TTL spart den Request ganz, das 304
// spart die Payload, wenn der Request doch rausgeht (nach TTL-Ablauf oder
// SSE-Invalidierung).

interface ReferenceRoute {
  // TTL in Millisekunden.
  ttl: number
  // SSE-Event-Typen, die diesen Cache-Eintrag verwerfen.
  invalidatedBy: string[]
}

// Schlüssel = URL relativ zu `/api` (wie an api.get übergeben), ohne Query.
const REFERENCE_ROUTES: Record<string, ReferenceRoute> = {
  '/seasons': { ttl: 60 * 60 * 1000, invalidatedBy: ['seasons', 'settings'] },
  '/teams': { ttl: 5 * 60 * 1000, invalidatedBy: ['settings', 'kader', 'seasons'] },
  '/venues': { ttl: 24 * 60 * 60 * 1000, invalidatedBy: ['venues'] },
  '/age-class-rules': { ttl: 24 * 60 * 60 * 1000, invalidatedBy: ['settings'] },
  '/duty-types': { ttl: 5 * 60 * 1000, invalidatedBy: ['duties'] },
}

interface CacheEntry {
  data: unknown
  expires: number
}

const referenceCache = new Map<string, CacheEntry>()
const inflight = new Map<string, Promise<unknown>>()

// clearReferenceCache leert den gesamten Referenz-Cache (Identitätswechsel).
export function clearReferenceCache() {
  referenceCache.clear()
  inflight.clear()
}

// invalidateReferenceCache verwirft alle Cache-Einträge, deren Route auf das
// Live-Update-Event `event` hört. Wird von useLiveUpdates aufgerufen.
export function invalidateReferenceCache(event: string) {
  for (const [key, route] of Object.entries(REFERENCE_ROUTES)) {
    if (route.invalidatedBy.includes(event)) {
      referenceCache.delete(key)
      inflight.delete(key)
    }
  }
}

// getReference lädt eine Referenzroute mit TTL-Cache + Single-Flight. `url` ist
// relativ zu `/api` (z. B. '/teams'). Nicht-Allowlist-URLs gehen ohne Cache
// direkt an `api.get`. `force` umgeht den Cache (kritische Flows), füllt ihn
// aber neu. Bei einem Netzfehler wird der Eintrag nicht vergiftet.
export async function getReference<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  const route = REFERENCE_ROUTES[url]
  const force = (config as { force?: boolean } | undefined)?.force === true
  if (!route) {
    const res = await api.get<T>(url, config)
    return res.data
  }

  const now = Date.now()
  if (!force) {
    const cached = referenceCache.get(url)
    if (cached && cached.expires > now) return cached.data as T

    const pending = inflight.get(url)
    if (pending) return pending as Promise<T>
  }

  const request = api
    .get<T>(url, config)
    .then(res => {
      referenceCache.set(url, { data: res.data, expires: Date.now() + route.ttl })
      return res.data
    })
    .finally(() => {
      inflight.delete(url)
    })
  inflight.set(url, request)
  return request as Promise<T>
}
