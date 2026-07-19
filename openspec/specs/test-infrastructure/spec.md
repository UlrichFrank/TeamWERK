# test-infrastructure Specification

## Purpose

Diese Spezifikation beschreibt die Capability `test-infrastructure`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Migrations-FS ist aus Tests zugänglich

Die Datenbankmigrationen SHALL als exportiertes `db.FS embed.FS` in `internal/db/migrations.go` verfügbar sein, damit Test-Helpers die vollständige Migrationskette ausführen können ohne den `cmd/teamwerk`-Package zu importieren.

#### Scenario: testDB läuft alle Migrations durch

- **WHEN** `testutil.NewDB(t)` aufgerufen wird
- **THEN** wird eine SQLite-In-Memory-Datenbank mit vollständig angewendetem Schema zurückgegeben (alle Migrations von 001 bis aktuell)

#### Scenario: main.go kompiliert nach dem Refactor weiterhin

- **WHEN** `db.FS` statt des inline-embeds in `main.go` verwendet wird
- **THEN** kompiliert `go build ./cmd/teamwerk` ohne Fehler und das Verhalten der Anwendung ändert sich nicht

---

### Requirement: testutil.NewDB liefert isolierte In-Memory-Datenbank

`testutil.NewDB(t)` SHALL eine frische SQLite-In-Memory-Datenbank mit angewendeten Migrations zurückgeben. Jeder Test-Aufruf MUSS eine vollständig isolierte DB-Instanz erhalten (kein geteilter State zwischen Tests).

#### Scenario: Zwei Tests laufen ohne gegenseitige Beeinflussung

- **WHEN** zwei Tests jeweils `testutil.NewDB(t)` aufrufen und beide Daten schreiben
- **THEN** sieht kein Test die Daten des anderen

#### Scenario: DB wird nach dem Test automatisch freigegeben

- **WHEN** ein Test mit `testutil.NewDB(t)` endet
- **THEN** wird die DB-Verbindung via `t.Cleanup` geschlossen (kein Leak)

---

### Requirement: testutil.NewServer baut einen partiellen Chi-Router

`testutil.NewServer(t, db, routes)` SHALL einen `*httptest.Server` zurückgeben, der nur die übergebenen Routen registriert und die Auth-Middleware (`auth.Middleware`) korrekt eingebunden hat.

#### Scenario: Unauthentifizierter Request wird abgelehnt

- **WHEN** ein Request ohne `Authorization`-Header an eine geschützte Route gesendet wird
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Authentifizierter Request mit gültigem Token wird durchgelassen

- **WHEN** ein Request mit einem via `testutil.Token()` erzeugten Bearer-Token gesendet wird
- **THEN** erreicht der Request den Handler und gibt keinen 401 zurück

---

### Requirement: testutil.Token erzeugt signierte JWT-Tokens für beliebige Rollen

`testutil.Token(userID, role, clubFunctions)` SHALL einen gültigen JWT-String zurückgeben, der von der `auth.Middleware` des Testservers akzeptiert wird.

#### Scenario: Token für Trainer-Rolle

- **WHEN** `testutil.Token(42, "standard", []string{"trainer"})` aufgerufen wird
- **THEN** enthält der resultierende JWT `uid: 42` und `club_functions: ["trainer"]`

---

### Requirement: Fixture-Helpers erstellen minimale Test-Datensätze

`testutil` SHALL Helper-Funktionen für häufig benötigte Fixtures bereitstellen: `CreateUser`, `CreateTeam`, `CreateSeason`, `CreateTrainingSeries`, `CreateTrainingSession`.

Jede Funktion MUSS via `t.Fatal` abbrechen wenn das Einfügen fehlschlägt, und die erzeugte Entität zurückgeben.

#### Scenario: CreateUser legt einen User mit Passwort-Hash an

- **WHEN** `testutil.CreateUser(t, db, "standard", teamID)` aufgerufen wird
- **THEN** existiert ein User-Datensatz in der DB mit bcrypt-gehashetem Passwort und der angegebenen Rolle

#### Scenario: CreateSeason erstellt eine aktive Saison

- **WHEN** `testutil.CreateSeason(t, db, "2025/26")` aufgerufen wird
- **THEN** existiert ein Saison-Datensatz mit `is_active = 1`

---

### Requirement: Vitest-Infrastruktur im Frontend

Das Frontend SHALL eine vitest-basierte Test-Infrastruktur bereitstellen. Sie umfasst:

- `vitest`, `@vitest/coverage-v8`, `jsdom`, `@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`, `axios-mock-adapter` als Dev-Dependencies.
- `web/vitest.config.ts` mit `environment: 'jsdom'`, `setupFiles: ['./src/test/setup.ts']`, `globals: true`, `css: true`.
- `web/src/test/setup.ts` lädt `@testing-library/jest-dom` und reset axios-Mocks nach jedem Test.
- `web/package.json` enthält Scripts `test` (Watch) und `test:run` (Single Run).
- `make test` ruft `cd web && pnpm test:run` zusätzlich zu `go test ./...` auf.

#### Scenario: pnpm test:run läuft alle Vitest-Tests durch
- **WHEN** ein Entwickler `cd web && pnpm test:run` ausführt
- **THEN** läuft Vitest, findet alle `*.test.tsx`/`*.test.ts`-Dateien in `web/src/**` und liefert Exit-Code 0 bei allen Tests grün

#### Scenario: make test deckt Backend und Frontend ab
- **WHEN** ein Entwickler `make test` aufruft
- **THEN** läuft sowohl `go test ./...` als auch der Frontend-Test-Runner, und beide müssen Exit-Code 0 liefern

---

### Requirement: renderAsPersona-Helper

`web/src/test/renderAsPersona.tsx` SHALL einen Render-Helper bereitstellen, der eine React-Komponente mit einer der 11 Personas (siehe `permissions`-Spec) als aktivem User rendert.

Signatur:

```ts
export function renderAsPersona(
  personaId: PersonaId,
  ui: React.ReactNode,
  options?: { route?: string; childrenStub?: Child[] }
): RenderResult
```

Der Helper SHALL:
- einen `AuthContext.Provider` mit einem User aus der Persona-Definition aufsetzen,
- die `MemoryRouter` mit `initialEntries=[options.route ?? '/']` als Wrapper nutzen,
- den axios-Mock so initialisieren, dass Default-Antworten für `/profile/me` (mit ggf. übergebenen Children-Stubs), `/chat/conversations` und `/chat/broadcasts` vorliegen.

#### Scenario: renderAsPersona setzt den AuthContext-User
- **WHEN** `renderAsPersona('trainer', <TestComponent />)` aufgerufen wird
- **THEN** sieht `TestComponent` via `useAuth().user` einen User mit `role: 'standard'`, `clubFunctions: ['trainer']`, `isParent: false`

#### Scenario: renderAsPersona mit Route rendert MemoryRouter mit initial entry
- **WHEN** `renderAsPersona('admin', <App />, { route: '/mitglieder' })` aufgerufen wird
- **THEN** ist die Initial-URL `/mitglieder` und `App.tsx` rendert `MembersPage`

---

### Requirement: Persona-Fixtures geteilt zwischen Backend und Frontend

`internal/permissions/personas_test.go` und `web/src/test/personas.ts` SHALL die 11 in der `permissions`-Capability definierten Personas mit identischen Werten enthalten (Persona-ID, `role`, `clubFunctions`, `isParent`).

Jede Datei SHALL am Anfang einen Kommentar tragen, der die jeweils andere Datei referenziert.

#### Scenario: Persona-Listen sind identisch
- **WHEN** ein Entwickler die zwei Persona-Definitionen vergleicht
- **THEN** stimmen ID, role, clubFunctions und isParent für alle 11 Personas exakt überein

---

### Requirement: Backend-Permission-Matrix-Test-Helper

`internal/permissions/matrix_test.go` SHALL einen Tabelle-getriebenen Test bereitstellen, der pro (Persona × Endpoint) den erwarteten HTTP-Status verifiziert.

Struktur:

```go
type endpointCase struct {
    method   string
    path     string
    expected map[string]int  // Persona-ID → erwarteter HTTP-Status
}

var matrix = []endpointCase{ /* ein Eintrag pro Route */ }
```

Der Test SHALL:
- pro Test-Case eine eigene `testutil.NewDB(t)` aufsetzen,
- den vollständigen Router via `app.BuildRouter(handlers, nil)` aufbauen,
- pro Persona ein `testutil.Token(...)` erzeugen,
- den Request via `httptest.NewRecorder()` ausführen,
- den Status-Code mit `case.expected[persona.ID]` vergleichen.

#### Scenario: Matrix-Test prüft eine konkrete Endpoint-Persona-Kombination
- **WHEN** für `GET /api/members` der Eintrag `expected["spieler"] = 403` definiert ist und der Test ausgeführt wird
- **THEN** sendet der Test einen Request mit Spieler-Token an `GET /api/members` und assertiert `rec.Code == 403`

---

### Requirement: Drift-Check für unbekannte Routen

Der Backend-Matrix-Test SHALL beim Start alle in `internal/app/router.go` registrierten Routen durch `chi.Walk` ermitteln und vergleichen, ob jede Route auch in der `matrix`-Tabelle vorkommt. Routen ohne Eintrag SHALL den Test mit einer klaren Fehlermeldung failen lassen.

#### Scenario: Neue Route ohne Matrix-Eintrag failt
- **WHEN** in `internal/app/router.go` eine neue Route `r.Get("/api/new", h.X.Y)` ergänzt wird, ohne dass die Matrix-Tabelle erweitert wurde
- **THEN** failt der Matrix-Test mit `"Route GET /api/new ist nicht in der Permission-Matrix gepflegt — bitte specs/permissions/spec.md ergänzen"`

#### Scenario: Matrix-Eintrag ohne Route warnt
- **WHEN** ein Matrix-Eintrag existiert für eine Route, die nicht (mehr) im Router registriert ist
- **THEN** failt der Matrix-Test mit einer entsprechenden Warnung („Stale Matrix-Eintrag")

---

### Requirement: axios-Mock-Adapter-Setup

`web/src/test/apiMock.ts` SHALL einen Helper bereitstellen, der `axios-mock-adapter` an die `api`-Instanz aus `web/src/lib/api.ts` hängt und Default-Antworten für die im `renderAsPersona`-Setup benötigten Endpoints konfiguriert.

Default-Antworten:
- `GET /profile/me` → `{ id: 1, email: 'persona@test.local', children: [] }` (oder Override aus `options.childrenStub`)
- `GET /chat/conversations` → `[]`
- `GET /chat/broadcasts` → `[]`
- Sonstige GET-Routen → 200 mit `[]`

Nach jedem Test SHALL der Mock zurückgesetzt werden (`afterEach` Hook in `setup.ts`).

#### Scenario: axios-Mock liefert Default-Antworten
- **WHEN** ein Test eine Komponente rendert, die `api.get('/dashboard')` aufruft, ohne explizit zu mocken
- **THEN** liefert der Mock-Adapter `200 []` und der Test bleibt deterministisch

---

### Requirement: e2e-seed CLI-Subcommand

Das `teamwerk`-Binary SHALL einen Subcommand `e2e-seed --db=<path>`
bereitstellen, der eine frische SQLite-DB unter `<path>` anlegt, alle
Migrations anwendet und einen deterministischen Test-Datensatz einträgt.
Nach erfolgreichem Lauf ist die DB von Playwright direkt nutzbar (kein
weiterer Setup-Schritt nötig).

Der Datensatz umfasst mindestens:
- 1 Admin-User: `e2e@test.local` mit Passwort `E2ETestPassword!` (bcrypt
  gehasht wie im Prod-Login-Flow)
- 3 Standard-Test-User
- 1 Gruppenkonversation „E2E Chat mit Bildern" mit ≥ 10 Text- und
  ≥ 3 Bild-Nachrichten, alle für den Admin als gelesen markiert
- 1 Gruppenkonversation „E2E Chat unread" mit ≥ 20 Text-Nachrichten,
  letzte 3 nicht für den Admin gelesen (unreadCount = 3)

Der Subcommand SHALL idempotent bei existierender Ziel-Datei sein: wenn
`<path>` existiert, wird die Datei ohne Rückfrage überschrieben.

#### Scenario: e2e-seed legt frische DB an

- **WHEN** `teamwerk e2e-seed --db=./tmp-e2e.db` in einem leeren
  Verzeichnis ausgeführt wird
- **THEN** existiert `./tmp-e2e.db` mit vollständig migriertem Schema
  und dem Test-Datensatz

#### Scenario: e2e-seed überschreibt bestehende Datei

- **GIVEN** eine Datei `./tmp-e2e.db` existiert bereits
- **WHEN** `teamwerk e2e-seed --db=./tmp-e2e.db` erneut ausgeführt wird
- **THEN** wird die Datei überschrieben (deterministisch derselbe Inhalt)
  und der Exit-Code ist 0

#### Scenario: Login gegen geseedete DB funktioniert

- **GIVEN** die DB wurde per `e2e-seed` erzeugt und das Backend läuft
  gegen sie
- **WHEN** ein Request `POST /api/auth/login` mit
  `e2e@test.local` / `E2ETestPassword!` gesendet wird
- **THEN** antwortet der Server mit HTTP 200 und einem gültigen JWT

---

### Requirement: Trennung Vitest vs. Playwright — Konvention

Die Projekt-Konvention in `docs/agent/07-testing.md` SHALL beschreiben,
welche Test-Ebene für welche Klasse von Bugs zuständig ist:

- **Vitest + jsdom**: JS-Logik, Handler, Component-Rendering-Ausgabe,
  API-Mocks, State-Übergänge.
- **Playwright**: Browser-Verhalten (Scroll-Physik, Bild-Decode-Timing,
  echtes Layout, Focus, Animation, IntersectionObserver mit echtem
  Layout).

Bei UI-Änderungen an Scroll/Layout/Animation/Focus-Verhalten SHALL die
Konvention einen E2E-Test als „Reminder" vorschlagen (nicht harte
Pflicht, um Overhead zu vermeiden — der Autor entscheidet).

#### Scenario: 07-testing.md enthält den Abschnitt

- **WHEN** ein Entwickler `docs/agent/07-testing.md` liest
- **THEN** findet er einen Abschnitt „Wann Vitest, wann Playwright" mit
  klarem Kriterienkatalog und dem Verweis auf `make test-e2e`
