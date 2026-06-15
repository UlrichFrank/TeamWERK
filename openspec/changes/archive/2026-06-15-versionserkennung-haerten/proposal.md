## Why

Zwei reale Bugs im aktuellen Produktivstand:

1. **Versionslink in der Sidebar fehlt häufig.** Der Button unterhalb von „Abmelden" (Anzeige `v <git-sha>`) wird nur gerendert, wenn `useVersionCheck()` eine Version geliefert hat. Real beobachten Anwender:innen jedoch oft keinen Link — die initiale `__version:`-SSE-Message kommt nicht zuverlässig durch.
2. **Update-Banner erscheint selten, App bleibt auf altem Stand.** Nach einem Deploy sehen viele Anwender:innen weiterhin den alten Stand mit alten gecachten Daten. Erst mehrfaches Ausloggen und Browser-/App-Beenden bringt irgendwann die neue Version. Der Update-Banner zeigt sich in der Praxis selten.

**Wurzelursache (gemeinsam für beide Bugs):** Der Service Worker fängt **alle** `/api/*`-Requests mit einer `NetworkFirst`-Strategie (mit 10s-Timeout) ab — inklusive der long-lived SSE-Endpoints `/api/events` und `/api/chat/events`. Workbox' `NetworkFirst` ist für long-lived `text/event-stream`-Verbindungen semantisch ungeeignet: der Response-Klon-Versuch fürs Caching, das Timeout-Fallback auf Cache und das mögliche Ausliefern gecachter Stream-Fragmente führen zu unzuverlässigem SSE-Verhalten. Konkret:

- Nach einem Server-Neustart (Deploy) trifft das Reconnect des Browsers häufig in die 10s-Restart-Latenz → `NetworkFirst` fällt auf den (für SSE sinnlosen) Cache zurück → kein `__version:`-Re-Send → kein Banner, keine Version.
- Bei sporadischen Fetch-Cache-Treffern kann der gecachte alte `__version:`-Frame ausgeliefert werden → der Vergleich `knownVersion === current` schlägt fälschlich „passt" zurück.
- Browser-/PWA-Restart räumt den SW-State auf — danach läuft eine frische SSE-Verbindung sauber durch und der Update wird erst dann erkannt. Das deckt sich exakt mit dem gemeldeten Symptom.

Zusätzlich verschärfen folgende sekundäre Schwächen das Bild:

- **`useVersionCheck` ist fragil:** Effect-Dependency `[]` (kein Reconnect bei Login/Logout), `?token=…`-Query ist wirkungslos (Server gated via `CookieMiddleware`), kein `onerror`-Handler, hartes `if (import.meta.env.DEV) return`.
- **`useVersionCheck` läuft doppelt:** je ein Aufruf in `AppUpdateBanner` (App-Root, oft ohne Token) und in `AppShell` (post-Login). Zwei parallele SSE-Verbindungen, einer davon meist tot.
- **Reload-Race:** `reloadWithSwActivation` macht ein nacktes `location.reload()`, wenn `reg.waiting` zum Klick-Zeitpunkt noch null ist — der alte SW serviert weiter den Precache. Damit zeigt der „Jetzt laden"-Knopf der Anwender:in den alten Stand, obwohl die SSE eine neue Version gemeldet hatte.
- **Dismiss ist sticky:** `dismissed: boolean` resetted nie. Wer den Banner einmal wegklickt, sieht ihn für weitere Deploys in der gleichen Session nicht mehr.

## What Changes

- **Service Worker:** `/api/events` und `/api/chat/events` werden vor der `/api/*`-NetworkFirst-Regel mit `NetworkOnly` registriert. SSE-Verbindungen laufen damit garantiert ungefiltert ins Netzwerk; Reconnect und `__version:`-Re-Send funktionieren wieder zuverlässig.
- **`useVersionCheck`-Hook gehärtet:**
  - reagiert auf `user` als Dependency (Reconnect bei Login/Logout/Impersonation, analog zu `useLiveUpdates`),
  - `?token=…`-Query entfällt (war wirkungslos),
  - kein DEV-Early-Exit für `version` — in DEV wird `dev` als Versions-String geliefert, damit der Sidebar-Link auch lokal sichtbar ist,
  - `onerror` bleibt leise (EventSource auto-reconnectet); kein eigenes Close mehr im Error-Pfad.
- **Zentralisierung:** Neuer `VersionContext`/`VersionProvider`, exponiert `useVersion(): { version, updateAvailable }`. `useVersionCheck` wird vom Provider intern verwendet — die UI ruft nur noch `useVersion()` auf. Damit verschwinden die zwei parallelen SSE-Verbindungen, die heute aus den zwei `useVersionCheck`-Calls resultieren.
- **Robuster Reload-Flow:** `reloadWithSwActivation` triggert zuerst `reg.update()`, pollt bis ~5 s auf `reg.waiting`, und macht erst dann `SKIP_WAITING`+`controllerchange`+`reload`. Wenn nach Timeout immer noch kein `waiting`-SW da ist, wird der `api-cache` explizit geleert (`caches.delete('api-cache')`), bevor `location.reload()` läuft. Damit gibt es keinen Pfad mehr, bei dem der „Jetzt laden"-Klick den alten Precache-Stand zeigt.
- **Banner-Dismiss versionsbezogen:** `dismissed` wird nicht mehr als boolean, sondern als Versions-String (oder `null`) gehalten. Eine neue erkannte Version macht den Banner wieder sichtbar, auch wenn der vorherige weggeklickt war.

## Capabilities

### New Capabilities

_Keine — die Änderung modifiziert ausschließlich bestehende Capabilities._

### Modified Capabilities

- `deploy-version-detection`: `useVersionCheck` reagiert auf `user`, hat kein wirkungsloses `?token=`, hat in DEV einen sichtbaren `version`-Wert; Dismiss wird versionsbezogen.
- `pwa-support`: SSE-Endpoints (`/api/events`, `/api/chat/events`) sind vom `/api/*`-NetworkFirst-Caching ausgenommen und laufen `NetworkOnly`. Der „Update verfügbar"-Reload-Flow räumt vor dem Reload Caches/SW-State auf.
- `version-display`: Sidebar-Versionsanzeige ist auch im Dev-Modus sichtbar (zeigt `v dev`).

## Impact

- **Frontend:**
  - `web/src/sw.ts`: zwei `NetworkOnly`-Routes über der `/api/*`-NetworkFirst-Route.
  - `web/src/hooks/useVersionCheck.ts`: Refactor (siehe oben). Bleibt als internes Detail des Providers.
  - `web/src/contexts/VersionContext.tsx`: neue Datei (Provider + `useVersion`-Hook).
  - `web/src/App.tsx`: `<VersionProvider>` einhängen, `AppUpdateBanner` konsumiert `useVersion()` + `dismissed`-State versionsbezogen.
  - `web/src/components/AppShell.tsx`: nutzt `useVersion()` statt direktem `useVersionCheck()`.
  - `web/src/lib/reload.ts`: Polling auf `reg.waiting`, Cache-Clear-Fallback.
- **Backend:** Keine Änderungen.
- **Datenbank:** Keine Migration nötig.
- **Tests:**
  - Hub-Backend bleibt unverändert (Tests in `internal/hub/handler_test.go` weiterhin grün).
  - Frontend-Unit-Tests bestehen nicht in dieser Form — Verifikation läuft über die existierende manuelle PWA-Prüfung (Deploy → Browser-Tab offen lassen → Banner muss < 30 s erscheinen → Reload zeigt neuen Stand).
- **Migrationspfad für Nutzer:innen mit altem SW:** Beim ersten Besuch nach Deploy holt sich der Browser den neuen `sw.js`. Sobald dieser aktiv ist, sind die SSE-Endpoints wieder NetworkOnly. Anwender:innen, die aktuell „festhängen", brauchen einmalig einen Hard-Reload (Browser schließen oder DevTools-„Application → Service Workers → Unregister"). Das ist die gleiche Lösung, die schon heute bei diesem Bug funktioniert — wir empfehlen sie als Hinweis im Release-Changelog.
