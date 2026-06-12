## Why

Änderungen an der Kalender- und Trainings-Logik werden aktuell nur manuell verifiziert — Regressions rutschen durch. Die KI-gestützte Entwicklung (Dark-Factory / Spec-Driven) braucht automatisierte Tests als verlässliches Abnahmekriterium: grün = fertig.

## What Changes

- **Neues Package `internal/testutil/`** mit wiederverwendbaren Helpers für alle künftigen Integration-Tests (DB, Router, Token, Fixtures)
- **Migration-Embed verschoben** von `cmd/teamwerk/main.go` nach `internal/db/` (Export als `db.FS`) — damit Tests die Migrations nutzen können ohne den main-Package zu importieren
- **Integration-Tests für `internal/trainings`**: ListSessions, CreateSeries, Respond (RSVP), SaveAttendances
- **Integration-Tests für `internal/games`**: ListGames, CreateGame
- **Statische Codeanalyse** via golangci-lint mit kuratierter Konfiguration (`.golangci.yml`)
- **Makefile-Targets** `make test` und `make lint`

## Capabilities

### New Capabilities

- `test-infrastructure`: Gemeinsame Test-Helpers (testDB, testServer, testToken, Fixtures) als Grundlage für alle zukünftigen Integration-Tests
- `trainings-test-coverage`: Integration-Tests für die Trainings- und Kalender-API (die Hauptquelle bisheriger Regressions)
- `static-analysis`: golangci-lint-Setup mit projektspezifischer Konfiguration

### Modified Capabilities

*(keine bestehenden Spec-Anforderungen ändern sich)*

## Impact

- **`internal/db/`**: Neue Datei `migrations.go` mit `//go:embed` und exportiertem `FS`; `cmd/teamwerk/main.go` nutzt `db.FS` statt eigenem embed
- **`internal/testutil/`**: Neues Package, nur in `_test.go`-Dateien importierbar (kein Produktions-Overhead)
- **`internal/trainings/handler_test.go`**: Erste ~10 Integration-Tests
- **`internal/games/handler_test.go`**: Erste ~5 Integration-Tests
- **`.golangci.yml`**: Neue Datei im Repo-Root
- **`Makefile`**: Zwei neue Targets (`test`, `lint`)
- **Dependencies**: `golangci-lint` (Binary, kein Go-Modul-Eintrag nötig); kein neuer Runtime-Overhead
