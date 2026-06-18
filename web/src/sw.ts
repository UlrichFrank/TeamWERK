/// <reference lib="webworker" />
import { precacheAndRoute } from 'workbox-precaching'
import { registerRoute } from 'workbox-routing'
import { CacheFirst, NetworkFirst, NetworkOnly } from 'workbox-strategies'
import { ExpirationPlugin } from 'workbox-expiration'

declare let self: ServiceWorkerGlobalScope & { __WB_MANIFEST: unknown[] }

precacheAndRoute(self.__WB_MANIFEST)

// Navigations: NetworkFirst so a fresh index.html (pointing at new asset
// hashes) wins whenever the network answers within 3s. index.html is NOT in
// the precache anymore; the 'app-shell' cache keeps the last good shell for
// offline / slow-network cold starts. Must be registered FIRST (first match
// wins) so it beats the /api/* and font routes below.
registerRoute(
  ({ request }) => request.mode === 'navigate',
  new NetworkFirst({
    cacheName: 'app-shell',
    networkTimeoutSeconds: 3,
    plugins: [new ExpirationPlugin({ maxEntries: 1, maxAgeSeconds: 60 * 60 * 24 * 30 })],
  })
)

// Google Fonts CSS
registerRoute(
  ({ url }) => url.origin === 'https://fonts.googleapis.com',
  new CacheFirst({
    cacheName: 'google-fonts-cache',
    plugins: [new ExpirationPlugin({ maxEntries: 10, maxAgeSeconds: 60 * 60 * 24 * 365 })],
  })
)

// Google Fonts static assets
registerRoute(
  ({ url }) => url.origin === 'https://fonts.gstatic.com',
  new CacheFirst({
    cacheName: 'google-fonts-static-cache',
    plugins: [new ExpirationPlugin({ maxEntries: 10, maxAgeSeconds: 60 * 60 * 24 * 365 })],
  })
)

// Auth routes: never cache
registerRoute(
  ({ url }) => url.pathname.startsWith('/api/auth/'),
  new NetworkOnly()
)

// SSE endpoints: NetworkOnly. text/event-stream is long-lived; NetworkFirst's clone-for-cache
// and timeout-fallback semantics break Reconnect and can serve stale __version: frames.
// These rules must come BEFORE the /api/* NetworkFirst rule below (first match wins).
registerRoute(
  ({ url }) => url.pathname === '/api/events' || url.pathname === '/api/chat/events',
  new NetworkOnly()
)

// Other API routes: network-first
registerRoute(
  ({ url }) => url.pathname.startsWith('/api/'),
  new NetworkFirst({ cacheName: 'api-cache', networkTimeoutSeconds: 10 })
)

// Push notification handler
self.addEventListener('push', (event) => {
  if (!event.data) return
  const data = event.data.json() as { title: string; body: string; url: string }
  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: '/icons/icon-192.png',
      badge: '/icons/icon-192.png',
      data: { url: data.url },
    })
  )
})

// Activate new SW on demand from the reload handler
self.addEventListener('message', (event) => {
  if ((event.data as { type: string })?.type === 'SKIP_WAITING') self.skipWaiting()
})

// Open the app at the correct URL when notification is clicked
self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = (event.notification.data as { url: string })?.url ?? '/'
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      const existing = clients.find((c) => c.url.includes(self.location.origin))
      if (existing) {
        existing.focus()
        existing.navigate(url)
      } else {
        self.clients.openWindow(url)
      }
    })
  )
})
