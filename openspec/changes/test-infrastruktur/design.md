## Context

TeamWERK hat 19 Backend-Packages, von denen 18 ausschließlich HTTP-Handler enthalten — kein Service-Layer. Die gesamte Business-Logik ist direkt in Handler-Methoden implementiert, die `*sql.DB` direkt ansprechen. Es gibt einen einzigen Test (`internal/kader/age_brackets_test.go`) für eine reine Berechnungsfunktion.

Das Projekt nutzt `modernc.org/sqlite` (pure Go, kein CGo). Das ermöglicht SQLite-In-Memory-Tests ohne Build-Tags oder Cgo-Umgebung. Die Migrations-FS ist aktuell via `//go:embed` in `cmd/teamwerk/main.go` definiert und damit für Tests nicht zugänglich.

## Goals / Non-Goals

**Goals:**
- Wiederverwendbare Test-Infrastruktur, die für alle künftigen Packages nutzbar ist
- Integration-Tests, die den vollständigen HTTP→Handler→SQLite-Pfad abdecken, ohne Mocking
- Erster Fokus auf `internal/trainings` und `internal/games` (Kalender-Bereich, meiste Regressions)
- Statische Codeanalyse via golangci-lint als separates `make lint` Target
- `make test` läuft lokal und ist CI-ready

**Non-Goals:**
- Kein Service-Layer-Refactoring (Handler bleiben wie sie sind)
- Keine E2E-Tests (Playwright) in diesem Change
- Keine Frontend-Tests (Vitest) in diesem Change
- Kein 100%-Coverage-Ziel — nur die kritischsten Pfade
- Kein Mock-Framework

## Decisions

### 1. Migrations-Embed in `internal/db/` statt `cmd/teamwerk/main.go`

**Entscheidung:** Neues `internal/db/migrations.go` mit `//go:embed migrations/*.sql` und exportiertem `var FS embed.FS`. `main.go` und `testutil` nutzen beide `db.FS`.

**Alternativen:**
- *Embed in testutil duplizieren*: `//go:embed ../../internal/db/migrations/*.sql` — relativer Pfad, funktioniert, aber dupliziert den Embed und ist fragil bei Umbenennungen.
- *Test-Setup mit Raw-SQL ohne Migrations*: Schneller, aber Schema driftet von Produktions-Migrations weg — gefährlich.

**Warum db.FS:** Einzige Quelle der Wahrheit. Migrations laufen in Tests exakt wie in Produktion.

### 2. Partial Router statt Full Router im testutil

**Entscheidung:** Jedes Test-Package baut seinen eigenen minimalen Chi-Router mit nur den Routen, die es testet. `testutil` stellt Bausteine bereit (`NewDB`, `NewServer`, `Token`), aber kein vollständiges Router-Assembly.

**Alternativen:**
- *Zentraler `testutil.NewFullServer()`*: Ein Server mit allen Handlers — würde Circular Imports erzeugen, da `testutil` alle Domain-Packages importieren müsste.
- *Router aus `main.go` extrahieren*: `main` als Library nutzbar machen — möglich, aber `main` importiert `embed` für `web/dist`, was in Tests unnötig ist.

**Warum Partial:** Isolation, keine Circular Imports, jedes Package testet nur sich selbst.

### 3. Config und Hub in Tests: echte Instanzen mit Leer-Werten

**Entscheidung:** `testutil.NewConfig()` gibt eine `*config.Config` mit gesetztem `JWTSecret` und leerem SMTP/VAPID zurück. `hub.New()` wird echt instanziiert aber nicht gestartet. Push-Notification-Fehler (wegen leerem VAPID) werden ignoriert — die Goroutine schlägt lautlos fehl.

**Alternativen:**
- *Interface-Mocks für Hub/Notifications*: Müsste Interfaces einführen, die im Produktionscode nicht existieren — zu invasiv.
- *Build-Tags für Test-Stubs*: Komplexität nicht gerechtfertigt.

**Warum echte Instanzen:** Handler-Code bleibt unverändert, kein Produktions-Overhead, Test-Fokus liegt auf HTTP-Logik nicht auf Notification-Stack.

### 4. Test-Token-Erzeugung via `auth.IssueAccessToken`

**Entscheidung:** `testutil.Token(userID, role, clubFunctions)` ruft `auth.IssueAccessToken` mit dem Test-JWT-Secret auf. Kein separater Token-Builder.

**Warum:** Nutzt denselben Code-Pfad wie Produktion — Token-Format kann sich nicht unbemerkt unterscheiden.

### 5. golangci-lint via Binary, nicht als Go-Modul-Dependency

**Entscheidung:** `.golangci.yml` im Repo-Root, `make lint` lädt/nutzt `golangci-lint` als extern installiertes Binary (via `go install` oder Homebrew).

**Linter-Set:** `errcheck`, `staticcheck`, `unused`, `govet`, `gosimple`. Kein `gofmt` (bereits vom Editor gehandhabt), kein `exhaustive` (zu viele False Positives bei bestehenden Switches).

**Warum kein Go-Modul-Eintrag:** golangci-lint als Dependency zieht alle Linter-Dependencies in `go.sum` — erheblicher Overhead. Binary-Ansatz ist Go-Ecosystem-Standard.

## Risks / Trade-offs

**[Risk] Tests schlagen fehl wenn Migrations-Schema nicht zu Fixtures passt** → Mitigation: Fixtures verwenden nur Felder, die seit Migration 001 stabil sind; bei neuen Migrations wird `testutil/fixtures.go` mitgepflegt.

**[Risk] In-Memory SQLite hat kein WAL-Mode** → Mitigation: Für Tests ist WAL nicht nötig — kein Concurrent-Write-Problem in single-threaded Tests. `db.Open()` bleibt für Produktion, `testutil.NewDB()` nutzt `sql.Open` direkt mit `:memory:`.

**[Risk] Hub-Goroutine läuft in Tests weiter** → Mitigation: `t.Cleanup(hub.Stop)` wenn der Hub eine Stop-Methode hat; andernfalls tolerierbar, da Tests kurzlebig sind.

**[Risk] golangci-lint nicht installiert → `make lint` schlägt fehl** → Mitigation: Makefile gibt einen klaren Fehler mit Installationshinweis aus; CI kann golangci-lint-action nutzen.

## Migration Plan

1. `internal/db/migrations.go` anlegen, embed dorthin verschieben, `main.go` anpassen — kompiliert, keine Verhaltensänderung
2. `internal/testutil/` anlegen — kein Produktionscode berührt
3. Tests für `trainings` und `games` anlegen — `go test ./...` muss grün sein
4. `.golangci.yml` + `make lint` — bestehende Lint-Fehler dokumentieren und beheben oder `//nolint` mit Begründung
5. `make test` und `make lint` dokumentieren in CLAUDE.md

Rollback: Steps 3-5 sind rein additiv. Step 1 ist ein Refactor ohne Verhaltensänderung — Rollback durch Rückverschieben des embeds.
