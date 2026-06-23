## ADDED Requirements

### Requirement: SQLite-Schreibkonkurrenz-Signal (WAL-Größe und BUSY-Counter)

Das System SHALL die Größe der SQLite-WAL-Datei und die Anzahl beobachteter `SQLITE_BUSY`-Returns im HTTP-Pfad als Pull-Metriken über `GET /api/metrics` exponieren, sodass ein externer Monitor Schreibkonkurrenz und WAL-Wachstum als Frühwarn-Signale auswerten kann. Konkret SHALL die Antwort `teamwerk_sqlite_wal_bytes` (gauge) und `teamwerk_sqlite_busy_total` (counter) enthalten. `teamwerk_sqlite_wal_bytes` SHALL `0` sein, wenn die WAL-Datei nicht existiert. `teamwerk_sqlite_busy_total` SHALL um genau `1` je beobachtetem BUSY-Return im HTTP-Schreibpfad steigen.

#### Scenario: WAL-Größe vorhanden

- **WHEN** `GET /api/metrics` mit gültigem Bearer-Token aufgerufen wird
- **THEN** enthält der Body die Zeile `teamwerk_sqlite_wal_bytes` mit einem Wert `≥ 0`

#### Scenario: WAL-Datei fehlt

- **WHEN** die WAL-Datei am DB-Pfad nicht existiert (z. B. nach Checkpoint)
- **THEN** ist der Wert von `teamwerk_sqlite_wal_bytes` `0`
- **AND** `GET /api/metrics` liefert dennoch `200`

#### Scenario: BUSY-Counter steigt im HTTP-Pfad

- **WHEN** ein HTTP-Handler bei einem DB-Schreibzugriff einen `SQLITE_BUSY`-Return erhält
- **THEN** ist `teamwerk_sqlite_busy_total` um genau `1` erhöht
- **AND** der externe Monitor kann das Signal über `/api/metrics` lesen

### Requirement: Scheduler-Schreibkonkurrenz als strukturiertes Log-Event

Das System SHALL `SQLITE_BUSY`-Returns im Scheduler-Pfad (separater Prozess `scheduler:run`) NICHT in den prozesslokalen HTTP-Counter zählen, sondern als strukturiertes `slog.Warn`-Log-Record mit dem stabilen Feld `event="sqlite_busy"` und `source="scheduler"` emittieren. Das Log-Record SHALL über `stdout`/journald an den austauschbaren Log-Collector ausgelieferbar sein, sodass ein externer Monitor BUSY-Vorfälle prozessübergreifend per Log-Query alarmieren kann. Das System SHALL für diesen Pfad KEINE zusätzliche DB-Tabelle oder Datei-basierte Counter-Persistenz einführen.

#### Scenario: BUSY-Event im Scheduler erzeugt Log-Record

- **WHEN** `scheduler.Run()` bei einem DB-Schreibzugriff einen `SQLITE_BUSY`-Return erhält
- **THEN** entsteht ein `slog`-Record mit `level="warn"`, `event="sqlite_busy"` und `source="scheduler"`

#### Scenario: Kein DB-State, kein App-Alert

- **WHEN** der Scheduler einen BUSY-Vorfall verzeichnet
- **THEN** wird KEINE neue Datenbank-Tabelle und KEINE Counter-Datei aktualisiert
- **AND** die App versendet KEINE eigene Benachrichtigung — die Auswertung obliegt dem externen Monitor

### Requirement: HTTP-Concurrency-Signal (in-flight Requests)

Das System SHALL die Anzahl aktuell laufender HTTP-Requests als Gauge `teamwerk_http_requests_in_flight` über `GET /api/metrics` exponieren, sodass CPU-Charts gegen Traffic-Last interpretierbar werden. Der Gauge SHALL prozessintern atomar geführt werden (kein Lock-Contention-Overhead), zum Startzeitpunkt eines Requests inkrementiert und beim Verlassen des Request-Pfads (auch bei Panic) dekrementiert werden.

#### Scenario: Gauge steigt während eines Requests

- **WHEN** ein HTTP-Request gerade verarbeitet wird
- **THEN** liefert `teamwerk_http_requests_in_flight` einen Wert `≥ 1`

#### Scenario: Gauge sinkt nach Abschluss

- **WHEN** der Request beendet ist (regulär oder durch Panic-Recovery)
- **THEN** ist der Beitrag dieses Requests zum Gauge wieder auf `0` zurückgegangen

### Requirement: Host-Telemetrie via Vector-Pipeline (außerhalb des App-Codes)

Das System SHALL keine Host-Metriken (CPU, Memory, Network, Disk, Swap) selbst sammeln oder exponieren — diese Verantwortung liegt bei der austauschbaren Telemetrie-Pipeline (heute Vector auf dem VPS). Die Betriebsdokumentation (`deploy/setup-vps.sh`, `deploy/vps-setup-runbook.md`, `docs/monitoring.md`) SHALL eine vendor-neutrale Pipeline-Konfiguration beschreiben, die (a) Host-Telemetrie über eine `host_metrics`-Source einliest, (b) `/api/metrics` über eine `prometheus_scrape`-Source einliest, und (c) beide Datenströme in einen austauschbaren Metrics-Sink schreibt. Ein Wechsel des Sink-Anbieters SHALL ohne Anwendungscode-Änderung möglich sein.

#### Scenario: Host-Charts gespeist ohne App-Eingriff

- **WHEN** die dokumentierte Pipeline-Konfiguration auf dem VPS aktiv ist
- **THEN** liefert die Pipeline CPU-, Memory-, Network-, Disk- und Swap-Metriken an den externen Sink
- **AND** der Anwendungscode wurde dafür NICHT angepasst

#### Scenario: App-Metriken via Prometheus-Scrape

- **WHEN** die Pipeline `/api/metrics` mit Bearer `METRICS_TOKEN` scrapet
- **THEN** erscheinen die `teamwerk_*`-Metriken im selben externen Sink wie die Host-Metriken

#### Scenario: Sink-Wechsel ohne App-Änderung

- **WHEN** der Metrics-Sink in der Pipeline-Konfiguration durch einen anderen Anbieter ersetzt wird
- **THEN** ist KEINE Änderung an der App, ihren Endpunkten oder ihrer Konfiguration nötig
