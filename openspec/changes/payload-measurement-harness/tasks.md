# Tasks — payload-measurement-harness

> Test-/Tooling-only, kein Produktions-Code-Change. Kommt vor den drei Optimierungs-Changes. Ein Commit pro Task.

## 1. Deterministisches Seeding

- [ ] 1.1 `internal/measure/seed.go` (oder im Testfile): fixer Datensatz über `testutil`-Fixtures, feste Referenzzeit-Konstante, keine Zufallswerte.
- [ ] 1.2 `internal/arch/arch_test.go`: `internal/measure` als Composition-/Test-Support klassifizieren (darf `app`+`testutil` importieren).
- [ ] 1.3 Test: `TestMeasure_SeedIsDeterministic`.

  _Commit:_ `test(measure): deterministischer Seed für Payload-Messung`

## 2. Payload- und 304-Messung

- [ ] 2.1 `internal/measure/measure_test.go`: konfigurierte Routenliste via `httptest` abrufen, `len(body)`+Status erfassen.
- [ ] 2.2 Referenzrouten zweimal (zweiter Call mit `If-None-Match`), Status+Bytes des zweiten Calls erfassen.
- [ ] 2.3 Tests: `TestMeasure_RecordsPayloadPerRoute`.

  _Commit:_ `test(measure): Payload- und Revalidierungs-Messung pro Route`

## 3. SSE-Fan-out-Messung

- [ ] 3.1 M `httptest`-Clients an `/api/events`; eine Mutation auslösen; je Client gelieferte Events/Bytes im Zeitfenster zählen.
- [ ] 3.2 Test: `TestMeasure_SSEFanoutCountsPerClient` (M Clients, globale Mutation → M Events).

  _Commit:_ `test(measure): SSE-Fan-out-Volumen pro Mutation`

## 4. Report + Makefile

- [ ] 4.1 Report-Writer → `metrics/PAYLOAD.md` (Routen-Tabelle, Revalidierungs-Tabelle, Fan-out-Tabelle).
- [ ] 4.2 `Makefile`: Target `measure` (Report, Exit 0); `.gitignore` um `metrics/PAYLOAD.md` ergänzen.
- [ ] 4.3 Test: `TestMeasure_WritesReport`.

  _Commit:_ `feat(metrics): make measure schreibt metrics/PAYLOAD.md`

## 5. Baseline + optionaler Gate

- [ ] 5.1 `make measure` auf aktuellem `main` laufen lassen; Ergebnis als `metrics/payload-baseline.md` committen.
- [ ] 5.2 Optional: `metrics/payload-thresholds.yml` + `make measure-gate` (Ratchet, Exit 1 bei Regression); NICHT in `pre-push`.

  _Commit:_ `chore(metrics): Payload-Baseline auf main einfrieren`

## 6. Abschluss

- [ ] 6.1 `make test` (inkl. Architektur-Test) + `/verify-change`.
- [ ] 6.2 `openspec validate payload-measurement-harness --strict`.
- [ ] 6.3 Proposal archivieren.

  _Commit:_ `chore(metrics): archiviere payload-measurement-harness`
