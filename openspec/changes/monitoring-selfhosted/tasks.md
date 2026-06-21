# Tasks — Anbieter-neutrales Monitoring (Signal-Quelle + austauschbarer Monitor)

> Ein Commit pro Task (Conventional Commits, Scope: `health`, `metrics`, `scheduler`, `log`, `app`, `db`, `docs`).

## 1. Health-Endpoint (`/api/healthz`)
- [x] 1.1 Migration `005_monitoring_heartbeat` (`.up`/`.down`): `monitoring_heartbeat(id PK CHECK(id=1), updated_at TEXT NOT NULL)`
- [x] 1.2 Health-Handler (`internal/health/`): DB-Ping, `disk_free_pct` (`syscall.Statfs` auf `DB_PATH`-Dir), `scheduler_age_sec` aus Heartbeat; `200`/`status:"ok"` bzw. `503`/`db:"fail"`; Payload minimal & PII-frei
- [x] 1.3 Route im **Public-Tier** von `internal/app/router.go` (kein Auth, GET ⇒ kein `Broadcast`)
- [x] 1.4 Tests: `TestHealthz_OK`, `TestHealthz_DBDown`, `TestHealthz_NoAuthRequired`, `TestHealthz_SchedulerAgeReported`

## 2. Metrik-Schnittstelle (`/api/metrics`, Prometheus-Textformat)
- [x] 2.1 Prozess-Counter (`teamwerk_panics_total`, `teamwerk_uptime_seconds`-Start) als Paket-State im Health-/Metrics-Package
- [x] 2.2 Handler erzeugt Prometheus-Textformat: `teamwerk_up`, `teamwerk_db_up`, `teamwerk_disk_free_ratio`, `teamwerk_mem_free_ratio` (Linux, sonst weglassen), `teamwerk_scheduler_age_seconds`, `teamwerk_panics_total`, `teamwerk_uptime_seconds`
- [x] 2.3 Bearer-Token-Schutz: `METRICS_TOKEN` aus `appconfig`; ungesetzt ⇒ `404`, gesetzt ⇒ `Authorization: Bearer` Pflicht (sonst `401`). Route in `router.go`
- [x] 2.4 Tests: `TestMetrics_RequiresToken` (404 ohne Token / 401 falsch), `TestMetrics_ExposesSignals` (200 + Format + Schlüssel)

## 3. Scheduler-Heartbeat (reine Datenquelle)
- [x] 3.1 `scheduler.Run()` schreibt am Ende erfolgreich `INSERT … ON CONFLICT(id) DO UPDATE` auf `monitoring_heartbeat` — **kein** Self-Alert
- [x] 3.2 Test: `TestScheduler_HeartbeatRecorded`

## 4. Recover-Middleware (Panic → Counter + strukturiertes Log)
- [x] 4.1 Custom Recover-Middleware (ersetzt `chi.Recoverer` in `router.go`): Panic als `slog`-Record mit `event="panic"` + Stacktrace-Feld loggen, `teamwerk_panics_total++`, Response `500`, Prozess lebt — **keine** Mail/Push aus der App
- [x] 4.2 Tests: `TestRecover_Panic_IncrementsCounterAndRecovers` (500 + Counter +1 + Server lebt, keine Benachrichtigung), `TestRecover_Panic_StructuredLog` (JSON-Record mit `event="panic"`)

## 5. Strukturiertes Logging (`slog`) — dritte neutrale Schnittstelle
- [x] 5.1 Zentrale Logger-Init in `cmd/teamwerk/main.go`: `slog.SetDefault(...)`; `LOG_FORMAT=json` (Default) → `JSONHandler(os.Stdout)`, `text` → `TextHandler` (lokale DX); `LOG_FORMAT` in `appconfig`
- [x] 5.2 `main.go` umstellen: `log.Fatalf` → `fatal()` (slog.Error + os.Exit(1)), `log.Printf` → `slog.Info`
- [x] 5.3 Foundation-/übrige Packages durchsweepen (alle `log.*`-Aufrufe → `slog`, 7 Dateien); `go vet` + Architektur-Test grün, kein neues Package-Coupling
- [x] 5.4 Test: `TestLogger_EmitsJSON` (Default-Handler schreibt valides JSON mit `level`/`msg`/`time`) + `TestLogger_TextFormatNotJSON`

## 6. Referenz-Konsument (außerhalb des App-Repos) & Doku
- [x] 6.1 mittwald-PHP-Cron-Script dokumentiert (in `docs/monitoring.md`, nicht ausführbar im Repo): `/api/healthz` pollen, Schwellen auswerten, TLS-Cert via `openssl_x509_parse`, Alarm via `mail()`; mStudio-Cron-Einrichtung
- [x] 6.2 Redundanter GitHub-Actions-Workflow `.github/workflows/uptime.yml` (URL via Repo-Variable `HEALTHZ_URL`)
- [x] 6.3 Betriebsdoku `docs/monitoring.md`: Signal→Schnittstelle-Tabelle, Beispiel-Schwellen, „mind. 1 Monitor"-Hinweis, Prometheus-Scrape- **und** Vector-journald-Beispiel
- [x] 6.4 `.env.example` um `METRICS_TOKEN` und `LOG_FORMAT` ergänzt
- [ ] 6.5 Manuelle Verifikation (erzwungener Fehl-Poll → Test-Benachrichtigung) — **erst nach Deploy** des neuen Builds auf dem VPS durchführbar; Schritte in `docs/monitoring.md` dokumentiert

## 7. Abschluss
- [x] 7.1 Verifikation grün: `go build`/`go vet` sauber, `go test -race ./...` 708 passed, `gofmt` clean, `openspec validate --strict` ok, `golangci-lint` für geänderten Code clean (4 Findings sind Baseline-Altlast in `cmd/teamwerk/main.go`/`internal/config/handler.go`, unverändert seit `HEAD~6`)
- [ ] 7.2 Proposal archivieren (`openspec archive monitoring-selfhosted`) — **erst nach Merge + Deploy** (Task 6.5 Live-Verifikation steht noch aus)
