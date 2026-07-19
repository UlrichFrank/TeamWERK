## Context

Aktuelle Test-Landschaft (Stand nach `chat-open-at-unread`):

```
      ┌────────────────────────┐
      │  Vitest + jsdom  560   │  ~34 s, JS-Logik, keine Layout-Physik
      └────────────────────────┘
      ┌────────────────────────┐
      │  Go-Tests        alle  │  Backend-Contracts, HTTP + DB
      └────────────────────────┘

Neue Ebene:

      ┌────────────────────────┐
      │  Playwright   3-5 Tests│  echter Chromium, echtes Layout,
      │                        │  echte Bild-Decodes
      └────────────────────────┘
```

Der Bug-Trigger im Detail (Beweis dass jsdom das strukturell nicht
lösen kann): `<img>` mit `aspect-ratio`-Style wendet die berechnete
Höhe erst nach dem Image-Decode auf's Layout an. Das ist eine
Browser-interne Layout-Anpassung, kein DOM-Change, kein Style-Change,
kein scroll-Event. jsdom hat weder Layout noch Decode-Pipeline.

Playwright ist das für Web etablierte Framework (Microsoft, MIT
Lizenz, mature ~5 Jahre). Alternativen (Cypress, WebdriverIO,
Puppeteer direkt) haben jeweils ähnlichen Aufwand; Playwright hat
die beste TypeScript-Integration, Auto-Wait, und ist bei React-Apps
Community-Standard.

Aus `docs/agent/*.md` relevante Constraints:
- VPS 1 GB RAM — E2E-Tests laufen NICHT auf dem VPS, nur in CI und
  lokal. Kein Impact auf Produktion.
- `make test` schnell halten — E2E kommt in `make test-e2e` (separat).
- Kein Duplikat der Vitest-Coverage — E2E prüft **Browser-Verhalten**,
  nicht JS-Logik.
- Konvention „Ein Package pro Domäne" gilt für Backend; E2E-Suite ist
  ein flaches `web/e2e/`-Verzeichnis.

## Goals / Non-Goals

**Goals:**
- E2E-Test-Setup, das Browser-abhängige Bugs wie `chat-open-at-unread`
  gefangen hätte (Regressionstest inklusive).
- Deterministischer Test-Run: gleiche DB-Fixture, gleicher Zustand,
  keine Flakes.
- Klein starten: 3–5 Tests, jeder ein Killer-Case. Suite wächst
  organisch mit weiteren Bugs.
- CI-integriert: PR-blockierend bei Rot, aber in eigenem Job (schnelles
  Vitest-Feedback bleibt).
- Lokale Ausführbarkeit für Entwickler-Debugging (headed mode).

**Non-Goals:**
- **Keine flächendeckende UI-Coverage.** E2E ist teuer; die 80/20-Regel
  gilt. Vitest bleibt die primäre Test-Ebene.
- **Keine Visual Regression Tests** (Screenshots + Diff). Fügt Fragilität
  hinzu ohne den vorliegenden Bug-Typ zu catchen. Kann später als
  separater Change kommen.
- **Kein Pre-Push-Hook für E2E.** Playwright-Runs dauern zu lang;
  Entwickler sollen den lokalen Push nicht verzögern.
- **Keine Test-DB-Isolation pro Test.** Alle Tests einer Suite laufen
  gegen dieselbe Seed-DB. Reihenfolge egal (Read-only in den meisten
  Fällen; wenn nötig, per-test-cleanup via API).
- **Kein neuer Backend-Test-Modus.** Backend startet mit normaler
  Prod-Binary gegen Seed-DB. Keine In-Test-Mock-Endpoints.
- **Kein Ersatz für Chrome DevTools MCP als Diagnose-Werkzeug.** DevTools
  MCP bleibt für Live-Investigation; Playwright ist für Regressionsschutz.

## Decisions

### Decision 1: Playwright statt Cypress / WebdriverIO / Puppeteer

**Was**: `@playwright/test` als devDependency, Chromium als Ziel-Browser
(headless CI, headed lokal für Debug).

**Warum**:
- TypeScript-Integration ohne Zusatz-Setup.
- Auto-Wait: kein manuelles `sleep()` nötig, reduziert Flakes.
- Multi-Browser einfach (falls je nötig — für uns aktuell nur Chromium,
  weil das der Bug-Vector war).
- Aktiv gepflegt, große Community, gute DevX (Trace-Viewer, Codegen).

**Alternativen**:
- Cypress: schwerer, eigener Runner, weniger flexible Assertions;
  historisch flaky. Kein Gewinn für uns.
- Puppeteer direkt: Playwright ist der Nachfolger-in-Spirit; mehr Boilerplate.
- WebdriverIO: mehr Konfig, mehr Boilerplate; kein Vorteil hier.

### Decision 2: Test-DB via CLI-Subcommand `teamwerk e2e-seed`

**Was**: Neuer Subcommand im bestehenden `cmd/teamwerk`-Binary:
`teamwerk e2e-seed --db=<path>`. Legt eine deterministische DB an mit
- Migrations up
- 1 Admin: `e2e@test.local` / `E2ETestPassword!`
- 3 Test-User (Standard)
- 1 Gruppe „E2E Chat mit Bildern" (10 Text-Nachrichten + 3 Bild-Nachrichten,
  alle gelesen für den Admin)
- 1 Gruppe „E2E Chat unread" (20 Text-Nachrichten, letzte 3 ungelesen für Admin)

**Warum**:
- Deterministisch: keine Zufalls-Daten, Tests immer gegen bekannten Zustand.
- Nutzt bestehende `internal/testutil`-Fixtures wo möglich (via Import).
- Kein SQL-Wall in einem Skript — Go-Code ist typisiert und refactor-safe.
- Ein Binary weniger als separates `cmd/e2e-seed/`.

**Alternativen**:
- SQL-Datei mit `INSERT`-Statements: fragile bei Schema-Änderungen;
  duplicate Business-Logic (Password-Hash, JWT-Signatur usw. schwer).
- Bestandsauf-DB kopieren als Fixture: nicht deterministisch, verletzt
  „Tests dürfen nichts über Produktionsdaten wissen".
- Test-nur-Endpoint in der Prod-Binary: Sicherheitsrisiko wenn versehentlich
  produktiv.

### Decision 3: Playwright startet Backend + Vite als globalSetup

**Was**: `playwright.config.ts` mit `webServer`-Config:
```ts
webServer: [
  { command: 'go run ./cmd/teamwerk e2e-seed --db=./e2e.db && go run ./cmd/teamwerk serve --db=./e2e.db --port=18080', ... },
  { command: 'pnpm dev --port 15173', ... },
]
```
Beide Prozesse lebenslang der Test-Suite. Vite proxied `/api/*` auf `:18080`.

**Warum**:
- Playwright bringt eingebaute `webServer`-Orchestrierung mit.
- Separate Ports (`:18080`, `:15173`) kollidieren nicht mit dem Entwickler-
  Dev-Server (`:8080`, `:5173`).
- DB als lokale Datei — nach Test-Run löschbar; kein In-Memory-Modus, damit
  auch Server-Restart im Test möglich (falls je nötig).

**Alternativen**:
- Backend + Vite manuell vor Playwright starten (Skript oder Prozess-
  manager): mehr Boilerplate, mehr Fehlerquellen.
- E2E gegen laufenden Dev-Server auf `:5173`: nicht-deterministisch (echte
  DB, echte Daten).
- In-Memory-SQLite: geht nicht, weil Backend + Playwright zwei getrennte
  Prozesse sind. In-Memory-DB kann nicht geteilt werden.

### Decision 4: CI-Job `e2e` parallel zum `gate`-Job, nicht im pre-push

**Was**: Neuer GH-Actions-Job `e2e` in `.github/workflows/ci.yml`:
- Installiert Node, Go, pnpm, chromium (via `playwright install`)
- Startet `make test-e2e`
- Läuft parallel zu `gate`
- `main`-Protection erweitert um `e2e` als required check

Pre-Push-Hook (`make hooks`) bleibt bei `make test` + `openspec validate`
etc. — kein E2E lokal beim Push.

**Warum**:
- Vitest-Feedback im Pre-Push bleibt <1 min.
- CI-Zeit steigt um ~2–4 min für E2E, im Hintergrund parallelisiert
  durchläuft.
- Entwickler können `make test-e2e` lokal on-demand ausführen (nicht
  verpflichtend beim Push).

**Alternativen**:
- E2E im `gate`-Job seriell: CI-Zeit steigt spürbar; bei Vitest-Failure
  läuft E2E umsonst.
- E2E nur nightly: Regressionen zu spät entdeckt (nach 24 h).
- E2E im pre-push: Entwickler-Frustration; würde häufig `--no-verify`
  benutzt werden.

### Decision 5: `web/e2e/`-Verzeichnisstruktur, nicht innerhalb `web/src/`

**Was**:
```
web/
  e2e/
    playwright.config.ts
    fixtures.ts            # gemeinsame login-Helper, page-Objects
    chat-scroll.spec.ts    # Test 1-3
    auth-login.spec.ts     # Test 4
    chat-send.spec.ts      # Test 5
    tsconfig.json          # separates tsconfig für Playwright-Runner
  src/
    ...                    # unverändert
```

**Warum**:
- Klarer Trennstrich: Vitest sucht in `src/`, Playwright in `e2e/`.
- Separates `tsconfig.json` verhindert Kreuz-Import (E2E nutzt keine
  App-Interna, sondern nur den HTTP-Layer).
- Analog zu vielen etablierten React-Setups (Next.js Docs, Remix).

**Alternativen**:
- Innerhalb `src/__e2e__/`: mischt Test-Ebenen; Vitest muss diese
  Verzeichnisse explizit exkludieren.
- Auf Repo-Root-Ebene (`/e2e/`): passt nicht zu monorepo-artigen Setups
  wo `web/` self-contained ist.

## Risks / Trade-offs

- **CI-Zeit steigt um 2–4 min.** Für den Nutzen (Bug-Klasse abgedeckt)
  akzeptabel. Falls zu langsam: nur bei UI-Changes triggern (paths-filter
  in GH-Actions) — Follow-up-Optimierung.

- **Flakiness-Risiko** (klassisches E2E-Problem). Mitigations:
  - Playwright's Auto-Wait: kein `sleep()` in Tests.
  - Test-DB immer neu geseedet.
  - Wenige Tests (3–5) → kleine Angriffsfläche.
  - `test.retry(2)` in CI als sicherheit; lokale Runs ohne Retry.

- **Playwright-Update-Aufwand.** Wie jede Dep-Update-Kadenz. Renovate/
  Dependabot picks up.

- **Chromium-Binary im CI-Container** (~200 MB Download). GH-Actions
  cached das — nach erstem Run praktisch instant. Aktuell verwenden wir
  keine benutzerdefinierten Container.

- **Test-DB als Datei am Boden statt In-Memory** — kleiner Cleanup-
  Aufwand (Datei löschen). Playwright's `webServer`-Cleanup übernimmt.

- **Entwickler-Onboarding**: neuer Setup-Schritt (`playwright install
  chromium`). Reihen wir in `make init` ein.

## Migration Plan

1. **PR erstellen**: Setup + 5 Tests + Doku in einem PR.
2. **Deploy**: keine Prod-Änderung — nur Test-Infra. Kein Rollback nötig.
3. **CI-Rollout**: `e2e`-Job initial als *non-blocking*; nach 1–2 Wochen
   Beobachtung (Flakiness?) als required check aktivieren.
4. **Doku-Update**: `docs/agent/07-testing.md` bekommt Abschnitt „Wann
   E2E vs. Vitest" mit klaren Trennungskriterien.

## Open Questions

- Keine.
