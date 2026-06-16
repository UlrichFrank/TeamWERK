## 1. Persona-Definitionen festschreiben

- [ ] 1.1 `internal/permissions/personas_test.go` anlegen mit der Liste der 11 Personas (siehe `design.md` §2). Felder: `ID`, `Role`, `ClubFunctions`, `IsParent`. Commit: `test(permissions): Persona-Fixtures für Backend-Matrix-Test`
- [ ] 1.2 `web/src/test/personas.ts` mit identischen 11 Personas (TypeScript-Variante). Commit: `test(web): Persona-Fixtures für Vitest-Tests`
- [ ] 1.3 Beide Persona-Listen im Code-Review-Hinweis aneinander koppeln: README-Kommentar oben in beiden Dateien, der auf die jeweils andere verweist. Commit: Teil von 1.1/1.2.

## 2. Permissions-Spec final reviewen

- [ ] 2.1 `openspec/changes/permissions-baseline-tests/specs/permissions/spec.md` durch den Vorstand reviewen — speziell die Sektionen „Status quo: Designlöcher" für `vorstand_beisitzer`, `kassierer`, `/anfragen`-Mismatch. Wenn der Vorstand dort eine Realität anders sieht, in der Spec korrigieren BEVOR Tests geschrieben werden. Commit (falls Anpassungen): `docs(specs): permissions-spec Reviewer-Feedback`

## 3. Vitest-Infrastruktur einführen

- [ ] 3.1 `cd web && pnpm add -D vitest @vitest/coverage-v8 jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event axios-mock-adapter` — Dev-Dependencies installieren. Commit: `chore(web): vitest-Dependencies hinzugefügt`
- [ ] 3.2 `web/vitest.config.ts` anlegen mit `environment: 'jsdom'`, `setupFiles: ['./src/test/setup.ts']`, `globals: true`, `css: true`, Alias-Import wie in `vite.config.ts`. Commit: `chore(web): vitest.config.ts`
- [ ] 3.3 `web/src/test/setup.ts` mit `import '@testing-library/jest-dom'`, `afterEach(() => cleanup())` und axios-Mock-Adapter-Reset. Commit: Teil von 3.2.
- [ ] 3.4 `web/src/test/apiMock.ts` — Helper, der einen `axios-mock-adapter` an die `api`-Instanz aus `web/src/lib/api.ts` hängt, mit Default-Mocks (`/profile/me` → leeres Objekt, alle anderen → 200 mit `[]`). Commit: `test(web): axios-Mock-Setup für Persona-Tests`
- [ ] 3.5 `web/src/test/renderAsPersona.tsx` — Helper, der einen Komponentenbaum mit `AuthContext.Provider` rendert, in dem `user` aus der Persona-Definition gespeist wird. Akzeptiert Optionen `route` (initial path), `personaId`. Liefert die Render-Result von `@testing-library/react`. Commit: `test(web): renderAsPersona-Helper`
- [ ] 3.6 `web/package.json` Scripts: `"test": "vitest"`, `"test:run": "vitest run"`, `"test:ui": "vitest --ui"` (optional). Commit: `chore(web): vitest-Scripts`
- [ ] 3.7 `Makefile`-Target `test`: aktuelles `go test ./...` ergänzen um `&& cd web && pnpm test:run`. Falls kein `test`-Target existiert, neu anlegen. Commit: `chore(make): test-Target deckt Backend und Frontend ab`

## 4. Backend Permission-Matrix-Test

- [ ] 4.1 `internal/permissions/matrix_test.go` anlegen. Definiere `endpointCase` aus `design.md` §4 und die Liste `var matrix = []endpointCase{...}`. Für jeden Endpoint aus `internal/app/router.go` einen Eintrag mit `expected: map[string]int` (Status pro Persona-ID). Quelle: `openspec/changes/permissions-baseline-tests/specs/permissions/spec.md`. Commit: `test(permissions): Matrix-Test-Skelett mit allen Routen`
- [ ] 4.2 Test-Body schreiben: pro `endpointCase` einen `t.Run(case.method+" "+case.path, …)`, der pro Persona einen Subtest startet. Setup: `testutil.NewDB(t)`, `testutil.NewServer(t, db, allRoutes)` ODER kompletter Router-Aufbau via `app.BuildRouter(handlers, nil)`. Token: `testutil.Token(...)` mit der Persona. Request: `httptest.NewRequest(...)`. Assert: `recorder.Code == expected[persona.ID]`. Commit: `test(permissions): Matrix-Test-Body und Persona-Iteration`
- [ ] 4.3 Drift-Check: am Anfang des Test-Bodys eine Liste aller registrierten Chi-Routen aus dem Router walken (`chi.Walk`), und sicherstellen dass jede Route in `matrix` vorkommt. Wenn nicht: `t.Fatal("Route X ist nicht in der Permission-Matrix gepflegt — bitte specs/permissions/spec.md ergänzen")`. Commit: `test(permissions): Drift-Check für neue Routen ohne Spec`
- [ ] 4.4 `make test` lokal grün — alle Matrix-Cases laufen und matchen `design.md`-Tabelle. Falls einzelne Cases nicht matchen: Spec war falsch — Spec anpassen (Test ist Wahrheit), oder im Reviewer-Loop klären. Commit: pro Anpassung `docs(specs): permissions Korrektur an Realität (Endpoint X)`.

## 5. Frontend RoleRoute-Test

- [ ] 5.1 `web/src/__tests__/RoleRoute.permissions.test.tsx` anlegen. Test pro Persona × jeder Route aus `App.tsx`: render `<MemoryRouter initialEntries={[route]}><App /></MemoryRouter>` (mit AuthContext via `renderAsPersona`), prüfe ob die erwartete Page-Markierung (`data-testid="page-{name}"` ODER ein eindeutiger Page-Heading) sichtbar ist, oder ob auf `/` weitergeleitet wurde. Erwartungen kommen aus `permissions`-Spec. Commit: `test(web): RoleRoute-Matrix-Test`
- [ ] 5.2 In `App.tsx` ggf. `data-testid="page-XXX"` an Wrapper jeder Page ergänzen, ODER einen `<PageMarker name="…" />` einbauen. Empfehlung: kleines Wrapper-Pattern `<Page name="dashboard">…</Page>`, der ein `data-testid` setzt. Alternativ Reliance auf bestehende Headings (brüchig). Commit: `refactor(web): Page-Test-IDs für Permission-Tests`

## 6. Frontend AppShell-Nav-Test

- [ ] 6.1 `web/src/components/__tests__/AppShell.permissions.test.tsx` anlegen. Test pro Persona: render `AppShell` mit `Outlet`-Stub, prüfe per `screen.queryByText(label)` für jeden Nav-Item aus `navModules`, ob er sichtbar/versteckt ist. Erwartungen aus Spec-Tabelle `permissions.nav-visibility`. Commit: `test(web): AppShell-Nav-Sichtbarkeit pro Persona`
- [ ] 6.2 Module-Header (`Verwaltung`, etc.) prüfen: wenn alle Items des Moduls unsichtbar sind, MUSS auch der Modul-Header unsichtbar sein (Bestätigung der `visibleItems.length === 0`-Logik in `AppShell.tsx:163`). Commit: Teil von 6.1.

## 7. Frontend Inline-Gate-Tests

Pro Page-Inline-Gate aus `design.md` §5: eine Test-Datei `web/src/pages/__tests__/<Page>.permissions.test.tsx`. Jede Datei testet das Sichtbarkeits-Verhalten des Action-Elements pro Persona.

- [ ] 7.1 `MembersPage.permissions.test.tsx` — „Mitglied anlegen"-Button. Commit: `test(web): MembersPage Inline-Gates pro Persona`
- [ ] 7.2 `MemberDetailPage.permissions.test.tsx` — „Bearbeiten"-Button. Commit: `test(web): MemberDetailPage Inline-Gates`
- [ ] 7.3 `TerminePage.permissions.test.tsx` — „Training anlegen"-Button. Commit: `test(web): TerminePage Inline-Gates`
- [ ] 7.4 `TermineDetailPage.permissions.test.tsx` — Edit-Actions. Commit: `test(web): TermineDetailPage Inline-Gates`
- [ ] 7.5 `SpieltagDetailPage.permissions.test.tsx` — „Spiel bearbeiten". Commit: `test(web): SpieltagDetailPage Inline-Gates`
- [ ] 7.6 `DutyPage.permissions.test.tsx` — Slot-Mutation-Actions. Commit: `test(web): DutyPage Inline-Gates`
- [ ] 7.7 `ChatPage.permissions.test.tsx` — „Broadcast schreiben" + User-Picker-Erweiterung. Commit: `test(web): ChatPage Inline-Gates`
- [ ] 7.8 `KalenderPage.permissions.test.tsx` — „Spiel anlegen" + „Abwesenheit anlegen". Commit: `test(web): KalenderPage Inline-Gates`
- [ ] 7.9 `MemberDatenschutzTab.permissions.test.tsx` — SEPA-Mandat-Aktionen. Commit: `test(web): MemberDatenschutzTab Inline-Gates`

## 8. Spec-Validierung & Doku

- [ ] 8.1 `CLAUDE.md` Abschnitt „Test-Standard" erweitern: „Bei neuen Routen MUSS `openspec/specs/permissions/spec.md` ergänzt UND ein Eintrag in `internal/permissions/matrix_test.go` und in den Frontend-Permission-Tests hinzugefügt werden, sonst failt der Drift-Check (§4.3)." Commit: `docs(claude-md): Test-Standard erweitert um Permission-Matrix`
- [ ] 8.2 `openspec/AGENTS.md` (falls vorhanden) prüfen — andernfalls `openspec/specs/permissions/spec.md` mit einem README-Hinweis im Kopf versehen, der den Bezug zu Matrix-Test und Frontend-Tests dokumentiert. Commit: `docs(specs): permissions-spec README-Hinweis`

## 9. Verifikation

- [ ] 9.1 `make test` lokal grün — Backend-Matrix + alle Frontend-Tests laufen durch.
- [ ] 9.2 Probe-Drift: temporär eine Test-Route in `internal/app/router.go` hinzufügen (`r.Get("/api/test-drift", h.Dashboard.Get)`) → erwarte dass Drift-Check failt. Anschließend Route zurückbauen. Nicht commiten — manuelles Smoke.
- [ ] 9.3 Probe-Drift Frontend: temporär eine Route in `App.tsx` hinzufügen → erwarte dass Smoke-Test failt. Anschließend zurückbauen.
- [ ] 9.4 Coverage: `cd web && pnpm test:run --coverage` — Ziel-Coverage NICHT als Gate, aber als Sicht: alle Pages aus §7 sollten ≥80% Statement-Coverage haben.

## 10. Abschluss

- [ ] 10.1 PR-Beschreibung listet alle Personas, alle getesteten Pages, alle getesteten Endpoints. Commit: `chore(openspec): permissions-baseline-tests Proposal applied`
- [ ] 10.2 Follow-up-Issue für `permissions-cleanup`-Proposal anlegen, der die in §10 der Spec dokumentierten Designlöcher adressiert.
