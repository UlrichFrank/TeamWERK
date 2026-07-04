# Tasks — payload-measurement-harness

> Test-/Tooling-only, kein Produktions-Code-Change. Kommt vor den drei Optimierungs-Changes. Ein Commit pro Task.

## 1. Deterministisches Seeding

- [x] 1.1 `internal/measure/seed.go` (oder im Testfile): fixer Datensatz über `testutil`-Fixtures, feste Referenzzeit-Konstante, keine Zufallswerte.
- [x] 1.2 `internal/arch/arch_test.go`: `internal/measure` als Composition-/Test-Support klassifizieren (darf `app`+`testutil` importieren).
- [x] 1.3 Test: `TestMeasure_SeedIsDeterministic`.

  _Commit:_ `test(measure): deterministischer Seed für Payload-Messung`

## 2. Payload- und 304-Messung

- [x] 2.1 `internal/measure/measure_test.go`: konfigurierte Routenliste via `httptest` abrufen, `len(body)`+Status erfassen.
- [x] 2.2 Referenzrouten zweimal (zweiter Call mit `If-None-Match`), Status+Bytes des zweiten Calls erfassen.
- [x] 2.3 Tests: `TestMeasure_RecordsPayloadPerRoute`.

  _Commit:_ `test(measure): Payload- und Revalidierungs-Messung pro Route`

## 3. SSE-Fan-out-Messung

- [x] 3.1 M `httptest`-Clients an `/api/events`; eine Mutation auslösen; je Client gelieferte Events/Bytes im Zeitfenster zählen.
- [x] 3.2 Test: `TestMeasure_SSEFanoutCountsPerClient` (M Clients, globale Mutation → M Events). Plus `TestMeasure_FanoutClientRosterIsFixed`.

  _Commit:_ `test(measure): SSE-Fan-out-Volumen pro Mutation`

## 4. Report + Makefile

- [x] 4.1 Report-Writer → `metrics/PAYLOAD.md` (Routen-Tabelle, Revalidierungs-Tabelle, Fan-out-Tabelle).
- [x] 4.2 `Makefile`: Target `measure` (Report, Exit 0); `.gitignore` um `metrics/PAYLOAD.md` ergänzen.
- [x] 4.3 Test: `TestMeasure_WritesReport`.

  _Commit:_ `feat(metrics): make measure schreibt metrics/PAYLOAD.md`

## 5. Baseline + optionaler Gate

- [x] 5.1 `make measure` auf aktuellem `main` laufen lassen; Ergebnis als `metrics/payload-baseline.md` committen.
- [ ] 5.2 Optional: `metrics/payload-thresholds.yml` + `make measure-gate` (Ratchet, Exit 1 bei Regression); NICHT in `pre-push`. — _bewusst zurückgestellt (optional); Baseline reicht als erster Vergleichsanker._

  _Commit:_ `chore(metrics): Payload-Baseline auf main einfrieren`

## 6. Abschluss

- [x] 6.1 `make test` (inkl. Architektur-Test) + `/verify-change`. — _`go vet ./...` + `go test ./...` (inkl. Arch-Test) grün; `go test -tags measure ./internal/measure` grün. Frontend (`pnpm test`) nicht betroffen (backend/tooling-only)._
- [x] 6.2 `openspec validate payload-measurement-harness --strict`. — _valid._
- [ ] 6.3 Proposal archivieren. — _OFFEN: bewusst dem Menschen nach Review überlassen._

  _Commit:_ `chore(metrics): archiviere payload-measurement-harness`
