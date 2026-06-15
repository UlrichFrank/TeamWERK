# Tasks

## 1. Service Worker: SSE-Routes von NetworkFirst ausnehmen

- [x] 1.1 In `web/src/sw.ts` zwei `NetworkOnly`-Routes vor der bestehenden `/api/*`-`NetworkFirst`-Route registrieren:
  - `/api/events`
  - `/api/chat/events`
- [x] 1.2 Commit: `fix(pwa): SSE-Endpoints aus NetworkFirst-Caching ausnehmen`

## 2. VersionContext einführen

- [x] 2.1 `web/src/contexts/VersionContext.tsx` anlegen:
  - `VersionProvider` umschließt die App, ruft intern einmal `useVersionCheck()` auf.
  - `useVersion(): { version: string | null, updateAvailable: boolean }` als public API.
- [x] 2.2 `web/src/App.tsx`: `<VersionProvider>` zwischen `<AuthProvider>` und `<BrowserRouter>` einhängen.
- [x] 2.3 Commit: `feat(version): VersionContext zentralisiert SSE-Versionserkennung`

## 3. useVersionCheck härten

- [x] 3.1 In `web/src/hooks/useVersionCheck.ts`:
  - Effect-Dependency `[user]` (analog zu `useLiveUpdates`).
  - `?token=…`-Query entfernen (war wirkungslos, CookieMiddleware nutzt das Cookie).
  - Im DEV-Modus: `version = 'dev'` setzen, keine SSE öffnen, `updateAvailable` bleibt `false`.
  - Kein eigenes `es.close()` im `onerror` — EventSource auto-reconnectet.
- [x] 3.2 Sicherstellen, dass der Hook beim Logout (`user === null`) die SSE schließt und `version` auf `null` zurücksetzt.
- [x] 3.3 Commit: `fix(version): Hook reagiert auf user, DEV zeigt v dev, ?token entfernt`

## 4. Konsumenten auf useVersion umstellen

- [x] 4.1 `web/src/components/AppShell.tsx`: `useVersionCheck()` durch `useVersion()` ersetzen.
- [x] 4.2 `web/src/App.tsx` (`AppUpdateBanner`): `useVersionCheck()` durch `useVersion()` ersetzen.
- [x] 4.3 Commit: `refactor(version): AppShell und Banner konsumieren useVersion-Context`

## 5. Reload-Flow härten

- [x] 5.1 In `web/src/lib/reload.ts` `reloadWithSwActivation` umbauen:
  - Wenn `reg.waiting` direkt da → bisheriger Pfad.
  - Sonst: `await reg.update()`, dann bis ~5 s auf `reg.waiting` pollen (z. B. via `setInterval` 250 ms + Promise).
  - Wenn nach Timeout immer noch kein `waiting`: `await caches.delete('api-cache')`, dann `location.reload()`.
- [x] 5.2 Commit: `fix(pwa): Reload wartet auf neuen SW und leert ggf. api-cache`

## 6. Banner-Dismiss versionsbezogen

- [x] 6.1 In `AppUpdateBanner` (`web/src/App.tsx`):
  - `dismissed: boolean` → `dismissedVersion: string | null`.
  - Banner sichtbar wenn `(sseUpdateAvailable || swUpdateAvailable) && dismissedVersion !== version`.
  - Beim Dismiss: aktuelle `version` als `dismissedVersion` speichern.
  - Erweiterung: `useVersionCheck` exposed zusätzlich `latestVersion` (zuletzt vom Server gesehener Hash) — Banner-Dismiss matcht darauf statt auf der App-eigenen `version`, sodass ein zweiter Deploy den Banner zuverlässig wieder zeigt.
- [x] 6.2 Commit: `fix(version): Dismiss bezieht sich auf konkrete Version`

## 7. Verifikation & Dokumentation

- [ ] 7.1 Lokal verifizieren:
  - `pnpm dev` → Sidebar zeigt `v dev`, Link öffnet `ChangelogModal`.
  - `make build && go run ./cmd/teamwerk` → Sidebar zeigt echten Git-Hash.
- [ ] 7.2 Deploy-Smoke auf VPS:
  - Tab vor `make deploy` offen halten → Banner muss innerhalb ~10 s erscheinen.
  - „Jetzt laden" → Versions-Link zeigt neuen Hash.
- [ ] 7.3 Dismiss-Roundtrip: Banner dismissen, zweiten Deploy auslösen → Banner erscheint erneut.
- [ ] 7.4 Im Changelog/Release-Note vermerken: Anwender:innen, die aktuell „hängen", brauchen einmalig Tab schließen + neu öffnen, damit der neue SW aktiv wird.
- [ ] 7.5 Commit: `docs(changelog): Hinweis zur einmaligen Update-Prozedur`

## 8. OpenSpec archivieren

- [ ] 8.1 Nach Implementierung und Smoke: `/openspec-archive-change versionserkennung-haerten`.
