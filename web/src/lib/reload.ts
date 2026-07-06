const WAITING_POLL_INTERVAL_MS = 250
const WAITING_POLL_TIMEOUT_MS = 5000
export const API_CACHE_NAME = 'api-cache'
const APP_SHELL_CACHE_NAME = 'app-shell'
const WORKBOX_PRECACHE_PREFIX = 'workbox-precache'

function activateWaitingAndReload(reg: ServiceWorkerRegistration): void {
  const fallback = setTimeout(() => location.reload(), 3000)
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    clearTimeout(fallback)
    location.reload()
  }, { once: true })
  reg.waiting!.postMessage({ type: 'SKIP_WAITING' })
}

function waitForWaiting(reg: ServiceWorkerRegistration, timeoutMs: number): Promise<ServiceWorker | null> {
  return new Promise((resolve) => {
    if (reg.waiting) return resolve(reg.waiting)
    const deadline = Date.now() + timeoutMs
    const tick = () => {
      if (reg.waiting) return resolve(reg.waiting)
      if (Date.now() >= deadline) return resolve(null)
      setTimeout(tick, WAITING_POLL_INTERVAL_MS)
    }
    setTimeout(tick, WAITING_POLL_INTERVAL_MS)
  })
}

export async function reloadWithSwActivation(): Promise<void> {
  const reg = await navigator.serviceWorker?.getRegistration()
  if (!reg) {
    location.reload()
    return
  }

  if (reg.waiting) {
    activateWaitingAndReload(reg)
    return
  }

  // SSE may have detected the new version before the browser has fetched the
  // new sw.js. Force a check and give it a short window to land.
  try {
    await reg.update()
  } catch {
    // Update may fail (offline, server error). Fall through to the wait/cache-clear path.
  }
  const waiting = await waitForWaiting(reg, WAITING_POLL_TIMEOUT_MS)
  if (waiting) {
    activateWaitingAndReload(reg)
    return
  }

  // No new SW arrived. As the emergency hammer, drop every cache that could
  // still serve the old app — the Workbox precache(s), the NetworkFirst
  // app-shell, and the API cache — so the reload below can only come from the
  // server. Unrelated caches (Google Fonts) are left untouched.
  try {
    const keys = await caches.keys()
    await Promise.all(
      keys
        .filter(
          (n) =>
            n === API_CACHE_NAME ||
            n === APP_SHELL_CACHE_NAME ||
            n.startsWith(WORKBOX_PRECACHE_PREFIX)
        )
        .map((n) => caches.delete(n))
    )
  } catch {
    // caches may be unavailable; ignore and still reload below.
  }
  location.reload()
}
