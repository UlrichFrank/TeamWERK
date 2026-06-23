## 1. App-Metriken (Spur B) — Health-Package

- [x] 1.1 `internal/health/health.go`: globaler `atomic.Int64` für `httpInFlight`; globaler `atomic.Int64` für `sqliteBusyTotal`; Public-Funktionen `RecordSQLiteBusy()`, `CheckSQLiteBusy(err) bool` (true bei `errors.Is(err, sqlite.ErrBusy)` + Increment)
- [x] 1.2 `internal/health/middleware.go` (neu): `InFlightMiddleware` mit `httpInFlight.Add(+1)/defer Add(-1)`
- [x] 1.3 `internal/health/health.go`: WAL-Größe per `os.Stat(dbPath + "-wal")` in `metricsHandler` ermitteln (Fehler/Not-Exist → 0); `health.Handler` um `dbPath string`-Feld erweitern und Konstruktor anpassen
- [x] 1.4 `internal/health/health.go`: drei neue `writeMetric`-Aufrufe — `teamwerk_sqlite_wal_bytes` (gauge), `teamwerk_sqlite_busy_total` (counter), `teamwerk_http_requests_in_flight` (gauge)
- [x] 1.5 `internal/app/router.go`: `InFlightMiddleware` vor Recover-Middleware in die Kette einhängen
- [x] 1.6 `cmd/teamwerk/main.go`: `health.NewHandler` mit `cfg.DBPath` aufrufen (Konstruktor-Signatur folgen)

## 2. Scheduler-BUSY-Event (Spur B)

- [x] 2.1 `internal/scheduler/`: an der Stelle, an der `scheduler.Run()` heute DB-Errors behandelt, `errors.Is(err, sqlite.ErrBusy)` prüfen und bei Match `slog.Warn("sqlite_busy", "source", "scheduler", "op", "<context>")` emittieren (zusätzlich zur bestehenden Fehlerbehandlung)

## 3. BUSY-Erfassung am Driver-Layer (Spur B, Null-Touch-Handler)

- [x] 3.1 `internal/db/busy_driver.go` (neu): `busyDriver` registriert sich als `"sqlite-busy-counting"`, delegiert an den existierenden `"sqlite"`-Driver von `modernc.org/sqlite` und ruft am Error-Pfad `health.CheckSQLiteBusy(err)`. Implementiert `driver.Conn` (Prepare, Close, Begin) + die Subinterfaces, die `modernc.org/sqlite` selbst bietet: `ConnPrepareContext`, `ConnBeginTx`, `ExecerContext`, `QueryerContext`, `Pinger`, `SessionResetter`, `Validator`, `NamedValueChecker`. `driver.Stmt` (Close, NumInput, Exec, Query) + `StmtExecContext`, `StmtQueryContext`, `NamedValueChecker`. Build-Time-Assertions (`var _ driver.ExecerContext = (*busyConn)(nil)` etc.) absichern.
- [x] 3.2 `internal/db/db.go`: `sql.Open(...)` von `"sqlite"` auf `"sqlite-busy-counting"` umstellen
- [x] 3.3 `internal/testutil/`: prüfen, ob ein eigener `sql.Open` darin steht — falls ja, ebenfalls auf den neuen Driver umstellen (sonst zählen Tests BUSY nicht)

## 4. Tests (Spur B)

- [x] 4.1 `TestMetrics_ExposesSQLiteWALBytes` in `internal/health/health_test.go`: Antwort enthält `teamwerk_sqlite_wal_bytes` (Wert ≥ 0); zweiter Sub-Case: ohne `-wal`-Datei → Wert `0`
- [x] 4.2 `TestMetrics_SQLiteBusyCounterIncrements` in `internal/health/health_test.go`: vor/nach `RecordSQLiteBusy()` (oder erzwungenem BUSY) ist der Counter um genau 1 höher; Format korrekt
- [x] 4.3 `TestMetrics_InFlightRequestsTracked` in `internal/health/health_test.go`: Test mit Handler, der innerhalb der Middleware den Gauge-Wert via Sidechannel ausliest und bestätigt, dass `httpInFlight ≥ 1` während des Requests ist; nach Abschluss zurück auf vorherigen Wert
- [x] 4.4 `TestScheduler_SQLiteBusyEmitsLog` in `internal/scheduler/`: mit injiziertem Logger (`slog.New(JSONHandler(buf))`) sicherstellen, dass bei simuliertem BUSY-Error ein Log-Record mit `event="sqlite_busy"` und `source="scheduler"` entsteht

## 5. Vector-Pipeline (Spur A) — `setup-vps.sh` & Token

- [x] 5.1 `deploy/setup-vps.sh`: Token-File `/etc/teamwerk/betterstack-metrics-token` analog zum bestehenden Logs-Token-Block anlegen (chmod 600, Placeholder `REPLACE_WITH_BETTERSTACK_METRICS_TOKEN`)
- [x] 5.2 `deploy/setup-vps.sh`: `METRICS_TOKEN` in `/etc/teamwerk/env` automatisch via `openssl rand -hex 32` erzeugen, falls leer/unset (idempotent; Warnung in der Schlussübersicht, dass der Token in Vector eingetragen werden muss)
- [x] 5.3 `deploy/setup-vps.sh`: `vector.toml`-Heredoc um `[sources.host] type = "host_metrics" collectors = ["cpu","memory","disk","network","filesystem","load"] scrape_interval_secs = 30` erweitern
- [x] 5.4 `deploy/setup-vps.sh`: `vector.toml`-Heredoc um `[sources.teamwerk_app] type = "prometheus_scrape" endpoints = ["http://127.0.0.1:8080/api/metrics"] scrape_interval_secs = 30 auth.strategy = "bearer" auth.token = "<METRICS_TOKEN>"` erweitern
- [x] 5.5 `deploy/setup-vps.sh`: zweiten Sink `[sinks.betterstack_metrics]` mit `inputs = ["host","teamwerk_app"]` und Better-Stack-Telemetry-Token aus Token-File hinzufügen; Schlussübersicht erweitern
- [x] 5.6 `setup-vps.sh` lokal mit `bash -n` (Syntax) und Trockenlauf gegen Fixture/Container prüfen

## 6. Runbook + Doku (Spur A)

- [x] 6.1 `deploy/vps-setup-runbook.md`: Voraussetzungs-Tabelle um Zeile "Better Stack Metrics-Source | telemetry.betterstack.com | Vector → Metrics-Pipeline" ergänzen; Schritt 0/3 anpassen (zweites Token besorgen, Token-File befüllen); Schritt 7 erweitern (Vector startet jetzt zwei Sources + zwei Sinks)
- [x] 6.2 `docs/monitoring.md`: neue Sektion "Host- und App-Metriken via Vector-Pipeline" — Tabelle der zusätzlichen Signale (`teamwerk_sqlite_wal_bytes`, `teamwerk_sqlite_busy_total`, `teamwerk_http_requests_in_flight`, Host-Metriken), Beispiel-Schwellen (z. B. `teamwerk_sqlite_wal_bytes > 50 MB`, `rate(teamwerk_sqlite_busy_total[5m]) > 0.05`)
- [x] 6.3 `docs/monitoring.md`: Hinweis, dass Better-Stack-Telemetry-Source-Konfiguration aus dem dortigen UI gezogen wird (keine fixe Snippet-Übernahme); Verlinkung zum Vector-Sink-Block im Setup-Script

## 7. Verifikation & Abschluss

- [x] 7.1 `go build ./...`, `go vet ./...`, `go test -race ./...` grün (745 tests, 38 packages); `golangci-lint run ./internal/health/... ./internal/scheduler/... ./internal/db/...` 0 Findings; `gofmt -l` clean
- [x] 7.2 `openspec validate monitoring-host-and-sqlite-metrics --strict` grün
- [x] 7.3 `/verify-change` durchlaufen (Build/Test/Lint + Projekt-Invarianten)
- [x] 7.4 Live-Verifikation nach Deploy (manuell, in Doku festhalten): (a) Vector startet ohne Fehler (`journalctl -u vector`), (b) `host_metrics` füllen Better-Stack-Charts, (c) `teamwerk_*`-Metriken erscheinen unter Vector-Source `teamwerk_app`, (d) künstlicher BUSY-Test (paralleler Schreibhammer) hebt `teamwerk_sqlite_busy_total` sichtbar
- [x] 7.5 Proposal archivieren (`openspec archive monitoring-host-and-sqlite-metrics`) — nach Live-Verifikation
