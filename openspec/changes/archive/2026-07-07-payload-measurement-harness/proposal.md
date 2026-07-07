## Why

Drei geplante Changes (`efficient-data-loading-quickwins`, `list-endpoint-pagination`, `scoped-live-updates`) sollen den Datenverkehr zum Client senken. Ob sie wirken, ist heute **nicht messbar**: `make metrics`/`code-metrics` erhebt Größe/Komplexität/Coverage/Duplikation (Code-Metriken), `client-telemetry` zählt Matomo-Pageviews, `production-monitoring` misst in-flight Requests und WAL/BUSY — **keine** dieser Quellen erfasst Payload-Bytes pro Route, 304-Quote oder das SSE-Fan-out-Volumen pro Mutation.

Ohne Baseline-Zahl vor der Umsetzung gibt es später keinen belastbaren Vorher/Nachher-Vergleich. Dieser Change baut das Messwerkzeug **zuerst** und friert eine Baseline auf dem aktuellen `main` ein, an die sich die drei Optimierungs-Changes mit ihren Nachher-Zahlen hängen.

Besonders wichtig: Der größte erwartete Gewinn (`scoped-live-updates`) ist in einer Einzel-Request-Payload-Messung **unsichtbar** — er entsteht erst im Produkt `verbundene Clients × Mutationsrate`. Das Werkzeug muss daher ein **Fan-out-Szenario** abbilden (mehrere SSE-Clients, eine Mutation, gelieferte Events/Bytes zählen), nicht nur Content-Length pro Route.

## What Changes

- **Reproduzierbares Mess-Werkzeug** als Go-Integrationstest, der `testutil.NewServer` + die bestehenden Fixtures (`CreateMember`, `CreateGame`, `CreateDutySlot`, …) nutzt, um einen **realistischen, deterministischen** Datensatz zu seeden (feste Größen, keine Zufallswerte/kein `time.Now()` in den Assertions — nur in der Report-Kopfzeile).
- **Payload-Messung pro schwerer Route:** ruft die relevanten GET-Endpoints via `httptest` auf und erfasst die Response-Größe (`Content-Length`/`len(body)`) sowie den HTTP-Status. Ergebnis: Tabelle Route → Bytes.
- **304/Cache-Messung:** ruft Referenzrouten zweimal auf (zweiter Call mit `If-None-Match`) und erfasst Status + Bytes des zweiten Calls (misst die Wirkung von `reference-data-caching`, sobald umgesetzt; auf `main` erwartbar 200 + volle Bytes).
- **SSE-Fan-out-Messung:** abonniert einen **festen Satz von M = 8 benannten Clients** (bekannte Funktion/Team, siehe `design.md`) an `/api/events`, löst je Messung genau **eine** feste Mutation aus (`members` / `games(T1)` / `settings`) und zählt je Client die gelieferten Events/Bytes. Auf `main` erwartbar `8 / 8 / 8` (globaler Fan-out); nach `scoped-live-updates` `3 / 5 / 8`.
- **Report-Ausgabe** analog `make metrics`: neues Target `make measure` schreibt `metrics/PAYLOAD.md` (gitignored). Eine committete **Baseline** (`metrics/payload-baseline.md`) hält den `main`-Stand fest; die Optimierungs-Changes aktualisieren jeweils ihre Zeile.
- **Optionaler Ratchet:** eine Schwellwert-Datei (analog `metrics/thresholds.yml`), gegen die `make measure-gate` prüft — Payload/Fan-out darf nicht regredieren. Bewusst zunächst **nicht** ins blockierende Gate (`pre-push`) gehängt, um Flakiness zu vermeiden; separates, freiwilliges Target.

## Capabilities

### Added Capabilities

- `payload-measurement`: reproduzierbares Werkzeug zur Messung von Response-Payload pro Route, 304-Verhalten und SSE-Fan-out-Volumen pro Mutation, mit committeter Baseline.

## Test-Anforderungen

| Mechanismus | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| Seeding | `TestMeasure_SeedIsDeterministic` | Zwei Läufe mit demselben Seed erzeugen dieselben Datensatz-Größen (kein `Math.rand`/`time.Now` im Datensatz). |
| Payload | `TestMeasure_RecordsPayloadPerRoute` | Für jede konfigurierte Route wird ein Bytes-Wert > 0 und der HTTP-Status erfasst. |
| Fan-out | `TestMeasure_SSEFanoutCountsPerClient` | Bei den 8 festen Clients und einer global gebroadcasteten Mutation zählt das Werkzeug 8 zugestellte Events. |
| Fan-out-Roster | `TestMeasure_FanoutClientRosterIsFixed` | Die 8 Mess-Clients haben die in `design.md` festgelegten Funktionen/Teams (deterministisch). |
| Report | `TestMeasure_WritesReport` | `make measure` erzeugt `metrics/PAYLOAD.md` mit Routen-Tabelle + Fan-out-Zeile; Exit 0. |

**Garantierte Invariante:** Das Werkzeug **beobachtet nur** — es verändert keine Produktions-Code-Pfade und keine Auth-/Sichtbarkeitsregeln. Die Messung nutzt ausschließlich öffentliche HTTP-Endpoints über `testutil.NewServer`, keine internen Shortcuts, damit die Zahlen dem realen Client-Erlebnis entsprechen.

## Impact

- **Neu:** `internal/measure/measure_test.go` (Mess-Szenarien) — nutzt `testutil.NewServer`; `internal/arch/arch_test.go` um Klassifizierung von `internal/measure` ergänzen (Test-/Composition-Support, darf `app` + `testutil` importieren, keine Domain↔Domain-Neuverdrahtung).
- **Neu:** `Makefile`-Targets `measure` (Report) und optional `measure-gate` (Schwellwert); `metrics/payload-baseline.md` (committet) + `metrics/PAYLOAD.md` (gitignored, `.gitignore` ergänzen); optional `metrics/payload-thresholds.yml`.
- **Kein** Produktions-Code-Change, **kein** Schema-/Migrations-Change, **keine** neue Route, **kein** neues NPM-Paket. Ausschließlich Test-/Tooling-Ebene.
- **Reihenfolge:** Dieser Change kommt **vor** `efficient-data-loading-quickwins` / `list-endpoint-pagination` / `scoped-live-updates`, damit deren `## Mess-Anforderungen` gegen eine echte Baseline laufen.
- **Grenze der Aussagekraft:** Die Baseline ist ein synthetischer, fixer Datensatz — sie zeigt **relative** Verbesserung zuverlässig, ersetzt aber keine reale Traffic-Messung (dafür wäre `bytes_out` im Access-Log via `production-monitoring` der Weg). Bewusst als „Regressions-/Wirkungsnachweis im CI", nicht als Produktions-Kennzahl.
