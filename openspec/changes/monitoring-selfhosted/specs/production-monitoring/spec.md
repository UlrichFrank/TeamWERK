## ADDED Requirements

### Requirement: Standardisierter Health-Endpoint (Liveness/Readiness)

Das System SHALL unter `GET /api/healthz` ohne Authentifizierung einen Health-Status im De-facto-Standard liefern, den jeder Uptime-Checker konsumieren kann. Die Antwort SHALL `status` (`ok`|`degraded`), `db` (`ok`|`fail`), `disk_free_pct` (Integer) und `scheduler_age_sec` (Integer) enthalten und KEINE personenbezogenen oder internen Detaildaten (Pfade, Versionen, Hostnamen). Bei nicht erreichbarer Datenbank SHALL der Status-Code `503` sein, sonst `200`.

#### Scenario: Gesundes System

- **WHEN** `GET /api/healthz` aufgerufen wird und die Datenbank erreichbar ist
- **THEN** ist der Status-Code `200`
- **AND** der Body enthält `status:"ok"` und `db:"ok"`

#### Scenario: Datenbank nicht erreichbar

- **WHEN** der DB-Ping fehlschlägt
- **THEN** ist der Status-Code `503`
- **AND** der Body enthält `db:"fail"`

#### Scenario: Kein Login erforderlich

- **WHEN** `GET /api/healthz` ohne Access-Token aufgerufen wird
- **THEN** antwortet das System mit `200` (Public-Tier)

### Requirement: Anbieter-neutrale Metrik-Schnittstelle

Das System SHALL Betriebsmetriken unter `GET /api/metrics` im Prometheus-Textformat bereitstellen, sodass beliebige Scraper sie ohne App-Anpassung konsumieren können. Der Endpoint SHALL per Bearer-Token (`METRICS_TOKEN`) geschützt sein; ist das Token nicht gesetzt, SHALL der Endpoint deaktiviert sein (`404`). Die Ausgabe SHALL mindestens `teamwerk_db_up`, `teamwerk_disk_free_ratio`, `teamwerk_scheduler_age_seconds`, `teamwerk_panics_total` und `teamwerk_uptime_seconds` enthalten.

#### Scenario: Zugriff ohne gültiges Token

- **WHEN** `GET /api/metrics` ohne oder mit falschem Bearer-Token aufgerufen wird und `METRICS_TOKEN` gesetzt ist
- **THEN** antwortet das System mit `401`

#### Scenario: Endpoint deaktiviert ohne Konfiguration

- **WHEN** `GET /api/metrics` aufgerufen wird und `METRICS_TOKEN` nicht gesetzt ist
- **THEN** antwortet das System mit `404`

#### Scenario: Metriken im Standardformat

- **WHEN** `GET /api/metrics` mit gültigem Bearer-Token aufgerufen wird
- **THEN** ist der Status-Code `200`
- **AND** der Body ist Prometheus-Textformat und enthält `teamwerk_disk_free_ratio`, `teamwerk_scheduler_age_seconds` und `teamwerk_panics_total`

### Requirement: Scheduler-Heartbeat als Dead-Man-Datenquelle

Das System SHALL nach jedem erfolgreichen `scheduler:run`-Lauf einen Heartbeat-Zeitstempel persistieren und dessen Alter über `/api/healthz` (`scheduler_age_sec`) sowie `/api/metrics` (`teamwerk_scheduler_age_seconds`) exponieren. Das System SHALL aus dem Heartbeat KEINE eigene Alarmierung ableiten — die Dead-Man-Auswertung obliegt dem externen Monitor.

#### Scenario: Heartbeat nach Lauf aktualisiert

- **WHEN** `scheduler.Run()` erfolgreich durchläuft
- **THEN** ist der persistierte Heartbeat-Zeitstempel auf den aktuellen Lauf aktualisiert

#### Scenario: Alter extern auswertbar

- **WHEN** `GET /api/healthz` aufgerufen wird
- **THEN** entspricht `scheduler_age_sec` der Differenz zwischen jetzt und dem letzten Heartbeat

#### Scenario: Keine App-seitige Alarmierung

- **WHEN** der Scheduler lange keinen Heartbeat geschrieben hat
- **THEN** versendet die App selbst keine Benachrichtigung
- **AND** das hohe `scheduler_age_sec` ist über die Schnittstellen sichtbar, sodass ein externer Monitor alarmieren kann

### Requirement: Panic-Observability ohne Anbieter-Bindung

Das System SHALL HTTP-Handler-Panics abfangen, den Stacktrace strukturiert loggen, mit `500` antworten und den Prozess am Leben halten. Es SHALL jeden Panic in einem Zähler `teamwerk_panics_total` sichtbar machen. Das System SHALL aus Panics heraus KEINE anbieter-spezifische Benachrichtigung (Mail/Push/Webhook) versenden — der Zähler ist das neutrale Signal.

#### Scenario: Panic wird abgefangen und sichtbar

- **WHEN** ein Handler in einer Anfrage paniced
- **THEN** antwortet das System mit `500`
- **AND** der Server läuft weiter
- **AND** `teamwerk_panics_total` ist um genau 1 erhöht

#### Scenario: Keine eingebaute Alarmierung

- **WHEN** ein Panic auftritt
- **THEN** versendet die App selbst keine Mail/Push/Webhook
- **AND** die Erhöhung von `teamwerk_panics_total` ist über `/api/metrics` für einen externen Monitor sichtbar

### Requirement: Strukturierte, anbieter-neutrale Logs

Das System SHALL seine Logs strukturiert (JSON über `slog`) nach stdout schreiben, sodass beliebige Log-Collector sie ohne App-Wissen parsen können. Das Ausgabeformat SHALL über `LOG_FORMAT` (`json`|`text`) konfigurierbar sein, mit `json` als Default. Das System SHALL Logs NICHT selbst an einen externen Dienst versenden (der Collector ist austauschbare Betriebskonfiguration). Panics SHALL als Log-Record mit dem stabilen Feld `event="panic"` und einem Stacktrace-Feld erscheinen.

#### Scenario: JSON-Logs als Default

- **WHEN** der Default-Logger ohne `LOG_FORMAT`-Override initialisiert wird
- **THEN** schreibt er valide JSON-Records mit mindestens `level`, `msg` und `time`

#### Scenario: Lesbares Format für lokale Entwicklung

- **WHEN** `LOG_FORMAT=text` gesetzt ist
- **THEN** schreibt der Logger menschenlesbare Textzeilen statt JSON

#### Scenario: Panic als strukturierter Record

- **WHEN** ein Handler paniced und die Recover-Middleware greift
- **THEN** entsteht ein Log-Record mit `event="panic"` und einem Stacktrace-Feld

#### Scenario: Kein App-seitiges Log-Shipping

- **WHEN** das System läuft
- **THEN** sendet es Logs ausschließlich nach stdout und an keinen fest verdrahteten externen Log-Dienst

### Requirement: Austauschbarer externer Monitor

Das System SHALL so beschaffen sein, dass die Überwachung von einem externen, außerhalb des VPS laufenden Monitor übernommen wird, der ausschließlich die Standard-Schnittstellen `/api/healthz` und/oder `/api/metrics` konsumiert. Die App SHALL keinen konkreten Monitoring-Anbieter im Code referenzieren; ein Wechsel des Monitors SHALL ohne Änderung am Anwendungscode möglich sein. Als Referenz-Implementierung dient ein Cron auf dem bestehenden mittwald-Webhosting.

#### Scenario: Erreichbarkeitsausfall extern erkannt

- **WHEN** der externe Monitor `/api/healthz` pollt und keinen `200` erhält
- **THEN** alarmiert der Monitor über seinen eigenen, VPS-unabhängigen Kanal

#### Scenario: Monitor austauschbar ohne App-Änderung

- **WHEN** der bisherige Monitor durch ein anderes System ersetzt wird, das dieselben Endpunkte konsumiert
- **THEN** ist keine Änderung am Anwendungscode erforderlich

#### Scenario: Zertifikatsablauf

- **WHEN** das TLS-Zertifikat der Domain in weniger als der vom Monitor definierten Frist abläuft
- **THEN** alarmiert der externe Monitor (die App selbst prüft kein Zertifikat)
