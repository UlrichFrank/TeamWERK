## Context

`monitoring-selfhosted` (in-progress, Live-Verifikation steht aus) etabliert die Architektur: App = Signal-Quelle (`/api/healthz`, `/api/metrics`, `slog`-JSON), Monitor = austauschbar. Auf dem VPS läuft bereits ein **Vector**-Prozess, der heute ausschließlich journald-Logs an Better Stack Logs shippt (`deploy/setup-vps.sh` Zeilen 134–171).

Better Stack Telemetry hält ein vorgefertigtes "Host(Vector)"-Dashboard mit Charts für CPU, Memory, Network, Swap, Disk bereit; diese Charts sind heute leer, weil keine Host-Metriken geliefert werden. Gleichzeitig ist `/api/metrics` zwar implementiert, wird aber von keinem Scraper konsumiert — der mittwald-PHP-Cron pollt nur `/api/healthz`.

Mit wachsender Nutzerzahl rücken zwei Frühwarn-Klassen ins Blickfeld: Host-Sättigung (CPU/RAM/Disk-IO/Network) und SQLite-Schreibkonkurrenz (WAL-Wachstum, `SQLITE_BUSY`). Beides ist heute nicht beobachtbar.

## Goals / Non-Goals

**Goals:**

- Die 4 Host-Charts in Better Stack füllen, ohne einen zweiten Agenten zu installieren — der vorhandene Vector wird um Sources erweitert.
- SQLite-spezifische App-Signale (`teamwerk_sqlite_wal_bytes`, `teamwerk_sqlite_busy_total`) und ein traffic-Indikator (`teamwerk_http_requests_in_flight`) als Pull-Metriken über `/api/metrics` exponieren.
- Architektur-Doktrin von `monitoring-selfhosted` strikt beibehalten: kein In-App-Alerting, keine Vendor-Bindung im App-Code, Schwellen leben im austauschbaren Monitor.
- Doku (`deploy/setup-vps.sh`, Runbook, `docs/monitoring.md`) so erweitern, dass eine VPS-Neueinrichtung beide Pipelines reproduzierbar aufbaut.

**Non-Goals:**

- Kein Histogram-Latency-Tracking (Prometheus-Histogramme blasen die Cardinality auf — kritisch auf 1-GB-Box).
- Kein Per-Route-Counter (gleicher Grund; `chi`-Middleware mit Route-Pattern als Label würde linear mit der Routen-Zahl wachsen).
- Keine zweite DB-Tabelle für Cross-Prozess-Counter (Scheduler-Prozess persistiert keine Counter).
- Kein Wechsel des Sink-Anbieters (Better Stack bleibt — austauschbar bleibt es durch die Architektur, nicht durch dieses Change).
- Keine Histogramme über SQLite-Latencies oder DB-Connection-Pool-Größe (kein Bedarf bei aktueller Last).
- Frontend bleibt unberührt.

## Decisions

### SQLite-BUSY-Counter über Wrapping-Driver am `database/sql`-Layer

Die Wahl zwischen mehreren Optionen:

| Option | Beschreibung | Pro | Contra |
|---|---|---|---|
| α Handler-Sweep | jede `Exec/Query`-Aufrufstelle anfassen | direkt, lokal | **258 Aufrufstellen in 22 Dateien**; Drift bei neuen Stellen unvermeidbar |
| β Treiber-Statistiken auslesen | über `modernc.org/sqlite`-interne API | zentral | API instabil/nicht exportiert |
| γ Helper + Konvention | kleine `health.CheckSQLiteBusy(err)` im Schreibpfad, dokumentiert in CLAUDE.md | Lokalität | bei 258 Stellen unrealistisch, leise Drift garantiert |
| **δ Wrapping-Driver (gewählt)** | zweiter `database/sql/driver` registriert, der `"sqlite"` delegiert und Errors auf BUSY prüft | zero Handler-Änderungen, jeder zukünftige `Exec` automatisch erfasst, eine Stelle | ~150 LOC Boilerplate-Delegation; fehlendes Subinterface fällt auf Slow-Path zurück |

Die Reibung bei α/γ (258 Stellen, 22 Files, dauerhaft) übersteigt die Implementierungs-Reibung von δ (einmalig ~150 LOC in `internal/db/busy_driver.go`). δ ist auch **future-proof**: jeder neue Mutations-Handler wird ohne Konvention erfasst, der Counter kann nicht heimlich erodieren.

**Implementierung:**

```
internal/db/busy_driver.go
├─ init(): sql.Register("sqlite-busy-counting", &busyDriver{})
├─ busyDriver.Open → wraps driver.Conn
├─ busyConn implementiert: Prepare, Close, Begin, PrepareContext,
│                          BeginTx, ExecContext, QueryContext, Ping,
│                          ResetSession, IsValid, CheckNamedValue
├─ busyStmt implementiert: Close, NumInput, Exec, Query,
│                          ExecContext, QueryContext, CheckNamedValue
└─ jede Methode delegiert + ruft health.CheckSQLiteBusy(err) am Error-Pfad

internal/db/db.go
└─ sql.Open("sqlite-busy-counting", path)  // statt "sqlite"
```

Handler bleiben `*sql.DB` (Typ unverändert), keine Konvention nötig. `health.CheckSQLiteBusy(err)` bleibt als Public-API erhalten — wird jetzt zentral vom Driver aufgerufen.

→ **Entschieden:** Wrapping-Driver. Setup-Aufwand einmalig, Wartung null.

### Scheduler-BUSY via slog statt Cross-Prozess-Counter

Der Scheduler ist ein **separater Prozess** (Cron `* * * * * teamwerk scheduler:run`, lebt < 1 s). Ein In-Memory-Counter im HTTP-Daemon erfasst ihn nicht. Drei Optionen:

1. **DB-persistenter Counter:** Tabelle `monitoring_counters(name, value)` mit `UPDATE ... SET value=value+1`. Mehr Code, neue Migration, dreht ausgerechnet an SQLite-Konkurrenz, die wir messen wollen — Selbstinterferenz.
2. **Filesystem-Counter:** Datei lock+increment. Mehr Code, fragiler.
3. **slog-Event (gewählt):** `slog.Warn("sqlite_busy", "source", "scheduler", "op", "<context>")`. Die JSON-Logs gehen durch journald → Vector → Better Stack Logs. Alarmierung als **Log-Query** (`level:warn event:sqlite_busy source:scheduler`) im Monitor. Konsistent mit der bestehenden Architektur (`panic` → strukturierter Log + Counter; hier nur Log).

→ **Entschieden:** Option 3. Kostet eine Zeile `slog.Warn(...)` an genau einer Stelle im Scheduler-Pfad, der heute schon `sqlite.ErrBusy` als Error sieht.

### `teamwerk_http_requests_in_flight` als atomic-Gauge

Implementierung:

```go
var httpInFlight atomic.Int64

func InFlightMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        httpInFlight.Add(1)
        defer httpInFlight.Add(-1)
        next.ServeHTTP(w, r)
    })
}
```

Kosten: ~30–50 ns je atomic-Op, 8 Bytes Speicher, Cardinality 1. Eingehängt in `internal/app/router.go` **vor** der Recover-Middleware (damit ein Panic den Counter korrekt dekrementiert — `defer` läuft auch dort).

Nutzen: deutet CPU-Spitzen — CPU↑ + in_flight↑ = traffic-getrieben; CPU↑ + in_flight≈0 = GC/Disk-IO/Scheduler.

### `teamwerk_sqlite_wal_bytes` per `os.Stat` zur Read-Zeit

Beim `/api/metrics`-Request einmal `os.Stat(dbPath + "-wal")` aufrufen, Größe in Bytes ausgeben. Falls Datei nicht existiert (WAL nicht aktiv oder gerade geleert): 0. Kein Caching nötig — `/api/metrics` wird einmal pro Scrape-Intervall (typisch 30–60 s) aufgerufen.

`DB_PATH` ist über die bestehende Config bekannt; der `health.Handler` braucht ihn injiziert. Bestehender Konstruktor erweitern (additiv, kein Breaking Change).

### Vector-Erweiterung: zusätzliche Sources, neuer Sink — kein zweiter Agent

`vector.toml` wird um drei Blöcke erweitert:

```toml
[sources.host]
type = "host_metrics"
collectors = ["cpu", "memory", "disk", "network", "filesystem", "load"]
scrape_interval_secs = 30

[sources.teamwerk_app]
type = "prometheus_scrape"
endpoints = ["http://127.0.0.1:8080/api/metrics"]
auth.strategy = "bearer"
auth.token = "${METRICS_TOKEN}"
scrape_interval_secs = 30

[sinks.betterstack_metrics]
type = "http"  # konkretes Format gemäß Better-Stack-Telemetry-Doku
inputs = ["host", "teamwerk_app"]
uri = "<aus Better Stack: Telemetry → Sources → Vector>"
auth.strategy = "bearer"
auth.token = "${BETTERSTACK_METRICS_TOKEN}"
```

(Genaue Sink-Konfig wird beim Anlegen der Better-Stack-Metrics-Source dort generiert und in `docs/monitoring.md` als Referenz dokumentiert.)

**Voraussetzung an die App-Konfig:** `METRICS_TOKEN` muss auf dem VPS gesetzt sein. Heute ist es das nicht zwingend (Endpoint antwortet sonst mit `404`). Setup-Script wird angepasst.

### Token-File-Schema bleibt

Analog zu `/etc/teamwerk/betterstack-logs-token` wird `/etc/teamwerk/betterstack-metrics-token` angelegt (chmod 600). Vector liest beide via `${VAR}`-Substitution oder direkte `auth.token`-Felder. Konsistenz mit dem bestehenden Pattern — keine neue Mechanik.

### Spur A und B in einem Change

Beide Spuren teilen das Motiv "Vorbereitung auf Wachstum", und Spur B macht ohne Spur A in Better Stack nichts sichtbar (die App-Metriken würden zwar exponiert, aber nicht gescraped). Trennung wäre Buchhaltung ohne Nutzen.

## Risks / Trade-offs

- **Vector-RAM-Footprint** → Mitigation: aktuell ~20–40 MB resident (nur journald). `host_metrics` + `prometheus_scrape` addieren laut Vector-Doku < 10 MB. Insgesamt bleibt der Prozess deutlich unter 100 MB auf einer 1-GB-Box. Wird im Runbook dokumentiert; bei Engpässen kann `host_metrics`-Collector-Liste reduziert werden.
- **Selbstinterferenz `prometheus_scrape` → `/api/metrics`** → der Scrape erzeugt selbst HTTP-Requests, die `teamwerk_http_requests_in_flight` und potenziell `teamwerk_sqlite_busy_total` (nein — `/metrics` schreibt nicht) beeinflussen. Mitigation: `/api/metrics` selbst inkrementiert keinen BUSY-Counter, und in_flight-Bias ist konstant +1 für die Scrape-Dauer (irrelevant für Trendanalyse).
- **`METRICS_TOKEN` nicht gesetzt** → Vector scrapet `/api/metrics`, bekommt 404, App-Metriken fehlen still. Mitigation: Runbook-Checkliste explizit; `setup-vps.sh` prüft und warnt, falls Token leer.
- **Better-Stack-Metrics-Sink-Format** → Better Stack akzeptiert unter Telemetry mehrere Vector-Sink-Typen; die exakte Konfiguration hängt von der dort angelegten Source ab. Mitigation: in `docs/monitoring.md` wird der **Vorgang** (Source anlegen → URL+Token in Vector eintragen) dokumentiert, nicht ein fixer Snippet — der Snippet wird beim Setup aus dem Better-Stack-UI kopiert.
- **`os.Stat` auf `*-wal`-Datei** → bei sehr seltenem Race kann SQLite gerade einen Checkpoint machen, sodass die Datei für Millisekunden 0 Bytes hat. Mitigation: das ist ein **gewollter** Snapshot, kein Fehler — der Reader liefert "0" und der Monitor sieht beim nächsten Scrape den realistischen Wert.
- **Fehlendes driver.Conn-Subinterface im Wrapper** → wenn der Wrapper z. B. `driver.ExecerContext` nicht implementiert, fällt `database/sql` auf eine Slow-Path-Iteration über `Prepare → Stmt.ExecContext` zurück; das funktioniert noch, aber langsamer. Mitigation: alle Subinterfaces, die `modernc.org/sqlite` selbst implementiert, im Wrapper spiegeln; Build-Time-Assertions (`var _ driver.ExecerContext = (*busyConn)(nil)`) machen Vergessen sichtbar.
- **Driver-Wrapper schluckt keinen Error, schreibt nur einen Counter** → das ist genau die gewünschte Semantik (Pass-through + Side-Effect). Mitigation: durch Tests gesichert (Counter steigt, Error kommt unverändert beim Handler an).

## Migration Plan

1. App-Code merge (3 Metriken + 1 slog-Event + Tests + CLAUDE.md-Konvention).
2. Deploy via `make deploy` — bestehendes Verhalten bleibt, neue Metriken erscheinen unter `/api/metrics` (sofern `METRICS_TOKEN` gesetzt; sonst weiterhin 404).
3. `METRICS_TOKEN` auf VPS setzen, falls noch nicht: in `/etc/teamwerk/env`, `systemctl restart teamwerk`.
4. In Better Stack: Telemetry → Sources → "Vector" anlegen → URL + Token notieren.
5. `/etc/teamwerk/betterstack-metrics-token` befüllen.
6. `vector.toml` aktualisieren (manuell oder via `setup-vps.sh` neu durchlaufen), `systemctl restart vector`.
7. In Better Stack: "Host(Vector)"-Dashboard öffnen — Charts füllen sich binnen ~1 min.
8. Verifikation: künstlicher DB-Hammer (paralleler Schreibtest) → `teamwerk_sqlite_busy_total` und `teamwerk_sqlite_wal_bytes` steigen sichtbar.

**Rollback:** alte `vector.toml` zurück, `systemctl restart vector`. App-Metriken bleiben exponiert (schaden nicht, niemand scrapet). Kein DB-Schema-Eingriff, also keine Migration-Rollback nötig.

## Open Questions

- Soll `setup-vps.sh` `METRICS_TOKEN` automatisch erzeugen (analog zur Heartbeat-URL-Datei), falls leer? **Vorschlag:** ja, generiere via `openssl rand -hex 32` und schreibe ins env, falls nicht gesetzt — sonst bleibt es eine manuelle Falle. Wird in Tasks präzisiert.
- Genaue Better-Stack-Vector-Sink-Konfiguration: wird beim Setup live aus dem Better-Stack-UI gezogen; in `docs/monitoring.md` als Schritt dokumentiert, nicht als Literal-Snippet.
