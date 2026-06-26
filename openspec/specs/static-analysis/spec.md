# static-analysis Specification

## Purpose

Diese Spezifikation beschreibt die Capability `static-analysis`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: golangci-lint läuft sauber auf dem gesamten Backend

Das Projekt SHALL eine `.golangci.yml` im Repository-Root enthalten und `make lint` SHALL fehlerfrei durchlaufen. Bestehende Findings, die nicht kurzfristig behebbar sind, MÜSSEN mit `//nolint:<linter> // Reason: ...` annotiert werden.

Aktivierte Linter: `govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`.

#### Scenario: make lint schlägt nicht fehl bei sauberem Code

- **WHEN** `make lint` nach dem Setup auf dem unveränderten Codestand ausgeführt wird
- **THEN** gibt golangci-lint Exit-Code 0 zurück

#### Scenario: Ignorierte Fehler sind begründet

- **WHEN** ein `//nolint`-Kommentar im Code vorkommt
- **THEN** enthält er eine Begründung (z.B. `//nolint:errcheck // Close error on read-only operation is irrelevant`)

---

### Requirement: make test führt alle Go-Tests aus

`make test` SHALL `go test ./...` mit Race-Detector ausführen und bei Fehlern einen Nicht-Null-Exit-Code zurückgeben.

#### Scenario: Alle Tests laufen grün

- **WHEN** `make test` auf einem sauberen Build ausgeführt wird
- **THEN** ist der Exit-Code 0 und alle Tests sind als PASS markiert

#### Scenario: Failing Test bricht make test ab

- **WHEN** ein Test fehlschlägt
- **THEN** gibt `make test` einen Nicht-Null-Exit-Code zurück

---

### Requirement: make lint gibt Installationshinweis wenn golangci-lint fehlt

`make lint` SHALL prüfen ob `golangci-lint` installiert ist und im Fehlerfall eine verständliche Fehlermeldung mit Installationsbefehl ausgeben.

#### Scenario: golangci-lint nicht installiert

- **WHEN** `make lint` aufgerufen wird und `golangci-lint` nicht im PATH ist
- **THEN** gibt make eine Meldung aus: `golangci-lint nicht gefunden. Installieren: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
