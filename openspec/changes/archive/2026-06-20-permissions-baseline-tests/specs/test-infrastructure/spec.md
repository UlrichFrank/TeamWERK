## ADDED Requirements

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
