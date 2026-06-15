const WAITING_POLL_INTERVAL_MS = 250
const WAITING_POLL_TIMEOUT_MS = 5000
const API_CACHE_NAME = 'api-cache'

function activateWaitingAndReload(reg: ServiceWorkerRegistration): void {
  navigator.serviceWorker.addEventListener('controllerchange', () => location.reload(), { once: true })
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

  // No new SW arrived. Clear the API cache so the reload at least surfaces
  // fresh server data — the precached shell may still be old, but that
  // resolves on the next SW activation.
  try {
    await caches.delete(API_CACHE_NAME)
  } catch {
    // caches may be unavailable; ignore.
  }
  location.reload()
}
