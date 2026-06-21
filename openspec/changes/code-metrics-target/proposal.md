## Why

Es gibt aktuell kein wiederkehrendes Werkzeug, um die Code-Qualität von TeamWERK an typischen Kennzahlen zu bewerten — vorhanden sind nur `make lint` (blockierendes Gate) und `make coverage` (nur Go). Größe und Komplexität (Backend ~23k Prod-LOC in Handler-Dateien bis 2489 Zeilen, Frontend ~25k LOC) wachsen unbeobachtet. Ein reproduzierbares `make metrics`-Target macht Qualität sichtbar und über die Zeit vergleichbar, ohne die Entwickler-Workflows zu blockieren.

## What Changes

- Neues Make-Target **`make metrics`** — reines Reporting (nie ein Fehlercode), erzeugt eine kompakte stdout-Tabelle **und** `metrics/REPORT.md`.
- Gemessen werden (Stufe 2 — echte Qualitätssignale, nicht nur Größe):
  - **Größe**: LOC/Sprache, Prod-vs-Test-Verhältnis, Kommentar-Anteil via `scc`.
  - **Go-Komplexität**: zyklomatische & kognitive Komplexität, Funktionslänge, Duplikation via `golangci-lint` mit separater `.golangci.metrics.yml` (`gocyclo`, `gocognit`, `funlen`, `dupl`, `--issues-exit-code 0`).
  - **Coverage**: Go (`go test -cover`) + Frontend (`vitest --coverage`, `@vitest/coverage-v8` ist bereits installiert).
  - **Lint-Dichte**: Issues/kLOC aus der bestehenden Haupt-`.golangci.yml`.
  - **Frontend-Duplikation**: via `jscpd`.
  - **Top-Hotspots**: die komplexesten Go-Funktionen als Liste.
- Neues opt-in Target **`make metrics-gate`** — liest `metrics/thresholds.yml` und beendet sich mit Exit-Code 1 bei Schwellwert-Regression (für späteres CI). Der Default-Lauf bleibt reines Reporting.
- Orchestrierung als **Go-Tool** (cmd-Subcommand `teamwerk metrics` oder `scripts/metrics.go`), das die Tool-Ausgaben (JSON) parst und den Report baut.
- **Tooling-Verankerung**: `jscpd` als pnpm-devDependency; `scc` als `go.mod`-`tool`-Direktive (Go 1.26). Kein `npm`, kein globales `go install`.
- `metrics/REPORT.md` wird **gitignored** (generiertes Artefakt).

**Nicht-Ziele:** keine COCOMO-/€-Schätzung (Vanity), kein blockierendes Default-Gate, keine Änderung an der bestehenden `.golangci.yml` (Gate bleibt unangetastet).

## Capabilities

### New Capabilities
- `code-metrics`: Ein wiederkehrend ausführbares Werkzeug, das Code-Größe, -Komplexität, Test-Coverage, Duplikation und Lint-Dichte für Go-Backend und TS/React-Frontend erhebt, als stdout-Tabelle und Markdown-Report ausgibt und optional gegen konfigurierte Schwellwerte prüft.

### Modified Capabilities
<!-- Keine bestehenden Capabilities ändern ihre Anforderungen. -->

## Impact

- **Neue Dateien**: Go-Orchestrierungs-Tool (cmd-Subcommand bzw. `scripts/metrics.go`), `.golangci.metrics.yml`, `metrics/thresholds.yml`.
- **Generiert (gitignored)**: `metrics/REPORT.md`.
- **Geänderte Dateien**: `Makefile` (Targets `metrics`, `metrics-gate`), `go.mod`/`go.sum` (`tool`-Direktive für `scc`), `web/package.json`/`pnpm-lock.yaml` (`jscpd`), `.gitignore`.
- **Unangetastet**: `.golangci.yml` (blockierendes Gate), bestehende Targets `lint`/`test`/`coverage`.
- **Abhängigkeiten**: `scc` (Go-Binary via `go tool`), `jscpd` (pnpm). Beide nur Build-/Dev-Zeit, kein Laufzeit-Footprint auf dem VPS.
- **Keine** neuen Laufzeit-Dienste, keine API-, DB- oder Berechtigungs-Änderungen.
