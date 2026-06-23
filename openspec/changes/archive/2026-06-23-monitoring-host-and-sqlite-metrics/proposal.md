## Why

Mit wachsender Nutzerzahl rücken zwei Risikoklassen ins Blickfeld, die das bestehende Tier-Modell (`monitoring-selfhosted`) nur indirekt abdeckt:

- **Host-Sättigung** auf der 1-GB-VPS (CPU/Disk/Network/Swap/Memory-Druck): das in Better Stack vorgesehene "Host(Vector)"-Dashboard zeigt heute leere Standard-Charts, weil keine Host-Telemetrie geliefert wird.
- **SQLite-Schreibkonkurrenz** (WAL-Wachstum, `SQLITE_BUSY`): bisher unsichtbar, kann aber das erste Wachstums-Bottleneck werden.

Der bereits installierte Vector-Prozess (heute nur Log-Shipper) wird zur Pipeline für beides erweitert. Die Architektur-Doktrin bleibt: die App ist Signal-Quelle (Pull über `/api/metrics` + `slog`-JSON), Vector ist die vendor-neutrale Pipeline, Better Stack ist der austauschbare Sink.

## What Changes

**Spur A — Vector-Pipeline-Erweiterung (Deploy/Doku, kein App-Code)**

- `host_metrics`-Source in `vector.toml` (CPU, Memory, Disk, Network, Swap, Filesystem)
- `prometheus_scrape`-Source gegen `http://localhost:8080/api/metrics` mit Bearer `METRICS_TOKEN`
- Neuer Sink "Better Stack Metrics" parallel zum bestehenden Logs-Sink; Token-File `/etc/teamwerk/betterstack-metrics-token` analog zum Logs-Token
- `deploy/setup-vps.sh`, `deploy/vps-setup-runbook.md` und `docs/monitoring.md` angepasst

**Spur B — App-Metriken-Erweiterung (kleiner Go-Eingriff)**

- `teamwerk_sqlite_wal_bytes` (gauge): Größe der `*-wal`-Datei (0 falls nicht vorhanden)
- `teamwerk_sqlite_busy_total` (counter): zählt `SQLITE_BUSY`-Returns im HTTP-Pfad über eine Middleware (γ-Wrapping um DB-Aufrufe pro Request). Scheduler-Pfad emittiert stattdessen ein `slog.Warn("sqlite_busy", source=scheduler)`-Event — log-basiert alarmierbar ohne Cross-Prozess-Counter-Persistenz.
- `teamwerk_http_requests_in_flight` (gauge): atomic-Counter via HTTP-Middleware (~15-Zeilen-Diff, ~50 ns je Request)

**Bewusste Nicht-Ziele**

- Kein Histogram-Latency-Tracking (Cardinality-Druck auf 1-GB-Box)
- Kein Per-Route-Counter (gleicher Grund)
- Keine Alerts im App-Code (Doktrin: Alerting lebt im austauschbaren Monitor)
- Keine zweite Tabelle zur Cross-Prozess-Counter-Synchronisation (Scheduler-BUSY läuft über strukturierte Logs)

## Capabilities

### New Capabilities

*(keine)*

### Modified Capabilities

- `production-monitoring`: erweitert um zwei zusätzliche Signal-Klassen — Host-Telemetrie (über Vector-Pipeline, außerhalb des App-Codes) und SQLite-spezifische App-Metriken (über `/api/metrics`). Pull-Architektur und Anbieter-Neutralität bleiben unverändert; keine Bestehende-Anforderung wird zurückgenommen.

## Impact

- **App-Code:** `internal/health/health.go` (3 neue Metriken in Prometheus-Output), `internal/health/middleware.go` neu (in-flight + sqlite-busy-Wrapper), `internal/scheduler/` (1 `slog.Warn`-Call im SQLite-BUSY-Error-Pfad).
- **Tests:** 4 neue Tests (`TestMetrics_ExposesSQLiteWALBytes`, `TestMetrics_SQLiteBusyCounterIncrements`, `TestMetrics_InFlightRequestsTracked`, `TestScheduler_SQLiteBusyEmitsLog`) in den jeweiligen Packages.
- **Router:** `internal/app/router.go` hängt die in-flight- und busy-Middleware in die HTTP-Kette ein (vor `chi.Recoverer` bzw. der projektspezifischen Recover-Middleware).
- **Deploy:** `deploy/setup-vps.sh` (Vector-TOML-Block um zwei Sources und einen Metrics-Sink erweitert, neues Token-File `/etc/teamwerk/betterstack-metrics-token`), `deploy/vps-setup-runbook.md` (Voraussetzungs-Tabelle + Schritte 0/3/7 angepasst).
- **Doku:** `docs/monitoring.md` neue Sektion "Host- und App-Metriken via Vector-Pipeline" inkl. Beispiel-Schwellen für die neuen Metriken.
- **Externe Aktion** (vor Deploy, kein Repo-Inhalt): in Better Stack eine **Metrics-Source** vom Typ Vector anlegen, Token in `/etc/teamwerk/betterstack-metrics-token` eintragen.
- **Voraussetzung:** `METRICS_TOKEN` muss auf dem VPS gesetzt sein (sonst antwortet `/api/metrics` mit `404` und Vector scrapet ins Leere).
- **Frontend:** nicht betroffen.
- **RAM-Impact:** Vector-Erweiterung minimal (`host_metrics` + `prometheus_scrape` sind leichte Sources); App-Metriken vernachlässigbar (atomics + ein WAL-`os.Stat`).
- **Keine neuen externen Dienste** — Vector und Better Stack sind bereits in Betrieb; nur Konfiguration ändert sich.
