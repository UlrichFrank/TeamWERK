## 1. Tooling verankern

- [x] 1.1 `scc` als `go.mod`-`tool`-Direktive pinnen (`go get -tool github.com/boyter/scc/v3`), `go tool scc --version` verifizieren
- [x] 1.2 `jscpd` als pnpm-devDependency hinzufügen (`pnpm -C web add -D jscpd`), `pnpm -C web exec jscpd --version` verifizieren
- [x] 1.3 `metrics/REPORT.md` in `.gitignore` aufnehmen (Verzeichnis `metrics/` mit `.gitkeep` für `thresholds.yml`)

## 2. Komplexitäts-Config (getrennt vom Gate)

- [x] 2.1 `.golangci.metrics.yml` anlegen: nur `gocyclo`, `gocognit`, `funlen`, `dupl` aktivieren, sinnvolle Report-Schwellwerte, ansonsten alle anderen Linter aus
- [x] 2.2 Verifizieren, dass `golangci-lint run -c .golangci.metrics.yml --issues-exit-code 0 ./...` Komplexitäts-Issues als JSON liefert und Exit 0 zurückgibt
- [x] 2.3 Sicherstellen, dass `.golangci.yml` (Haupt-Gate) inhaltlich unverändert bleibt — `make lint` weiterhin grün

## 3. Orchestrierungs-Tool (Go)

- [x] 3.1 Go-Tool-Gerüst anlegen (cmd-Subcommand `teamwerk metrics` bevorzugt; sonst `scripts/metrics.go`) mit Tool-Verfügbarkeitsprüfung + klarer Fehlermeldung samt Installbefehl
- [x] 3.2 Größe erheben: `scc --format json` parsen → LOC/Sprache, Prod-vs-Test-Ratio, Kommentar-%
- [x] 3.3 Go-Komplexität erheben: JSON aus `.golangci.metrics.yml`-Lauf parsen → zyklomatisch/kognitiv, `funlen`-Verstöße, Top-Hotspot-Funktionen
- [x] 3.4 Coverage erheben: Go (`go test -cover`/`go tool cover`) + Frontend (`vitest run --coverage`, Summary parsen) — getrennt ausweisen
- [x] 3.5 Lint-Dichte erheben: Issue-Count aus Haupt-`.golangci.yml` ÷ kLOC
- [x] 3.6 Duplikation erheben: Go (`dupl` aus 2.1) + Frontend (`jscpd --reporters json`)
- [x] 3.7 stdout-Tabelle rendern (kompakt, lesbar)
- [x] 3.8 `metrics/REPORT.md` rendern (Abschnitte: Größe, Komplexität, Coverage, Lint-Dichte, Duplikation, Top-Hotspots; Datum + Git-Hash im Header)

## 4. Make-Targets

- [x] 4.1 `make metrics` ergänzen (mit `## …`-Hilfetext) — ruft das Go-Tool im Report-Modus, immer Exit 0
- [x] 4.2 `metrics/thresholds.yml` anlegen — nach einem ersten echten `make metrics`-Lauf Startwerte knapp oberhalb Ist-Zustand setzen (Ratchet)
- [x] 4.3 `make metrics-gate` ergänzen — Erhebung + Vergleich gegen `thresholds.yml`, Exit 1 bei Verletzung mit Ausgabe der verletzten Werte

## 5. Tests & Verifikation

- [x] 5.1 Tests für das Go-Tool: JSON-Parsing der Tool-Ausgaben (Fixtures) + Threshold-Vergleichslogik (grün/rot)
- [x] 5.2 Verifizieren: `make metrics` → Exit 0 trotz vorhandener Hotspots; `metrics/REPORT.md` enthält alle Kennzahlen-Kategorien
- [x] 5.3 Verifizieren: `make metrics-gate` failt bei künstlich verschärftem Schwellwert (Exit 1) und ist grün mit Ist-Werten (Exit 0)
- [x] 5.4 Architektur-Test (`internal/arch`) berücksichtigen, falls neues `internal/`-Package entsteht (Klassifizierung)

## 6. Dokumentation & Abschluss

- [x] 6.1 `Makefile`-Hilfetexte + kurzer Abschnitt in CLAUDE.md/AGENTS.md (Metrics-Target, getrennte Configs, Tool-Verankerung)
- [x] 6.2 `openspec validate code-metrics-target --strict` grün
- [x] 6.3 `/verify-change` ausführen (Build/Test/Lint + Invarianten)
