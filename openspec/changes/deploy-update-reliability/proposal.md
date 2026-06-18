## Why

Nach `make deploy` bleiben viele User — besonders im iOS-PWA-Modus — dauerhaft auf der alten App-Version hängen. Sie können das Update nur durch Logout + komplettes Beenden der PWA forcieren; der "Jetzt laden"-Banner ist unzuverlässig und erscheint manchmal gar nicht. Drei Root-Causes sind identifiziert: (1) der Go-`spaHandler` setzt keine `Cache-Control`/ETag-Header, embed.FS liefert keine Validators → Browser cached `sw.js`/`index.html` heuristisch; (2) der Service Worker precached `index.html`, beim PWA-Coldstart wird die alte Shell aus dem Workbox-Precache geliefert; (3) der Reload-Fallback in `reload.ts` löscht nur `api-cache`, nicht den Precache-Cache. Das untergräbt die existierende Update-Detection (siehe `deploy-version-detection`-Spec) — die Erkennung läuft, die Auslieferung nicht.

## What Changes

- **Backend Cache-Header** (`cmd/teamwerk/main.go`): `spaHandler` setzt pfad-abhängige `Cache-Control`-Header: hashed Assets unter `assets/*-<hash>.<ext>` erhalten `public, max-age=31536000, immutable`; alle anderen Pfade (`/`, `/index.html`, `/sw.js`, `/manifest.webmanifest`, `/registerSW.js`, `/icons/*`) erhalten `no-cache, must-revalidate` und einen aus `buildHash`+Pfad abgeleiteten ETag.
- **Service Worker Navigation-Route** (`web/src/sw.ts`, `web/vite.config.ts`): `index.html` wird aus dem Workbox-Precache entfernt; Navigationen (`request.mode === 'navigate'`) laufen über `NetworkFirst` mit Cache `app-shell` und 3 s Timeout. Offline-Fallback bleibt erhalten.
- **Reload-Fallback härten** (`web/src/lib/reload.ts`): wenn weder waiting SW noch `reg.update()` greifen, löscht der Notnagel-Pfad zusätzlich zu `api-cache` auch alle Caches mit Präfix `workbox-precache` sowie `app-shell`, bevor `location.reload()` läuft.

## Capabilities

### New Capabilities

- `app-update-reliability`: garantiert, dass nach einem Deploy jeder User innerhalb einer Session — ohne Logout/PWA-Beenden — die neue App-Version erhält. Deckt HTTP-Cache-Header für statische Assets, die SW-Navigation-Strategie und das Verhalten des Update-Reload-Fallbacks ab.

### Modified Capabilities

- `pwa-support`: der Service-Worker-Precache enthält `index.html` NICHT mehr; Navigationen laufen über NetworkFirst statt aus dem Precache. Offline-Shell-Fallback wird über den `app-shell`-Runtime-Cache gewährleistet, nicht mehr über Precache.

## Impact

- **Code**:
  - `cmd/teamwerk/main.go` — `spaHandler` um Cache-Header- und ETag-Logik erweitern.
  - `web/vite.config.ts` — `injectManifest.globPatterns` so einschränken, dass `index.html` aus dem Precache fällt.
  - `web/src/sw.ts` — neue `NetworkFirst`-Route für Navigationen vor allen anderen Routen.
  - `web/src/lib/reload.ts` — Cache-Cleanup um `workbox-precache*` und `app-shell` erweitern.
- **Tests**:
  - Neue Backend-Tests in `cmd/teamwerk/` für Cache-Header und ETag-Verhalten (`If-None-Match` → 304).
  - Optionaler Vitest für die `reload.ts`-Cache-Cleanup-Logik.
- **APIs**: keine API-Änderungen. SSE `__version:`-Event und `useVersionCheck`-Hook bleiben unverändert.
- **Dependencies**: keine neuen Abhängigkeiten.
- **Deployment / Runtime**: `make deploy` unverändert. Initial nach Rollout sehen User einmalig die alte Shell weiter (bis der neue SW aktiv ist); danach ist der Mechanismus dauerhaft self-healing.
- **Risiko**: NetworkFirst mit 3 s Timeout könnte bei langsamen Netzen Cold-Start verzögern; durch Offline-Fallback aber begrenzt. Bestehende Workbox-Routen (`api-cache`, Google Fonts, `/api/events` NetworkOnly, `/api/auth/*` NetworkOnly) bleiben unangetastet.
