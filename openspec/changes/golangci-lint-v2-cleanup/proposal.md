## Why

Mit der Migration der `.golangci.yml` auf v2-Schema (vorausgegangener `chore(lint)`-Commit) wird der Linter erstmals seit längerer Zeit wieder zuverlässig ausgeführt — und meldet **23 Pre-existing-Issues** im Bestandscode. Diese Issues sind harmlos im Sinne von „nicht funktionsverändernd", aber sie machen den Lint-Gate unbrauchbar: solange `make lint` rot ist, kann der `pre-push`-Hook keinen sauberen Gate-Lauf abbilden, und neue Lint-Issues gehen im Rauschen unter.

Zusätzlich scannt der Linter aktuell `web/node_modules/` mit, weil die Config keinen Exclude-Pfad für JS-Abhängigkeiten setzt. Daher 2 govet-Findings in `flatted` (Third-Party-Go-Code, der über `pnpm install` mitkommt) — semantisch eindeutig kein Repo-Code.

Ohne diese Bereinigung kann der `pre-push`-Hook den Lint-Gate nicht aktivieren und der CLAUDE.md-Standard „`make lint` grün vor Push" ist nicht durchsetzbar.

## What Changes

**Lint-Config (`.golangci.yml`):**
- `linters.exclusions.paths` ergänzen um `web/node_modules` und `node_modules`, damit Drittanbieter-Go-Code unter `node_modules` nicht gescannt wird

**Toten Code entfernen (6× `unused`):**
- `cmd/teamwerk/main.go`: `corsMiddleware`, `spaHandler` löschen (SPA wird via `embed.FS` ausgeliefert; CORS wird nicht mehr im Code-Pfad benötigt — verifizieren, dass es keine versteckten Aufrufer gibt)
- `internal/chat/handler_test.go`: `directConvExists` löschen
- `internal/files/handler_test.go`: `claimsWithFn` löschen
- `internal/games/handler.go`: `effectiveEventDuration`, `findTemplateForGame` löschen (vor Löschung prüfen, ob sie nicht von außerhalb des `games`-Pakets referenziert sind — bei `func (h *Handler).foo` unwahrscheinlich)

**Error-Strings entkapitalisieren (8× `ST1005`):**
Erste Buchstaben in Fehler-Strings kleinschreiben, da Fehler in Go i. d. R. in zusammengesetzten Kontext-Strings auftauchen (`fmt.Errorf("... %w", err)`). Betrifft:
- `internal/config/handler.go:50, 60`
- `internal/games/handler.go:159, 177, 180`
- `internal/games/regen.go:540, 558, 561`

**Quick-Fix-Refactorings (3× `QF1001` + 4× `QF1003`):**
Mechanisch — `if-else if`-Ketten auf gleichen Diskriminator → `switch`-Statements; doppelte Negationen → De-Morgan-Vereinfachung:
- `internal/carpooling/handler.go:372, 390, 471` (tagged switch)
- `internal/carpooling/handler_test.go:393, 394` (De Morgan)
- `internal/games/handler.go:271` (De Morgan)
- `internal/members/welcome_email.go:79` (tagged switch)

**Nicht** geändert: Geschäftslogik, API-Verträge, Tests-Assertions (außer wenn ein Test auf einen entkapitalisierten Error-String prüft — dann Test mit-angleichen).

## Capabilities

### New Capabilities

- `lint-clean-baseline`: dokumentiert die Invariante „`golangci-lint run ./...` liefert 0 Issues" und die Mindest-Linter (`errcheck`, `govet`, `staticcheck`, `unused`). Dadurch wird der pre-push-Lint-Gate spezifisch abdeckbar.

### Modified Capabilities

_(keine — reiner Pflege-Change)_

## Test-Anforderungen

**Keine neuen HTTP-Routen** → keine neuen Route-Tests nötig.

Pflicht-Verifikation:
- `golangci-lint run ./...` muss **0 Issues** zeigen (Hauptinvariante)
- `go test -race ./...` muss durchlaufen (Verhalten unverändert)
- Falls ein bestehender Test einen kapitalisierten Error-String wörtlich vergleicht, muss er auf die kleingeschriebene Variante umgestellt werden (mit-im-Commit)

**Invariante:** Nach Abschluss gibt `golangci-lint run ./...` Exit-Code 0 mit `0 issues.`.

## Impact

- **Dateien:**
  - `.golangci.yml`
  - `cmd/teamwerk/main.go`
  - `internal/chat/handler_test.go`
  - `internal/files/handler_test.go`
  - `internal/games/handler.go`
  - `internal/games/regen.go`
  - `internal/config/handler.go`
  - `internal/carpooling/handler.go`
  - `internal/carpooling/handler_test.go`
  - `internal/members/welcome_email.go`
- **DB-Migration:** keine
- **API-Schema:** keine Änderung
- **Risiko:**
  - Beim Löschen von `corsMiddleware`/`spaHandler`: vorher mit `grep -r` prüfen, dass sie wirklich nirgends mehr aufgerufen werden (auch nicht via tag- oder reflektionsbasiert)
  - Tagged-switch-Refactorings sind semantisch äquivalent, aber per Code-Review verifizieren
- **Folge-Verbesserung:** anschließend kann der `pre-push`-Hook in `scripts/` (siehe CLAUDE.md → Harness) den Lint-Gate aktivieren, falls noch nicht aktiv
