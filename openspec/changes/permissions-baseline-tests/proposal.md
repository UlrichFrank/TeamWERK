## Why

Das Berechtigungsmodell hat zwei orthogonale Achsen — System-Rolle (`admin`/`standard`) und Vereinsfunktionen (`spieler`/`trainer`/`vorstand`/`vorstand_beisitzer`/`kassierer`/`sportliche_leitung`) plus den orthogonalen Eltern-Marker (`is_parent`). Welche Persona welche Aktion ausführen darf, ist heute an drei Stellen verteilt:

1. **Middleware-Gates** in `internal/app/router.go` (`RequireRole` / `RequireClubFunction`).
2. **Inline-Checks** in jedem Handler (`claims.HasFunction`, Ownership, Team-Filter, `claims.IsParent`).
3. **Frontend-Gates** in `web/src/App.tsx` (`RoleRoute`), `web/src/components/AppShell.tsx` (`navModules`) und ~10 Pages mit lokalen `const isXxx = …`-Checks.

Es gibt keine zentrale Spec, die festhält **welche Persona welche Wirkung sehen darf** — weder als Doku noch als Test. Das hat drei konkrete Folgen:

- **Regressionen schleichen sich ein**, weil keine fachliche Invariante geprüft wird. Beispiel aus dem letzten Quartal: eine neue Page wurde ohne `RoleRoute`-Guard exponiert; fiel erst über User-Feedback auf.
- **Inkonsistenzen wachsen**: dieselbe Bedingung (`isTrainer`, `canEdit`) wird in 7 Files leicht verschieden ausgedrückt — manchmal mit `vorstand`, manchmal ohne. Manche Drift ist gewollt, manche nicht; ohne Spec ist beides nicht unterscheidbar.
- **Designlöcher werden unsichtbar**: `vorstand_beisitzer` und `kassierer` haben heute null Wirkung jenseits von `standard`. Niemand weiß, ob das Absicht ist.

Wir wollen den **heutigen Stand** vollständig festschreiben (Specs + Regressions-Tests), bevor wir korrigieren. Korrekturen kommen in einem separaten Folge-Change `permissions-cleanup`.

## What Changes

- **NEW** Capability `permissions`: Eine Spec, die für jeden Backend-Endpoint und jede Frontend-Route/Navigation/Inline-Aktion festlegt, welche der 11 definierten Personas Zugriff hat und welche nicht. Quelle der Wahrheit für die Regressions-Tests.
- **MOD** Capability `test-infrastructure`:
  - Backend: Tabelle-getriebener Permission-Matrix-Test (`internal/permissions/matrix_test.go`), der pro Persona × Endpoint den erwarteten HTTP-Status verifiziert. Nutzt `testutil.Token()` und einen einmal aufgebauten Test-Router.
  - Frontend: **Vitest-Infrastruktur neu einführen** — Dependencies (`vitest`, `@testing-library/react`, `@testing-library/jest-dom`, `jsdom`, `@testing-library/user-event`), Vitest-Konfig in `vite.config.ts`, Setup-File mit Persona-Fixtures, axios-Mocks. Tests rendern jede Page mit jeder Persona und prüfen Render/Redirect plus die definierten Inline-Gates.
- **chore(web):** `pnpm test` und `pnpm test:run` Scripts in `web/package.json`.
- **chore(scheduler):** Backend-Permission-Matrix-Test läuft im `make test`-Pfad mit (keine eigene CI-Stage nötig — `go test ./...` reicht).
- **chore(ci):** `pnpm --filter web test:run` zu `make test` ergänzen, sodass beide Pfade auf einmal grün sein müssen.

**Was bewusst NICHT in diesem Proposal ist:**

- Keine Korrektur der Designlöcher (`vorstand_beisitzer`, `kassierer`, `/anfragen`-Mismatch). Diese landen im Folge-Change `permissions-cleanup`. Status quo wird hier **bewusst** als korrekt festgeschrieben — auch wenn er Lücken zeigt — damit der Folge-Change sie sichtbar adressieren kann.
- Keine Refaktorierung der Inline-Checks in zentrale Helfer (`permissions.ts`). Auch das gehört in einen Folge-Change, sobald die Spec die Drift sichtbar gemacht hat.
- Keine E2E-Flows (Persona klickt durch Spielanlage o.ä.). Smoke + Inline-Gate-Tests reichen, um Drift zu fangen.

## Capabilities

### New Capabilities

- **permissions** — Codifiziert die effektiven Rechte pro Persona × Wirkung (Backend-Route, Frontend-Route, Sidebar-Item, Inline-Button).

### Modified Capabilities

- **test-infrastructure** — Ergänzt Vitest-Infrastruktur, Persona-Fixtures, Render-Helpers (`renderAsPersona`), axios-Mock-Pattern, sowie den Backend-seitigen Permission-Matrix-Test-Helper.

## Impact

**Code (neu):**

- `internal/permissions/matrix_test.go` — Tabelle-getriebener Backend-Matrix-Test.
- `internal/permissions/personas_test.go` — Persona-Fixtures für Backend.
- `web/vitest.config.ts` — Vitest-Konfig (jsdom, setupFiles, alias).
- `web/src/test/setup.ts` — `@testing-library/jest-dom`, axios-Mock-Reset.
- `web/src/test/personas.ts` — Persona-Definitionen (typed, shared mit Spec).
- `web/src/test/renderAsPersona.tsx` — Render-Helper mit AuthContext-Stub.
- `web/src/test/apiMock.ts` — axios-Mock-Adapter-Setup (oder MSW — Entscheidung in `design.md`).
- `web/src/components/__tests__/AppShell.permissions.test.tsx` — Sidebar-Item-Sichtbarkeit pro Persona.
- `web/src/__tests__/RoleRoute.permissions.test.tsx` — RoleRoute-Redirects pro Persona.
- `web/src/pages/__tests__/*.permissions.test.tsx` — Pro Page eine Smoke + Inline-Gate-Test-Datei (~27 Files).

**Code (geändert):**

- `web/package.json` — vitest-Dependencies + Scripts.
- `web/vite.config.ts` — `test`-Block delegiert an `vitest.config.ts` (oder inline).
- `Makefile` — `test`-Target ruft zusätzlich `cd web && pnpm test:run` auf.

**Datenbank:**

Keine Migrations. Permissions sind reine Code-Invarianten.

**Doku:**

- `CLAUDE.md` Abschnitt „Test-Standard" um Hinweis ergänzen: bei neuen Routen ODER neuen Pages MUSS die `permissions`-Spec angepasst und der Matrix-Test erweitert werden.

**Risiko / Aufwand:**

- Vitest-Setup ist Standard-Pattern, aber das erste Mal in diesem Repo → ~1 Tag Lernzeit.
- 27 Pages × 11 Personen × 2 Test-Files (Smoke + Inline) ≈ ~594 Test-Cases. Werden über parametrisierte Tests (`test.each`) auf ~30 Test-Dateien mit ~5–15 Tests/Datei geschrumpft, NICHT 594 individuell.
- Backend-Matrix-Test: ~200 Endpoints × 11 Personen ≈ 2200 Assertions, via Table-Test auf eine Datei reduziert.
- Wenn ein Endpoint oder eine Page neu angelegt wird ohne dass die Spec angepasst wurde, schlagen die Tests zu (Coverage-Sicherung).

**Breaking changes:** Keine.

## Test-Anforderungen

- **Backend-Matrix-Test:** Für jede der 11 Personas × jeden Endpoint aus `internal/app/router.go` SHALL der erwartete HTTP-Status (200/201/204/400/403/404) erreicht werden. Test: `TestPermissionMatrix_Backend`.
- **Frontend-Route-Smoke:** Für jede der 11 Personas × jede Route aus `web/src/App.tsx` SHALL die Page rendern ODER auf `/` weiterleiten — gemäß der Spec-Vorgabe pro Route. Test: `RoleRoute.permissions.test.tsx`.
- **Sidebar-Sichtbarkeit:** Für jede Persona SHALL die Sidebar genau die definierten Nav-Items zeigen — nicht mehr, nicht weniger. Test: `AppShell.permissions.test.tsx`.
- **Inline-Gate-Sichtbarkeit:** Pro Page mit Inline-Gates (MembersPage, MemberDetailPage, TerminePage, TermineDetailPage, SpieltagDetailPage, DutyPage, ChatPage, KalenderPage, MemberDatenschutzTab) SHALL der Action-Button (`Neu`, `Bearbeiten`, `Broadcast`, `Abwesenheit anlegen`, …) pro Persona gemäß Spec sichtbar/versteckt sein.
- **Drift-Schutz:** Wenn eine neue Route oder ein neuer Inline-Gate ohne Spec-Anpassung eingeführt wird, MUSS mindestens ein bestehender Test failen (Sicherung über vollständige Persona-Schleife in den Tests).
