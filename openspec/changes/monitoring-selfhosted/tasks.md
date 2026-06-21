# Tasks — Anbieter-neutrales Monitoring (Signal-Quelle + austauschbarer Monitor)

> Ein Commit pro Task (Conventional Commits, Scope: `health`, `metrics`, `scheduler`, `log`, `app`, `db`, `docs`).

## 1. Health-Endpoint (`/api/healthz`)
- [ ] 1.1 Migration `004_monitoring_heartbeat` (`.up`/`.down`): `monitoring_heartbeat(id PK CHECK(id=1), updated_at TEXT NOT NULL)`
- [ ] 1.2 Health-Handler (`internal/health/`): DB-Ping, `disk_free_pct` (`syscall.Statfs` auf `DB_PATH`-Dir), `scheduler_age_sec` aus Heartbeat; `200`/`status:"ok"` bzw. `503`/`db:"fail"`; Payload minimal & PII-frei
- [ ] 1.3 Route im **Public-Tier** von `internal/app/router.go` (kein Auth, GET ⇒ kein `Broadcast`)
- [ ] 1.4 Tests: `TestHealthz_OK`, `TestHealthz_DBDown`, `TestHealthz_NoAuthRequired`, `TestHealthz_SchedulerAgeReported`

## 2. Metrik-Schnittstelle (`/api/metrics`, Prometheus-Textformat)
- [ ] 2.1 Prozess-Counter (`teamwerk_panics_total`, `teamwerk_uptime_seconds`-Start) als Paket-State im Health-/Metrics-Package
- [ ] 2.2 Handler erzeugt Prometheus-Textformat: `teamwerk_up`, `teamwerk_db_up`, `teamwerk_disk_free_ratio`, `teamwerk_mem_free_ratio` (Linux, sonst weglassen), `teamwerk_scheduler_age_seconds`, `teamwerk_panics_total`, `teamwerk_uptime_seconds`
- [ ] 2.3 Bearer-Token-Schutz: `METRICS_TOKEN` aus `appconfig`; ungesetzt ⇒ `404`, gesetzt ⇒ `Authorization: Bearer` Pflicht (sonst `401`). Route in `router.go`
- [ ] 2.4 Tests: `TestMetrics_RequiresToken` (404 ohne Token / 401 falsch), `TestMetrics_ExposesSignals` (200 + Format + Schlüssel)

## 3. Scheduler-Heartbeat (reine Datenquelle)
- [ ] 3.1 `scheduler.Run()` schreibt am Ende erfolgreich `INSERT … ON CONFLICT(id) DO UPDATE` auf `monitoring_heartbeat` — **kein** Self-Alert
- [ ] 3.2 Test: `TestScheduler_HeartbeatRecorded`

## 4. Recover-Middleware (Panic → Counter + strukturiertes Log)
- [ ] 4.1 Custom Recover-Middleware (ersetzt `chi.Recoverer` in `router.go`): Panic als `slog`-Record mit `event="panic"` + Stacktrace-Feld loggen, `teamwerk_panics_total++`, Response `500`, Prozess lebt — **keine** Mail/Push aus der App
- [ ] 4.2 Tests: `TestRecover_Panic_IncrementsCounterAndRecovers` (500 + Counter +1 + Server lebt, keine Benachrichtigung), `TestRecover_Panic_StructuredLog` (JSON-Record mit `event="panic"`)

## 5. Strukturiertes Logging (`slog`) — dritte neutrale Schnittstelle
- [ ] 5.1 Zentrale Logger-Init in `cmd/teamwerk/main.go`: `slog.SetDefault(...)`; `LOG_FORMAT=json` (Default) → `JSONHandler(os.Stdout)`, `text` → `TextHandler` (lokale DX); `LOG_FORMAT` in `appconfig`
- [ ] 5.2 `main.go` umstellen: `log.Fatalf` → `slog.Error(...)` + `os.Exit(1)`, `log.Printf` → `slog.Info/Warn`
- [ ] 5.3 Foundation-/übrige Packages durchsweepen (verbleibende `log.*`-Aufrufe → `slog`); `go vet` + Architektur-Test grün, kein neues Package-Coupling
- [ ] 5.4 Test: `TestLogger_EmitsJSON` (Default-Handler schreibt valides JSON mit `level`/`msg`/`time`)

## 6. Referenz-Konsument (außerhalb des App-Repos) & Doku
- [ ] 6.1 mittwald-PHP-Cron-Script dokumentieren (nicht im Repo ablegen): `/api/healthz` pollen, Schwellen (`disk_free_pct`, `scheduler_age_sec`, HTTP/`db`) auswerten, TLS-Cert via `openssl_x509_parse`, Alarm via `mail()`; Cron-Intervall + Einrichtung in mStudio
- [ ] 6.2 Optional: redundanter GitHub-Actions-Workflow `.github/workflows/uptime.yml` als zweites Auge (URL via Repo-Variable, nicht hardcodieren)
- [ ] 6.3 Betriebsdoku: welche Schnittstelle liefert welches Signal (`/healthz`, `/metrics`, slog-Logs), Beispiel-Schwellen, Hinweis „mind. 1 Monitor muss konfiguriert bleiben", Beispiel-Configs (Prometheus-Scrape **und** ein Log-Collector, z. B. Vector/Alloy auf journald)
- [ ] 6.4 `.env.example` + Deploy-Doku/`/etc/teamwerk/env` um `METRICS_TOKEN` und `LOG_FORMAT` ergänzen
- [ ] 6.5 Manuelle Verifikation: erzwungener Fehl-Poll löst Test-Benachrichtigung aus

## 7. Abschluss
- [ ] 7.1 `/verify-change` grün (Build/Test/Lint, Route→Tests, Migrationsnummer, `openspec validate`)
- [ ] 7.2 Proposal archivieren (`openspec archive monitoring-selfhosted`)
