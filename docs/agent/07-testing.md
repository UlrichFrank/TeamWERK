# Test-Standard

**Wo investieren?** Die Priorisierung (Risiko × Churn vor Coverage-Prozent, Arch-Test vor Copy-Paste, Bug-Fix vor Charakterisierung, E2E vor Vitest-Coverage) ist als Capability `test-strategy` festgehalten: `openspec/specs/test-strategy/spec.md`.

Jede neue HTTP-Route **muss** mindestens **Happy-Path** (Erfolg) und **Fehlerfall** (401/403/400/404/409) abdecken. Tests prüfen fachliche Invarianten — keine Dummy-Assertions zur Coverage-Erhöhung.

Jeder OpenSpec-Proposal mit neuen Routen / geänderter Geschäftslogik braucht einen Abschnitt `## Test-Anforderungen` (Route → Testname + erwarteter Status, plus die garantierte Invariante).

**Fixtures** in `internal/testutil/`: `NewDB`, `NewServer`, `CreateUser`, `CreateMember`, `CreateSeason`, `CreateTeam`, `CreateGame`, `CreateKader`, `CreateDutyType`, `CreateDutySlot`, `CreateInvitationToken`, `CreatePasswordResetToken`, `CreateRefreshToken`.

`make coverage` → stdout + HTML nach `/tmp/teamwerk-coverage.html`. Coverage ist Indikator, kein Gate.

`make metrics` → erhebt Größe/Komplexität/Coverage/Lint-Dichte/Duplikation, schreibt `metrics/REPORT.md` (gitignored). Komplexität nutzt **separate** `.golangci.metrics.yml` (`gocyclo`, `gocognit`, `funlen`, `dupl`) — die Haupt-`.golangci.yml` (Gate) bleibt unangetastet. Tools sind im jeweiligen Manifest gepinnt: `scc` als `go.mod`-`tool`-Direktive (`go tool scc`), `jscpd` als pnpm-devDependency (`pnpm -C web exec jscpd`). `make metrics-gate` vergleicht zusätzlich gegen `metrics/thresholds.yml` (Ratchet-Prinzip; Exit 1 bei Regression).

## E2E-Tests (Playwright) — wann statt Vitest

Drei Test-Ebenen: **Go** (Backend-Contracts, HTTP+DB) · **Vitest+jsdom** (JS-Logik, primäre Frontend-Ebene) · **Playwright** (echter Chromium, echtes Layout/Decode/Scroll). Kriterium:

- **Vitest** = reine JS-Logik, Rendering, Komponenten ohne Layout-Physik. Bleibt die primäre, schnelle Ebene.
- **Playwright** = **Browser-Verhalten**, das jsdom strukturell nicht kann: Scroll-Position nach Bild-Decode, Layout/`aspect-ratio`, Focus, echte Navigation über Seiten. Klein halten (wenige Killer-Cases), keine flächendeckende UI-Coverage.

Bei UI-Änderungen an Scroll/Layout/Animation/Focus → E2E-Test erwägen (keine harte Pflicht).

**Setup** (`web/e2e/`): Single-Origin — die Prod-Binary liefert die eingebettete SPA + API auf einem Port. `playwright.config.ts` startet via `webServer` die Test-Binary (`go build` → `e2e-seed` → serve auf :18080). Deterministische Seed-DB via Go-Subcommand `teamwerk e2e-seed --db=<pfad>` (1 Admin `e2e@test.local`/`E2ETestPassword!`, 3 Standard-User, 1 Chat-Konversation). Kein Vite-Proxy, keine geteilte Dev-DB.

**Kommandos:**
- `make test-e2e` — Frontend bauen + Playwright (echter Chromium). ~2–4 min, **nicht** Teil von `make test`/pre-push.
- `pnpm -C web exec playwright test --config e2e/playwright.config.ts --debug` — lokales Debugging (headed + Trace-Viewer).
- Einmalig: `pnpm -C web add -D @playwright/test && pnpm -C web exec playwright install chromium`.
