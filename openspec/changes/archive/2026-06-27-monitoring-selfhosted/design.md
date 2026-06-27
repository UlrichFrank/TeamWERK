# Design — Anbieter-neutrales Monitoring (Signal-Quelle + austauschbarer Monitor)

## Leitprinzip: App = Signal-Quelle, Monitor = austauschbarer Konsument

Die App bindet sich an **kein** Monitoring-System. Sie exponiert ihren Zustand über zwei Pull-basierte De-facto-Standards; **alle** Schwellwerte, Auswertung und Alarmierung leben im externen Monitor — und der ist frei austauschbar.

```
APP = SIGNAL-QUELLE (stabile, vendor-neutrale Schnittstellen)
  ├─ GET /api/healthz   → 200/503 + {status, db, disk_free_pct, scheduler_age_sec}
  │                        Liveness/Readiness · public · PII-frei · "spricht" jeder Uptime-Checker
  ├─ GET /api/metrics   → Prometheus-Textformat (Bearer-Token)
  │                        teamwerk_up / _db_up / _disk_free_ratio / _mem_free_ratio /
  │                        _scheduler_age_seconds / _panics_total / _uptime_seconds
  │                        "spricht" jeder Scraper (Prometheus/Grafana/Netdata/Datadog/…)
  └─ slog-JSON         → stdout/journald · stabile Felder (event=panic …)
                         "spricht" jeder Log-Collector (Vector/Alloy/Loki/Better Stack/…)

                        ▲ Pull (der Monitor holt; die App pusht nichts, kennt keinen Endpunkt)
                        │
MONITOR = AUSTAUSCHBARER KONSUMENT  (Schwellen + Alerting HIER, nicht in der App)
  mittwald-Cron (Referenz)  ─┐
  GitHub Actions (optional)  ─┼─→ pollt /healthz · scrapet /metrics · prüft TLS-Cert
  Better Stack / Grafana     ─┤    wertet Schwellen aus · alarmiert über EIGENEN Kanal
  UptimeRobot / Prometheus   ─┘
```

## Warum Pull, nicht Push

Würde die App aktiv an einen Endpunkt pushen (Mail, Webhook, Heartbeat-Ping zu Anbieter X), wäre Anbieter X im App-Code verdrahtet — genau die Bindung, die wir vermeiden. **Pull** dreht das um: die App kennt keinen Konsumenten, der Konsument kennt nur zwei Standard-URLs. Wechsel des Monitors = Konfiguration beim Monitor, **null** App-Änderung.

## Tier-Abdeckung — alle 5, rein über Signale

| Tier | Signal (App stellt bereit) | Auswertung (im Monitor, austauschbar) |
|---|---|---|
| 0 Erreichbarkeit | `/healthz` HTTP-Status | „≠ 200 ⇒ Alarm" |
| 1 App+DB | `/healthz` `db`, `503` | „db≠ok ⇒ Alarm" |
| 2 Cron lebt | `scheduler_age_seconds` | „> Schwelle ⇒ Alarm" (Dead-Man) |
| 3 Panics | `teamwerk_panics_total` (+ strukturiertes Log) | „Counter steigt ⇒ Alarm" |
| 4 Disk/RAM | `disk_free_ratio` / `mem_free_ratio` | „< Schwelle ⇒ Alarm" |
| + Cert | (App terminiert kein TLS) | Monitor prüft Domain-Cert selbst |

Entscheidend: In der Spalte „Auswertung" steht **nirgends App-Code**. Schwellwerte sind Monitor-Konfiguration, keine `.env` der App.

## Entscheidungen

### Zwei Schnittstellen statt einer: `/healthz` (dumm) + `/metrics` (reich)
- `/healthz`: minimal, **public**, für simple Checker (mittwald-Cron, UptimeRobot, k8s-Probe) — die brauchen kein Prometheus-Parsing. Trägt nur grobe, nicht-sensible Signale.
- `/metrics`: **Prometheus-Exposition** — die Lingua franca der Metrik-Welt. Damit ist der *Metrik*-Monitor austauschbar, ohne dass die App ein bespoke JSON-Schema erzwingt. Token-geschützt, weil reicher/granularer.

### `/api/metrics` & `/api/healthz` unter `/api/`
Der Go-Server bedient SPA-Embed mit Catch-all-Fallback; Root-Routen kollidierten. Scraper akzeptieren jeden Pfad, daher kostet `/api/`-Prefix nichts.

### `/metrics`-Zugriffsschutz: Bearer-Token, Default deaktiviert
`METRICS_TOKEN` aus `.env`. Ungesetzt ⇒ Endpoint liefert `404` (sicherer Default, keine versehentliche Exposition). Gesetzt ⇒ `Authorization: Bearer <token>` Pflicht. Prometheus, Grafana-Agent, Better-Stack-Collector etc. unterstützen alle Bearer/Basic-Auth am Scrape-Target.

### Kein In-App-Alerting (bewusster Bruch mit dem ersten Entwurf)
Der ursprüngliche Entwurf ließ den Scheduler selbst Mail/Push schicken und die Recover-Middleware alarmieren. Das ist ein **fest eingebautes Monitoring-System** — unvereinbar mit „austauschbar". Daher: Scheduler schreibt nur den Heartbeat (Datenquelle), Recover-Middleware inkrementiert nur den Counter. Wer eine On-Box-Fallback-Alarmierung über den vorhandenen Mailer/Push will, baut sie als **separaten, optionalen Konsumenten** der gleichen Signale — nicht in die Geschäftslogik. (Nicht Teil dieses Changes.)

### Heartbeat: Single-Row-Tabelle
`monitoring_heartbeat(id INTEGER PRIMARY KEY CHECK(id=1), updated_at TEXT NOT NULL)`. `scheduler.Run()` macht am Ende `INSERT … ON CONFLICT(id) DO UPDATE`. `/healthz` und `/metrics` lesen daraus das Alter. Datei-Variante verworfen (zusätzliche FS-Fehlerquelle, schlechter testbar).

### Disk/Speicher
- Disk: `syscall.Statfs` auf das `DB_PATH`-Verzeichnis (Linux **und** macOS/Dev → testbar).
- Speicher: `/proc/meminfo` (Linux-only); fehlt die Quelle, wird `teamwerk_mem_free_ratio` schlicht **nicht** exportiert (kein Fehler). Honest: RAM ist reiner Prod-Linux-Mehrwert.

### Referenz-Konsument auf mittwald (austauschbar)
PHP-Cron auf dem bestehenden Webhosting (eigene Failure-Domain, bereits bezahlt, punktgenau): `file_get_contents`/`curl` auf `/api/healthz`, JSON werten, TLS-Cert via `openssl_x509_parse`, Alarm via `mail()` (läuft auf mittwald ⇒ VPS-unabhängig). Liegt außerhalb des Repos ⇒ interne URL bleibt privat. **Dieser Konsument ist Beispiel, nicht Vertrag** — Better Stack, Grafana Cloud, UptimeRobot oder ein eigenes Prometheus konsumieren dieselben zwei Endpunkte genauso.

### `slog` als dritte neutrale Schnittstelle
Strukturierte JSON-Logs nach stdout→journald sind die anbieter-neutrale Log-Schnittstelle: jeder Collector (Vector/Alloy/Loki/Better-Stack/Datadog-Agent) parst JSON ohne App-Wissen, journald puffert ohnehin.

- **Zentrale Initialisierung** in `main.go`: `slog.SetDefault(slog.New(handler))`. `LOG_FORMAT=json` (Prod-Default) → `slog.NewJSONHandler(os.Stdout, …)`; `LOG_FORMAT=text` (lokal) → `slog.NewTextHandler` für lesbare DX. Kein zweiter Logging-Pfad, kein Dritt-Logger.
- **Schreiben, nicht shippen:** Die App emittiert nur nach stdout. Welcher Collector einsammelt, ist Betriebskonfiguration — identisches Pull-/Trennungs-Prinzip wie bei `/metrics`. Kein Anbieter im Code.
- **Stabile Felder:** Panics (aus der Recover-Middleware) und Schlüsselereignisse tragen feste Attribute (`event="panic"`, Stacktrace-Feld), damit ein log-basierter Alert zuverlässig darauf keyen kann — komplementär zum Counter `teamwerk_panics_total` (Counter = „wie viele", Log = „was genau").
- **Migration (Querschnitt, kontrolliert):** `cmd/teamwerk/main.go` zuerst (Großteil der `log.Fatalf`/`log.Printf`), `log.Fatalf` → `slog.Error(...)` + `os.Exit(1)`; danach Foundation-Packages durchsweepen. Domain-Handler nutzen `slog` über den Default-Logger; kein DI-Umbau nötig. Der Architektur-Test (`internal/arch`) bleibt unberührt (kein neues Package-Coupling).

Bewusst entkoppelt von den Pull-Signalen: `/healthz` und `/metrics` funktionieren unabhängig — fällt die Log-Migration aus dem Scope, stehen Tier 0–2/4 trotzdem. Sie wird hier aber **mitgenommen**, damit auch der Log-Collector austauschbar ist (Tier 3 vollständig agnostisch statt nur über den Counter).

## Abgelehnte Alternativen
- **In-App-Alerting (Mailer/Push) im Kern** — nicht austauschbar; verworfen (siehe oben).
- **Push-Heartbeat zu einem konkreten Anbieter** (z. B. Healthchecks.io-Ping) — verdrahtet den Anbieter; durch Pull-`scheduler_age` ersetzt.
- **Selbstgehostetes Prometheus/Grafana auf dem VPS** — sprengt 1 GB RAM, gleiche Failure-Domain.
- **Nur bespoke JSON statt Prometheus** — zwänge jeden Metrik-Monitor zu Custom-Parsing; Prometheus-Format maximiert Austauschbarkeit.

## Risiken / offene Punkte
- **Kein Monitor = keine Alarme.** Agnostik bedeutet: die App schweigt von sich aus. Es muss organisatorisch sichergestellt sein, dass ≥ 1 externer Monitor konfiguriert *bleibt*. (Ironie-Mitigation: der mittwald-Cron kann sich selbst über GH-Actions als zweites Auge absichern.)
- **Cert-Monitoring erst scharf, sobald Domain/Certbot stehen** (laut CLAUDE.md ausstehend) — bis dahin Cert-Schritt im Konsumenten tolerant.
- **`/healthz` öffentlich** — DoS-Oberfläche minimal (billiger Handler), Payload PII-frei; `/metrics` token-geschützt.
