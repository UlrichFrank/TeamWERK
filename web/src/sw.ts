/// <reference lib="webworker" />
import { precacheAndRoute } from 'workbox-precaching'
import { registerRoute } from 'workbox-routing'
import { CacheFirst, NetworkFirst, NetworkOnly, StaleWhileRevalidate } from 'workbox-strategies'
import { ExpirationPlugin } from 'workbox-expiration'

declare let self: ServiceWorkerGlobalScope & { __WB_MANIFEST: unknown[] }

precacheAndRoute(self.__WB_MANIFEST)

// Benutzerhandbuch: statische HTML-Seite mit eigenem Cache (vor der
// app-shell-Route, sonst überschreibt eine Doku-Navigation den app-shell-
// Eintrag mit maxEntries: 1 und der nächste Offline-Start lädt das Handbuch
// statt der App).
registerRoute(
  ({ url }) => url.pathname === '/benutzerhandbuch.html',
  new NetworkFirst({
    cacheName: 'docs-cache',
    networkTimeoutSeconds: 5,
    plugins: [new ExpirationPlugin({ maxEntries: 3, maxAgeSeconds: 60 * 60 * 24 * 30 })],
  })
)

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

// Referenz-Endpunkte: StaleWhileRevalidate — der gecachte Stand wird sofort
// ausgeliefert und im Hintergrund aus dem Netz erneuert. Quasi-statisch, daher
// unkritisch, wenn ein Abruf kurz veraltet ist (der In-Memory-TTL-Cache in
// api.ts + SSE-Invalidierung halten die App-Daten frisch).
// MUSS vor der generischen /api/*-Regel stehen (first match wins).
//
// NUR club-weit für ALLE authentifizierten Nutzer identische Routen. `/api/teams`
// ist bewusst NICHT dabei: `Games.ListTeamsForUser` filtert pro Nutzer → ein
// geteilter (geräteweiter) SW-Cache würde auf einem gemeinsamen Gerät nach
// Login-Wechsel die Teams des Vor-Nutzers ausliefern (Cross-User-Leak). `/api/teams`
// fällt daher auf die NetworkFirst-Regel unten zurück; der nutzerspezifische
// In-Memory-Cache in api.ts wird bei Identitätswechsel (setAccessToken) geleert.
const REFERENCE_PATHS = new Set([
  '/api/seasons',
  '/api/venues',
  '/api/age-class-rules',
  '/api/duty-types',
])
registerRoute(
  ({ url }) => REFERENCE_PATHS.has(url.pathname),
  new StaleWhileRevalidate({
    cacheName: 'api-reference-cache',
    plugins: [new ExpirationPlugin({ maxEntries: 32, maxAgeSeconds: 60 * 60 * 24 })],
  })
)

// Other API routes: network-first. Grenze setzen, damit der api-cache nicht
// unbegrenzt wächst (früher ohne maxEntries/maxAgeSeconds → monotones Wachstum).
registerRoute(
  ({ url }) => url.pathname.startsWith('/api/'),
  new NetworkFirst({
    cacheName: 'api-cache',
    networkTimeoutSeconds: 10,
    plugins: [new ExpirationPlugin({ maxEntries: 64, maxAgeSeconds: 60 * 60 * 24 })],
  })
)

// Push notification handler
self.addEventListener('push', (event) => {
  if (!event.data) return
  const data = event.data.json() as { title: string; body: string; url: string; badge?: number }
  const tasks: Promise<unknown>[] = [
    self.registration.showNotification(data.title, {
      body: data.body,
      // `icon` = große, farbige Vorschau in der aufgeklappten Notification.
      icon: '/icons/icon-192.png',
      // `badge` = monochromes Status-Bar-ICON (Android rendert nur den Alpha-
      // Kanal weiß). Eigene Silhouette statt der vollflächigen Kreisfläche.
      // Nicht zu verwechseln mit `data.badge` unten = App-Icon-Zahl (PR #46).
      badge: '/icons/badge-96.png',
      data: { url: data.url },
    }),
  ]
  if (typeof data.badge === 'number') {
    const nav = self.navigator as Navigator & {
      setAppBadge?: (n?: number) => Promise<void>
      clearAppBadge?: () => Promise<void>
    }
    if ('setAppBadge' in nav) {
      // .catch() ist Pflicht: rejected setAppBadge sonst Promise.all und damit
      // event.waitUntil — iOS wertet das als fehlgeschlagenen Push-Handler und
      // liefert nichts mehr aus.
      const badgePromise = data.badge > 0
        ? nav.setAppBadge?.(data.badge)
        : nav.clearAppBadge?.()
      if (badgePromise) tasks.push(badgePromise.catch(() => {}))
    }
  }
  event.waitUntil(Promise.all(tasks))
})

// Activate new SW on demand from the reload handler
self.addEventListener('message', (event) => {
  if ((event.data as { type: string })?.type === 'SKIP_WAITING') self.skipWaiting()
})

// Open the app at the correct URL when notification is clicked.
//
// `data.url` kann bewusst leer sein (Absage-Benachrichtigungen ohne Ziel, siehe
// design.md §3): `?? '/'` greift nur bei null/undefined, NICHT beim leeren String —
// ein naives `navigate('')` löst relativ gegen die Client-URL auf und lädt die
// gerade offene Seite neu. Die Zielauflösung steckt deshalb in einer exportierten,
// reinen Funktion, testbar ohne Service-Worker-Laufzeit (analog zur exportierten
// Factory in VideoUploadPage.tsx).
export function resolveClickTarget(data: unknown): { navigate: boolean; url: string } {
  if (typeof data !== 'object' || data === null) return { navigate: false, url: '/' }
  const url = (data as { url?: unknown }).url
  if (typeof url !== 'string' || url === '') return { navigate: false, url: '/' }
  return { navigate: true, url }
}

// Minimal an Shape, die eine Fokussierung/Navigation braucht — bewusst schmaler
// als `WindowClient`, damit Tests einfache Fakes (statt echter Service-Worker-
// Clients) übergeben können.
interface FocusableClient {
  url: string
  focus(): unknown
  navigate(url: string): unknown
}

// Client-Auswahl + Fokus-/Navigations-Entscheidung, getrennt von der
// addEventListener-Verdrahtung exportiert, damit der Pfad ohne echte
// Service-Worker-Clients testbar ist. Verhalten für gesetzte URLs unverändert:
// Fenster fokussieren + navigieren (bzw. neues Fenster mit der Ziel-URL öffnen).
// Bei `navigate: false` wird NICHT navigiert — nur fokussiert bzw. die App-Wurzel
// geöffnet, falls kein Fenster existiert.
export function applyClickTarget(
  target: { navigate: boolean; url: string },
  clientList: readonly FocusableClient[],
  origin: string,
  openWindow: (url: string) => void
): void {
  const existing = clientList.find((c) => c.url.includes(origin))
  if (existing) {
    existing.focus()
    if (target.navigate) existing.navigate(target.url)
  } else {
    openWindow(target.navigate ? target.url : '/')
  }
}

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const target = resolveClickTarget(event.notification.data)
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      applyClickTarget(target, clientList, self.location.origin, (url) => self.clients.openWindow(url))
    })
  )
})
