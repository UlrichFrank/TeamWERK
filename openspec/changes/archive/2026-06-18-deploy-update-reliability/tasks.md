## 1. Backend: Cache-Header und ETag im spaHandler

- [x] 1.1 In `cmd/teamwerk/main.go` Regex `hashedAssetRegex = regexp.MustCompile(\`^assets/[^/]+-[A-Za-z0-9_-]{8,}\.[a-z0-9]+$\`)` als Paket-Variable anlegen.
- [x] 1.2 `spaHandler` so erweitern, dass vor `fileServer.ServeHTTP` der `Cache-Control`-Header gesetzt wird: bei hashed Asset `public, max-age=31536000, immutable`, sonst `no-cache, must-revalidate`.
- [x] 1.3 Für nicht-hashed Pfade ETag setzen: `fmt.Sprintf(`"%s-%x"`, buildHash, sha256.Sum256([]byte(path))[:4])` (deterministisch, 4-Byte-Suffix als Hex).
- [x] 1.4 Vor dem Weiterreichen an `fileServer.ServeHTTP` `If-None-Match` prüfen — bei Treffer `w.WriteHeader(http.StatusNotModified)` und Return.
- [x] 1.5 Sicherstellen, dass die Header-Logik auch im SPA-Fallback-Zweig (Pfad nicht in FS → liefere index.html) greift; dort gilt die "alles außer hashed" -Regel mit ETag für `index.html`.
- [x] 1.6 In `cmd/teamwerk/main_test.go` (ggf. neu) Tabellen-Test `TestSpaHandler_CacheHeaders` schreiben: `/`, `/index.html`, `/sw.js`, `/manifest.webmanifest`, `/assets/index-AbCd1234.js`, `/icons/icon-192.png`.
- [x] 1.7 Test `TestSpaHandler_ETag_304` schreiben: erst GET `/index.html` → ETag aus Response auslesen, dann erneuter GET mit `If-None-Match` → Status 304, leerer Body.
- [x] 1.8 Test `TestSpaHandler_ETag_Changes_With_BuildHash` schreiben: ETag bei `buildHash="dev"` vs. `buildHash="x123"` muss sich unterscheiden (Helper, der die Paket-Variable testweise umschreibt, oder Refactor zu Handler-Param).
- [x] 1.9 `go test ./cmd/teamwerk/...` läuft grün.

## 2. Frontend: SW-Navigation auf NetworkFirst, index.html aus Precache nehmen

- [x] 2.1 In `web/vite.config.ts` `injectManifest.globPatterns: ['**/*.{js,css,ico,png,svg,woff2}']` setzen (kein `.html`).
- [x] 2.2 `pnpm build` lokal ausführen und im generierten SW-Bundle (Search nach `__WB_MANIFEST`) verifizieren, dass keine `.html`-Datei mehr im Precache-Manifest auftaucht. Notfalls über `injectManifest.manifestTransforms`-Hook filtern.
- [x] 2.3 In `web/src/sw.ts` neue Route VOR allen anderen Routen einfügen: `registerRoute(({ request }) => request.mode === 'navigate', new NetworkFirst({ cacheName: 'app-shell', networkTimeoutSeconds: 3, plugins: [new ExpirationPlugin({ maxEntries: 1, maxAgeSeconds: 60 * 60 * 24 * 30 })] }))`.
- [x] 2.4 Sicherstellen, dass die bestehenden Routen in dieser Reihenfolge bleiben: Navigation (neu) → Google Fonts CSS → Google Fonts static → `/api/auth/*` NetworkOnly → `/api/events` + `/api/chat/events` NetworkOnly → `/api/*` NetworkFirst.
- [x] 2.5 Verifizieren, dass der `push`-Event-Listener, der `message`-Listener (SKIP_WAITING) und der `notificationclick`-Listener unverändert im SW bleiben.

## 3. Frontend: Reload-Fallback härten

- [x] 3.1 In `web/src/lib/reload.ts` Konstante `APP_SHELL_CACHE_NAME = 'app-shell'` und `WORKBOX_PRECACHE_PREFIX = 'workbox-precache'` ergänzen.
- [x] 3.2 Im Fallback-Pfad (kein `waiting` nach `waitForWaiting`) bisherigen `caches.delete('api-cache')`-Aufruf ersetzen durch: `const keys = await caches.keys(); await Promise.all(keys.filter(n => n === API_CACHE_NAME || n === APP_SHELL_CACHE_NAME || n.startsWith(WORKBOX_PRECACHE_PREFIX)).map(n => caches.delete(n)))`.
- [x] 3.3 Fehlerfall (Caches-API unavailable, Browser ohne SW) muss weiterhin `location.reload()` ausführen — try/catch um den Cleanup, nicht um den Reload.
- [x] 3.4 Optionalen Vitest in `web/src/lib/reload.test.ts` schreiben, der `caches`-API mockt und prüft, dass genau die drei Cache-Namen gelöscht werden und `google-fonts-cache` unangetastet bleibt.

## 4. Lokale Smoke-Tests

- [x] 4.1 `pnpm --filter web build` + `go build -o bin/teamwerk ./cmd/teamwerk` ohne Fehler.
- [x] 4.2 `bin/teamwerk` lokal starten. `curl -I http://localhost:8080/` → `Cache-Control: no-cache, must-revalidate`, `ETag` vorhanden.
- [x] 4.3 `curl -I http://localhost:8080/sw.js` → `Cache-Control: no-cache, must-revalidate`, `ETag` vorhanden.
- [x] 4.4 Beispielhafte Asset-URL aus `web/dist/assets/` raussuchen, `curl -I http://localhost:8080/assets/<datei>` → `Cache-Control: public, max-age=31536000, immutable`.
- [x] 4.5 `curl -I -H 'If-None-Match: <ETag-von-4.2>' http://localhost:8080/` → Status `304`.
- [ ] 4.6 Im Chrome DevTools Application-Tab nach Build-Reload prüfen: `app-shell`-Cache existiert nach erster Navigation; `workbox-precache`-Cache enthält JS/CSS aber keine `.html`-Datei.

## 5. Deploy und Verifikation

- [x] 5.1 Branch öffnen, OpenSpec-Proposal-Datei committen (Conventional Commits, z. B. `chore(openspec): proposal deploy-update-reliability`).
- [x] 5.2 Implementierungs-Commits pro Task-Block (Backend / SW / Reload / Tests) — siehe CLAUDE.md OpenSpec-Workflow.
- [ ] 5.3 `make deploy` ausführen.
- [ ] 5.4 Direkt nach Deploy auf Produktion: `curl -I https://intern.team-stuttgart.org/` → erwartete Header.
- [ ] 5.5 Manuelle iOS-PWA-Verifikation (Vorstand-Tester): App schließen, öffnen → neue Version OHNE Banner-Klick im Sidebar-Footer sichtbar.
- [ ] 5.6 Manuelle Browser-Verifikation (Chrome Desktop): Mit alter Version offen lassen, deploy → Banner kommt → "Jetzt laden" → neuer Hash sichtbar.
- [ ] 5.7 Offline-Probe (DevTools → Offline) nach erfolgreichem Erst-Load: App startet aus `app-shell`-Cache, zeigt entweder gewohnte Shell oder Offline-Hinweis.

## 6. Archivierung

- [x] 6.1 Nach erfolgreicher Produktions-Verifikation `/opsx:archive deploy-update-reliability` ausführen (oder manuell archivieren).
- [ ] 6.2 Memory-Eintrag für künftige Sessions prüfen — gegebenenfalls Hinweis "iOS-PWA-Update läuft via NetworkFirst-Shell, kein Logout-Workaround mehr nötig" notieren.
