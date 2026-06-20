## ADDED Requirements

### Requirement: Repository erfüllt den golangci-lint-Baseline-Gate

Das Repository SHALL den Befehl `golangci-lint run ./...` (mit der projektweiten `.golangci.yml`) mit Exit-Code 0 und der Meldung „0 issues." beenden.

Folgende Linter SHALL aktiv und durchsetzbar sein (per `.golangci.yml` aktiviert oder als v2-Default):

- `errcheck` (mit dokumentierten Excludes für idiomatische Fire-and-forget-Patterns)
- `govet`
- `staticcheck` (umfasst Sub-Checks ST1xxx, QF1xxx)
- `unused`

Drittanbieter-Go-Code unter `node_modules`-Verzeichnissen (insbesondere `web/node_modules`) SHALL über `linters.exclusions.paths` von der Lint-Auswertung ausgeschlossen sein, damit Findings aus mitgelieferten Bibliotheken nicht den Gate brechen.

#### Scenario: Frischer Checkout besteht den Lint-Gate

- **WHEN** ein Entwickler `git clone` ausführt, `pnpm install` (für `web/node_modules`) sowie `go mod download` laufen lässt und anschließend `golangci-lint run ./...` aufruft
- **THEN** ist der Exit-Code 0
- **AND** die Ausgabe endet mit `0 issues.`

#### Scenario: node_modules-Drittanbietercode wird ignoriert

- **WHEN** unter `web/node_modules/<package>/...` Go-Quelltext liegt, der eigenständig `govet`- oder `staticcheck`-Findings auslösen würde
- **THEN** taucht dieser Code in `golangci-lint run ./...` nicht auf
- **AND** der Gesamt-Gate bleibt grün

#### Scenario: Ein neues Lint-Finding bricht den Gate

- **WHEN** ein Commit Code hinzufügt, der `staticcheck`, `govet`, `errcheck` (außerhalb der dokumentierten Excludes) oder `unused` triggert
- **THEN** liefert `golangci-lint run ./...` Exit-Code ≠ 0
- **AND** der Commit darf nicht ohne Bereinigung in `main` gemerged werden (durchgesetzt durch `pre-push`-Hook bzw. CI)

#### Scenario: Erweiterung der errcheck-Excludes erfolgt nur dokumentiert

- **WHEN** ein neues Pattern in `linters.settings.errcheck.exclude-functions` aufgenommen wird
- **THEN** SHALL die Begründung als Kommentar oberhalb des Eintrags in `.golangci.yml` stehen
- **AND** der Eintrag SHALL ein konkretes idiomatisches Muster (z. B. `tx.Rollback` in `defer`) und nicht eine ganze Bibliothek pauschal abdecken
