## Status (2026-07-18) — Fundament steht & verifiziert

**Umgesetzt & real grün** (echtes Chromium, Prod-Binary + Seed-DB, `2 passed`):
- Playwright installiert (`@playwright/test` devDep + Chromium-Binary). [1.1, 1.2]
- `web/e2e/`: `playwright.config.ts` (Single-Origin, `webServer` = `go build` → `e2e-seed` → serve :18080), `fixtures.ts` (`loginAsAdmin`), `tsconfig.json`. [1.4, 1.5]
- `.gitignore`-Einträge. [1.6]
- Go-Subcommand `e2e-seed` (`cmd/teamwerk/e2e_seed.go`): migrate + Admin + 3 Standard-User + Chat-Konversation (Text). [2.1, 2.2, 2.3 (Text-Teil)]
- `make test-e2e` + `web/package.json` `test:e2e`. [3.1, 3.2, 3.4]
- Tests: `auth-login.spec.ts` (Login-Golden-Path + Fehlerpfad). [4.1, 4.5]
- Doku `07-testing.md` „E2E — wann statt Vitest". [6.1 (Kern)]

**Noch offen (auf dem laufenden Fundament trivial nachrüstbar):**
- Seed um **Bild-Nachrichten** erweitern (deterministische PNGs) [2.4–2.6] — Voraussetzung für die chat-scroll-Killer-Cases.
- chat-scroll-Tests [4.2–4.4] (der ursprüngliche Bug-Vector: scrollTop nach Bild-Decode), chat-send [4.6], Dienstbörse-Golden-Path (Roadmap 7.2), Bank-Envelope-Zero-Knowledge (Roadmap 7.3).
- CI-Job `e2e` [5.x], `make init`-Schritt [1.3], Restdoku [6.2–6.5].

**Hinweis lokale Ausführung:** `make test-e2e` nutzt `pnpm build`; auf Maschinen mit zwei pnpm-Versionen (Homebrew 11 + Corepack/Standalone 10) muss die pnpm die zum Lockfile passt (v10) auf dem PATH vorn stehen — sonst bricht `pnpm` mit `ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY` ab. In CI (eine pnpm) unkritisch.

## 1. Playwright-Setup

- [ ] 1.1 `pnpm -C web add -D @playwright/test` (neue devDependency, ~150 MB)
- [ ] 1.2 `pnpm -C web exec playwright install chromium` (Chromium-Binary, ~200 MB — nur beim Setup)
- [ ] 1.3 In `make init` einen Schritt „Playwright-Chromium installieren" hinzufügen (silent skip wenn schon da)
- [ ] 1.4 `web/e2e/`-Verzeichnis anlegen mit `playwright.config.ts` (Chromium, headless CI/headed lokal via `PWDEBUG`, `webServer`-Config startet Backend :18080 + Vite :15173, `testDir: './'`, `reporter: 'list'`, `retries: process.env.CI ? 2 : 0`)
- [ ] 1.5 Separates `web/e2e/tsconfig.json` das nicht auf `src/` importiert (Playwright-Runner soll nicht mit App-Interna vermischen)
- [ ] 1.6 `.gitignore`: `web/e2e/.playwright/`, `web/e2e/test-results/`, `web/e2e/*.db` ergänzen

## 2. Test-DB-Seed

- [ ] 2.1 Neuen Subcommand `e2e-seed` in `cmd/teamwerk/main.go` registrieren: `--db=<path>` verpflichtend, überschreibt existierende Datei
- [ ] 2.2 Seed-Logik in neuem File `cmd/teamwerk/e2e_seed.go` (nutzt `internal/testutil`-Fixtures wo möglich, sonst direktes `database/sql`)
- [ ] 2.3 Datensatz: 1 Admin `e2e@test.local` / `E2ETestPassword!` (via bestehendes bcrypt-Hashing), 3 Standard-User, Gruppe „E2E Chat mit Bildern" (10 Text + 3 Bild, alle gelesen für Admin), Gruppe „E2E Chat unread" (20 Text, letzte 3 ungelesen für Admin)
- [ ] 2.4 Bilder für die 3 Bild-Nachrichten: kleine deterministische PNGs (z.B. 100x100 solid color) direkt im Go-Code generieren + über `internal/media`-Store-Pfad ablegen (falls MEDIA_DIR nicht existiert, anlegen)
- [ ] 2.5 Go-Test `TestE2ESeed_Idempotent`: zwei Läufe mit demselben Pfad ergeben denselben DB-Zustand (dumpvergleich oder Row-Counts)
- [ ] 2.6 Go-Test `TestE2ESeed_LoginWorks`: nach Seed ist Login mit Admin-Credentials über den bestehenden `auth`-Handler erfolgreich

## 3. Make-Target + Skripte

- [ ] 3.1 `web/package.json`: `"test:e2e": "playwright test --config e2e/playwright.config.ts"` ergänzen
- [ ] 3.2 `Makefile`: neuer Target `test-e2e` — baut Backend + Frontend, seedet DB, ruft `pnpm -C web test:e2e`, räumt nach Abschluss `e2e.db` auf (via `trap` in shell wenn nötig)
- [ ] 3.3 `Makefile`: `help`-Text für `test-e2e` mit Warnung „braucht 2–4 min, nicht Teil von `make test`"
- [ ] 3.4 `.PHONY` in Makefile um `test-e2e` erweitern

## 4. E2E-Tests (5 initial)

- [ ] 4.1 `web/e2e/fixtures.ts`: gemeinsamer `loginAsAdmin(page)`-Helper (füllt Login-Formular, klickt Submit, wartet auf `/chat`-URL)
- [ ] 4.2 `web/e2e/chat-scroll.spec.ts` — Test 1: „gelesene Konv mit Bildern → scrollTop ≈ scrollHeight-clientHeight nach Bild-Loads" (`page.waitForFunction(() => alle img.complete)`, dann distance messen, `expect(dist).toBeLessThan(5)`)
- [ ] 4.3 `web/e2e/chat-scroll.spec.ts` — Test 2: „unread Konv → Divider sichtbar im Viewport nach Bild-Loads" (`page.getByText(/3 ungelesene Nachrichten/).isIntersectingViewport()`)
- [ ] 4.4 `web/e2e/chat-scroll.spec.ts` — Test 3: „Deep-Link `?openUser=<id>` → scrollTop > 0" (Direktnachricht mit einem Standard-User; Setup fürs Vorhandensein einer Konv-mit-Verlauf im Seed)
- [ ] 4.5 `web/e2e/auth-login.spec.ts` — Test 4: „Login → /chat lädt" (Login-Form ausfüllen, warten auf `/chat`, Konv-Liste sichtbar)
- [ ] 4.6 `web/e2e/chat-send.spec.ts` — Test 5: „Nachricht senden erscheint in Bubble" (UUID-Marker, `expect(page.getByText(marker)).toBeVisible({ timeout: 3000 })`)

## 5. CI-Integration

- [ ] 5.1 Neuer Job `e2e` in `.github/workflows/ci.yml`: `runs-on: ubuntu-latest`, setup Node + Go + pnpm, `pnpm install`, `pnpm exec playwright install --with-deps chromium` (mit cache), `make test-e2e`
- [ ] 5.2 GH-Actions-Cache für Chromium-Binary (`~/.cache/ms-playwright`) — spart nach dem ersten Lauf ~200 MB Download
- [ ] 5.3 `e2e`-Job läuft parallel zu `gate` (kein `needs`)
- [ ] 5.4 Branch-Protection auf `main`: `e2e` als required check aktivieren (nach 1–2 Wochen Beobachtung; initial non-blocking)
- [ ] 5.5 Bei Test-Failure: `playwright-report/` als Actions-Artifact hochladen (`actions/upload-artifact@v4` mit `if: failure()`)

## 6. Dokumentation

- [ ] 6.1 `docs/agent/07-testing.md`: Abschnitt „Wann Vitest, wann Playwright" mit klarem Kriterienkatalog (Vitest = JS-Logik, Playwright = Browser-Verhalten), Beispiel-Bug (`chat-open-at-unread`) als Anker
- [ ] 6.2 `docs/agent/07-testing.md`: Reminder bei UI-Änderungen an Scroll/Layout/Animation/Focus → E2E-Test in Erwägung ziehen (nicht harte Pflicht)
- [ ] 6.3 `docs/agent/07-testing.md`: Kommando-Referenz `make test-e2e`, `make test` (was macht was) plus `pnpm -C web exec playwright test --debug` für lokales Debugging
- [ ] 6.4 `docs/agent/02-workflow.md`: `make test-e2e` in die Kommandoliste aufnehmen mit Vermerk „~2–4 min, für UI-riskante Änderungen"
- [ ] 6.5 `CLAUDE.md` Hard Rules: Chrome DevTools MCP als Diagnose-Werkzeug offiziell empfehlen (nicht Pflicht, aber schnellster Weg für Live-Bug-Analyse — Referenz auf den `chat-open-at-unread`-Bug als Beispiel)

## 7. Verifikation

- [ ] 7.1 `make test-e2e` lokal grün (alle 5 Tests)
- [ ] 7.2 Bewusste Regression: den `programmaticScrollUntilRef`-Guard in `ChatPage.tsx` auskommentieren → Test 2 „Divider sichtbar" schlägt fehl (beweist, dass die Regression tatsächlich gefangen wird), danach zurückrollen
- [ ] 7.3 CI-Run auf einem Proben-PR grün, `e2e`-Job unter 5 min
- [ ] 7.4 `make test` bleibt <45 s (E2E hat es nicht verlangsamt)
- [ ] 7.5 `openspec validate frontend-e2e-tests --type change` grün
- [ ] 7.6 Manuell: einmal `pnpm -C web exec playwright test --debug` starten, dass headed-Mode + Trace-Viewer funktionieren
