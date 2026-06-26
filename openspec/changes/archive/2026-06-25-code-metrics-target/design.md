## Context

TeamWERK hat heute `make lint` (blockierendes golangci-lint-Gate) und `make coverage` (nur Go). Es fehlt ein reproduzierbares Werkzeug, das Größe und Qualität über die Zeit sichtbar macht. Backend: ~23k Prod-LOC + ~15,5k Test-LOC über 33 `internal/`-Packages, Handler-Dateien bis 2489 Zeilen (`members/handler.go`). Frontend: ~25k LOC TS/TSX, Seiten bis 1522 Zeilen (`KalenderPage.tsx`).

Bereits vorhanden: `golangci-lint` v2 + `.golangci.yml` (nur Default-Linter aktiv — Komplexitäts-Linter `gocyclo`/`gocognit`/`funlen`/`dupl` sind **aus**), `staticcheck`, `@vitest/coverage-v8` als devDependency. Nicht vorhanden: `scc`, `jscpd`.

Constraints: Go 1.26 (go.mod `tool`-Direktive verfügbar), pnpm statt npm (Hard-Rule), VPS mit 1 GB RAM — Metrik-Tools laufen aber nur lokal/CI, nicht auf dem VPS.

## Goals / Non-Goals

**Goals:**
- Ein wiederkehrendes `make metrics` mit echten Qualitätssignalen (Stufe 2), nie blockierend.
- stdout-Tabelle **und** `metrics/REPORT.md` aus einem Lauf.
- Optionales `make metrics-gate` für späteres CI, getrennt vom Default.
- Reproduzierbar gepinnte Tools über das jeweilige Ökosystem-Manifest.
- Das bestehende blockierende Lint-Gate (`.golangci.yml`) unangetastet lassen.

**Non-Goals:**
- Keine COCOMO-/€-Aufwandsschätzung (Vanity-Metrik).
- Kein blockierendes Default-Gate (Coverage/Qualität bleibt Indikator).
- Keine Git-Churn-/Trend-Historie im Repo (Report ist gitignored; Trend ist späteres, separates Thema).
- Keine neuen Laufzeit-Abhängigkeiten auf dem VPS.

## Decisions

### D1: Stufe 2 — golangci-lint liefert die Komplexitäts-Kennzahlen
Statt separater Binaries (`gocyclo`, `gocognit`) nutzen wir die in golangci-lint **eingebauten** Linter `gocyclo`, `gocognit`, `funlen`, `dupl`. Sie laufen über eine **separate** `.golangci.metrics.yml` mit `--issues-exit-code 0`.
- *Warum:* golangci-lint ist bereits da; ein Tool weniger zu pinnen; Hotspot-Funktionen kommen direkt aus dem Issue-Output.
- *Alternative:* dedizierte `gocyclo`/`gocognit`-Binaries — verworfen (mehr Tools, redundant).
- *Wichtig:* die Haupt-`.golangci.yml` bleibt unverändert, damit `make lint`/pre-push nicht plötzlich an Komplexität scheitern.

### D2: `scc` via `go.mod` `tool`-Direktive, `jscpd` via pnpm
`scc` (LOC/Sprache/Kommentar-%, ein Go-Binary) wird als `go tool` gepinnt (`go get -tool github.com/boyter/scc/v3`, Aufruf `go tool scc`). `jscpd` (Frontend-Duplikation) als pnpm-devDependency, Aufruf via `pnpm exec jscpd` / `pnpm dlx`.
- *Warum:* erfüllt „fest verankern, aber nur pnpm und Co." — jedes Tool über sein natives Manifest, kein `npm`, kein globales `go install`.
- *Alternative:* globales `go install`/`brew` — verworfen (nicht reproduzierbar). Bordmittel `find/wc` — verworfen (kein Kommentar-%, keine Sprach-Aufschlüsselung).

### D3: Orchestrierung als Go-Tool
Ein Go-Programm parst die JSON-Ausgaben (`scc --format json`, `golangci-lint --output.json`, `go tool cover`, vitest-Coverage-Summary, `jscpd --reporters json`) und rendert stdout-Tabelle + `metrics/REPORT.md`.
- *Form:* bevorzugt als cmd-Subcommand `teamwerk metrics` (konsistent mit `migrate`/`gen-vapid`/`create-admin`), alternativ `scripts/metrics.go` via `go run`. Finale Wahl beim Apply, abhängig davon ob das Tool sauber ohne DB/Config-Bootstrap im bestehenden `main.go`-Subcommand-Schalter sitzt.
- *Warum:* typsicheres JSON-Parsing, testbar, passt zum Stack. Bash+awk für JSON+Report wäre fragil.

### D4: Zwei Targets, klare Aufgabenteilung
`make metrics` erhebt + berichtet (immer Exit 0). `make metrics-gate` liest dieselben Kennzahlen + `metrics/thresholds.yml` und failt bei Verletzung (Exit 1). `gate` ruft intern die Erhebung auf, damit es eigenständig in CI nutzbar ist.

### D5: Startwerte für thresholds.yml erst nach erstem Lauf
Die Schwellwerte werden nicht erraten, sondern nach dem ersten realen `make metrics`-Lauf knapp oberhalb des Ist-Zustands gesetzt (Ratchet-Prinzip), damit das Gate sofort grün ist und nur Regression fängt.

## Risks / Trade-offs

- **golangci-lint v2 Output-Format / Linter-Verfügbarkeit** → vor Implementierung `golangci-lint help linters` prüfen; `gocyclo`/`gocognit`/`funlen`/`dupl` sind Standard-Linter in v2, aber Config-Syntax (`linters.settings`) verifizieren.
- **`scc` als `go tool` erhöht die go.mod-Tool-Closure** (Build-Zeit-Deps) → akzeptabel, da Dev-/CI-only, kein VPS-Footprint.
- **Doppelter Coverage-Lauf** (metrics + bestehendes `make coverage`) kostet Zeit → `make metrics` darf `go test -cover` selbst aufrufen; Wiederverwendung des `/tmp`-Profils optional als Optimierung.
- **`dupl`/`jscpd` Schwellwerte sind anfangs verrauscht** → zunächst nur berichten, Gate-Schwellwerte konservativ; via D5 entschärft.
- **`pnpm dlx jscpd` lädt bei jedem Lauf** → daher devDependency (D2), nicht dlx, für reproduzierbare Version und Offline-Fähigkeit.

## Migration Plan

Additive Änderung, keine Migration nötig. Rollback = Targets/Dateien entfernen + `go.mod`-tool/`jscpd`-devDep zurücknehmen. Kein DB-, API- oder Deploy-Einfluss; `make build`/`make deploy` unberührt.

## Open Questions

- Subcommand `teamwerk metrics` vs. `scripts/metrics.go` — beim Apply final entscheiden (siehe D3).
- Lässt sich ein sinnvolles Frontend-Test-Ratio (Test-LOC/Prod-LOC) aus der vitest-Struktur ableiten, oder nur Coverage? → beim Apply prüfen.
- Welche konkreten Startschwellwerte? → erst nach dem ersten Lauf festlegen (D5).
