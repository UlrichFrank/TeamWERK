## Why

Der `chat-open-at-unread`-Bug (Scroll landet nach Bild-Loads 24–358 px vor
dem Ende) war JS-korrekt und alle 560 Vitest-Tests grün. Erst live in
Chrome (via DevTools MCP) sichtbar: Chrome-spezifisches Verhalten bei
Image-Decode, aspect-ratio-Anwendung und Scroll-Physik, das jsdom nicht
simuliert. Die aktuelle Test-Landschaft hat damit eine harte Decke —
Browser-Verhalten kann sie prinzipiell nicht prüfen.

Beispiele, die jsdom auch künftig nicht catchen wird:
- Scroll-Positionen nach asynchronem Content-Wachstum
- `<img>`-Decode-Timing und aspect-ratio-Anwendung
- `requestAnimationFrame`-Sequenzen
- Smooth-Scroll-Physik / Abbruchverhalten
- `IntersectionObserver` / `ResizeObserver` mit echtem Layout
- Focus-Management (`:focus-visible`, tabindex-Reihenfolge)
- Touch- / Pointer-Events

Ohne echten Browser fehlt diese Test-Ebene komplett.

## What Changes

- Neue Test-Schicht mit **Playwright** (headless Chromium) als
  devDependency in `web/`, zusätzlich zur bestehenden Vitest-Suite.
  Bestehende Vitest-Tests bleiben unverändert.
- Neues Verzeichnis `web/e2e/` für Playwright-Test-Files mit eigener
  `playwright.config.ts`. Tests laufen gegen den `pnpm dev`-Server
  (Vite auf `:5173`) und ein separates Go-Backend, das gegen eine
  deterministische Test-DB gestartet wird.
- **Test-DB-Seed**: kleines Go-Programm (`cmd/e2e-seed/`) baut eine
  frische SQLite-DB mit einem admin-User, einer Chat-Konv mit Bildern
  und einer Konv mit ungelesenen Nachrichten. Läuft vor jeder E2E-Suite.
- Neuer Make-Target `make test-e2e` startet: Backend mit Seed-DB →
  Vite → Playwright-Runner → Cleanup. `make test` bleibt reine
  Unit-Ebene und schnell (~34 s).
- Neuer CI-Job `e2e` in GitHub Actions (separater Job neben `gate`),
  läuft parallel zu `make test`. Bricht den PR-Merge, wenn Playwright
  fehlschlägt — bleibt aber außerhalb des lokalen `pre-push`-Hooks
  (Playwright-Runs dauern zu lang für jeden Push).
- **Initial 3–5 Tests**, klein starten:
  1. Chat: gelesene Konv mit Bildern öffnen → `scrollTop ≈ max` nach
     Bild-Loads (Regression für den chat-open-at-unread-Bug).
  2. Chat: Konv mit Ungelesenem öffnen → `UnreadDivider` sichtbar im
     Viewport, stabil auch nach Bild-Loads.
  3. Chat: Deep-Link `?openUser=<id>` → landet am Ende/Divider.
  4. Login-Flow: Login → Redirect nach `/chat` → Konv-Liste sichtbar.
  5. Nachricht senden → sichtbar in eigener Bubble (SSE-Zyklus komplett).
- **Konvention** ergänzen in `docs/agent/07-testing.md`: bei UI-Änderungen
  an Scroll/Layout/Animation/Focus-Verhalten → Playwright-Test in
  Erwägung ziehen (nicht Pflicht, aber Reminder). Chrome DevTools MCP
  bleibt als Live-Diagnose-Werkzeug etabliert.

Kein BREAKING. Vitest-Suite unverändert. Playwright ist zusätzlich.

## Capabilities

### New Capabilities

- `frontend-e2e-tests`: Playwright-basierte E2E-Test-Schicht mit
  Test-DB-Seed, Make-Target und CI-Job.

### Modified Capabilities

- `test-infrastructure`: Requirement für den `e2e-seed`-CLI-Befehl
  und die Konvention „Browser-Verhalten gehört in E2E, JS-Logik in
  Vitest" ergänzen.

## Impact

- **Neue Dependencies**:
  - `@playwright/test` als `web/`-devDependency (~150 MB)
  - Chromium-Binary (~200 MB) — nur beim Setup / im CI-Container
- **Backend**: Neuer CLI-Subcommand `cmd/teamwerk` … `e2e-seed --db=…`
  ODER separater `cmd/e2e-seed/`-Entrypoint. Setzt eine deterministische
  DB auf: 1 admin, 3 Test-User, 2 Konvs (eine gelesen mit Bildern, eine
  mit 3 unreads).
- **CI**: `make test-e2e`-Target startet Backend + Vite und triggert
  Playwright. Neuer GH-Actions-Job `e2e` läuft parallel zum `gate`-Job;
  ~2–4 min zusätzliche CI-Zeit erwartet.
- **Lokal**: Entwickler muss einmalig `pnpm exec playwright install
  chromium` ausführen. `make init` erledigt das automatisch.
- **Kein Prod-Code betroffen**. Nur Test-Infra.
- **Openspec-Konvention**: neue UI-Features mit Browser-Verhalten
  bekommen einen Playwright-Test (nicht harte Pflicht, aber im
  `07-testing.md` als Empfehlung).
