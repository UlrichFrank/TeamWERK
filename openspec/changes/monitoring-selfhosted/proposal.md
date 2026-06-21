## Why

Das Produktivsystem (`https://internal.team-stuttgart.org`, IONOS VPS Linux XS, 1 GB RAM) läuft **ohne jedes Monitoring**. Ausfälle — Prozess-Crash, gesperrte SQLite-DB, volle Disk, stillschweigend gestorbener `scheduler:run`-Cronjob, abgelaufenes TLS-Zertifikat — werden erst bemerkt, wenn Nutzer sich beschweren.

Randbedingungen:

- **1 GB RAM** — ein selbstgehosteter Monitoring-Stack auf derselben Box verbietet sich.
- **Kein Budget** — keine kostenpflichtige SaaS.
- **Physik** — ein Monitor *auf* der Box kann nicht melden, dass die Box tot ist. Ein Auge **außerhalb** des VPS ist zwingend.
- **Agnostik (Leitprinzip dieses Changes)** — die App darf sich an **kein** Monitoring-System binden, und das Monitoring-System muss **jederzeit austauschbar** sein (mittwald-Cron heute, Better Stack / Grafana Cloud / UptimeRobot / Prometheus morgen — ohne App-Änderung).

Daraus folgt die Architektur-Entscheidung: **Die App ist reine Signal-Quelle über standardisierte, Pull-basierte Schnittstellen. Auswertung, Schwellwerte und Alarmierung leben ausschließlich im externen, austauschbaren Monitor.** Es gibt bewusst **kein** in die App eingebautes, anbieter-spezifisches Alerting — das wäre selbst ein nicht-austauschbares Monitoring-System.

Die App committet sich damit nur auf **drei De-facto-Standards**, die jedes Monitoring-Tool spricht:
1. einen HTTP-**Health-Endpoint** (Liveness/Readiness, 200/503) — für jeden Uptime-Checker,
2. eine **Prometheus-Textformat-Metrik-Schnittstelle** — für jeden Metrik-Scraper,
3. **strukturierte JSON-Logs** nach stdout — für jeden Log-Collector.

Auch hier gilt Pull/Trennung: Die App **schreibt** nur (nach stdout→journald) und **shippt** nichts selbst — welcher Collector die Logs einsammelt, ist austauschbare Betriebskonfiguration, kein App-Code.

Als *Referenz*-Konsument (nicht Teil des App-Codes, beliebig ersetzbar) dient ein **Cron auf dem bestehenden mittwald-Webhosting**: eigener Anbieter (andere Failure-Domain als der VPS), bereits bezahlt, punktgenau, Script außerhalb des öffentlichen Repos.

**Bewusst in Kauf genommen:** Da die App nicht selbst alarmiert, muss **mindestens ein externer Monitor konfiguriert sein**, sonst sind die Signale zwar vorhanden, aber niemand schaut hin. Das ist der Preis der Agnostik — und gewollt.

## What Changes

- **`GET /api/healthz`** (Public-Tier, ohne Auth): standardisierte Liveness/Readiness-Prüfung. `200`/`status:"ok"` bzw. `503` bei DB-Fehler. Body trägt grobe, **nicht-sensible** Signale für simple Checker (`db`, `disk_free_pct`, `scheduler_age_sec`). Kein PII, keine internen Pfade/Versionen.
- **`GET /api/metrics`** im **Prometheus-Textformat**, per Bearer-Token geschützt (`METRICS_TOKEN`; ungesetzt ⇒ Endpoint deaktiviert / `404`). Exponiert `teamwerk_up`, `teamwerk_db_up`, `teamwerk_disk_free_ratio`, `teamwerk_mem_free_ratio` (Linux), `teamwerk_scheduler_age_seconds`, `teamwerk_panics_total`, `teamwerk_uptime_seconds`. Jeder Scraper (Prometheus, Grafana Agent, Netdata, Better-Stack-/Datadog-Collector …) kann das ohne App-Anpassung lesen.
- **Migration `004`**: Single-Row-Tabelle `monitoring_heartbeat` für den Zeitstempel des letzten erfolgreichen Scheduler-Laufs.
- **Scheduler-Heartbeat** (`scheduler.Run()`): schreibt nach erfolgreichem Lauf den Heartbeat — **reine Datenquelle**, kein Self-Alert. Der Dead-Man-Switch entsteht extern aus `scheduler_age_seconds`.
- **Custom Recover-Middleware** ersetzt `chi.Recoverer`: bei Panic Stacktrace strukturiert loggen + `teamwerk_panics_total` inkrementieren, Response `500`, Prozess lebt weiter. **Keine** Mail/Push aus der App heraus — der Counter ist das (agnostische) Signal.
- **Strukturiertes Logging (`slog`)**: Umstellung von stdlib `log` auf `slog` mit JSON-Handler nach stdout (`LOG_FORMAT=json|text`, Default `json` in Prod, `text` lokal). Panics und relevante Ereignisse werden zu maschinenlesbaren Log-Records mit stabilen Feldern (z. B. `event=panic`) — dritte neutrale Schnittstelle für beliebige Log-Collector, die App shippt selbst nicht.
- **Referenz-Konsument (nicht im App-Repo):** mittwald-PHP-Cron, der `/api/healthz` (+ optional `/api/metrics`) pollt, Schwellen auswertet, TLS-Cert prüft und bei Verletzung via `mail()` alarmiert. Beliebig durch ein anderes System ersetzbar; optional zusätzlich ein GitHub-Actions-Workflow als Redundanz.

## Capabilities

### New Capabilities

- `production-monitoring`: Garantien darüber, dass die App ihren Betriebszustand (Erreichbarkeit, DB, Disk/Speicher, Scheduler-Lebendigkeit, Panic-Aufkommen) über **standardisierte, anbieter-neutrale Pull-Schnittstellen** (`/api/healthz`, `/api/metrics`) bereitstellt — sodass Auswertung und Alarmierung von einem **frei austauschbaren** externen Monitor übernommen werden, ohne Bindung der App an einen konkreten Anbieter.

### Modified Capabilities

*(keine)*

## Impact

- **Zwei neue Routen** (`GET /api/healthz` public, `GET /api/metrics` token-geschützt) in `internal/app/router.go`; beide GET ⇒ kein `Broadcast`.
- **Migration `004`** (`004_monitoring_heartbeat.up.sql`/`.down.sql`).
- **`internal/scheduler/`** erweitert (Heartbeat-Schreiben). **`internal/health/`** (oder vergleichbar) neuer Handler für healthz + metrics + Panic-Counter.
- **Middleware-Tausch** in `router.go` (`chi.Recoverer` → eigene Recover-Middleware mit Counter + strukturiertem Log).
- **Keine** anbieter-spezifische Alerting-Logik im Repo. Externer Konsument (mittwald-Cron) lebt außerhalb des Repos.
- **Querschnitt:** Umstellung aller `log.Printf`/`log.Fatalf` (v. a. `cmd/teamwerk/main.go`, Foundation-Packages) auf `slog`; zentrale Logger-Initialisierung in `main.go`.
- Neue `.env`-Variablen `METRICS_TOKEN`, `LOG_FORMAT` (+ `.env.example`/Deploy-Doku). **RAM-Impact vernachlässigbar.** **Kein Frontend betroffen.**
- Restrisiko: `/api/healthz` ist öffentlich ⇒ Payload streng PII-/detail-frei. `/api/metrics` ist token-geschützt und ohne Token deaktiviert.

## Test-Anforderungen

| Route / Logik | Testname | Erwartung | Garantierte Invariante |
|---|---|---|---|
| `GET /api/healthz` (gesund) | `TestHealthz_OK` | `200`, `status:"ok"`, `db:"ok"` | Standardisierter OK-Status für jeden Uptime-Checker |
| `GET /api/healthz` (DB tot) | `TestHealthz_DBDown` | `503`, `db:"fail"` | DB-Ausfall ⇒ harter `503` |
| `GET /api/healthz` (ohne Auth) | `TestHealthz_NoAuthRequired` | `200` ohne Token | Public-Tier (Checker braucht keinen Login) |
| `GET /api/metrics` (ohne/falsches Token) | `TestMetrics_RequiresToken` | `404` (Token ungesetzt) bzw. `401` | Metriken nicht offen exponiert |
| `GET /api/metrics` (mit Token) | `TestMetrics_ExposesSignals` | `200`, Prometheus-Textformat, enthält `teamwerk_disk_free_ratio`, `teamwerk_scheduler_age_seconds`, `teamwerk_panics_total` | Anbieter-neutrale Metrik-Schnittstelle |
| Heartbeat | `TestScheduler_HeartbeatRecorded` | `monitoring_heartbeat.updated_at` frisch nach `Run()` | Erfolgreicher Lauf liefert Dead-Man-Datenquelle |
| Scheduler-Alter | `TestHealthz_SchedulerAgeReported` | `scheduler_age_sec` = Alter des Heartbeats | Dead-Man-Switch extern auswertbar |
| Recover-Middleware | `TestRecover_Panic_IncrementsCounterAndRecovers` | `500` + `teamwerk_panics_total` erhöht + Server lebt, **keine** Mail | Panic crasht nicht, wird als Signal sichtbar — ohne Vendor-Bindung |
| Strukturiertes Log | `TestLogger_EmitsJSON` | Default-Logger schreibt valides JSON mit `level`/`msg`/`time` | Logs maschinenlesbar für beliebige Collector |
| Panic-Log-Record | `TestRecover_Panic_StructuredLog` | Panic erzeugt JSON-Record mit `event="panic"` + Stacktrace-Feld | Log-basiertes Alerting kann auf Panics keyen |

*(Der externe Konsument — mittwald-PHP-Cron bzw. ein beliebiges Monitoring-Tool — ist nicht Teil dieses Repos und hat keine Go-Tests; manuell verifiziert über einen erzwungenen Fehl-Poll, der eine Test-Benachrichtigung auslöst.)*
