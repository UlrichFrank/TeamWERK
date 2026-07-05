import axios from 'axios'

let accessToken: string | null = null
let refreshPromise: Promise<string> | null = null
let maintenanceHandler: (() => void) | null = null

export const api = axios.create({ baseURL: '/api', withCredentials: true })

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
