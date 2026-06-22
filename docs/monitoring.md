# Monitoring & Alerting

TeamWERK ist **anbieter-neutral** überwachbar: Die App ist reine **Signal-Quelle** über
standardisierte Pull-Schnittstellen. Schwellwerte, Auswertung und Alarmierung leben in
einem **frei austauschbaren externen Monitor** — die App bindet sich an keinen Anbieter
und alarmiert nie selbst.

> ⚠️ **Mindestens ein externer Monitor muss konfiguriert sein und bleiben.** Da die App
> von sich aus nicht alarmiert, sind ohne Monitor zwar alle Signale vorhanden, aber niemand
> schaut hin. Das ist der bewusste Preis der Agnostik.

## Die drei Schnittstellen

| Schnittstelle | Was | Auth | Konsument |
|---|---|---|---|
| `GET /api/healthz` | Liveness/Readiness, `200`/`503` + grobe Signale | keine (public) | jeder Uptime-Checker |
| `GET /api/metrics` | Prometheus-Textformat, reiche Metriken | Bearer `METRICS_TOKEN` (ohne Token: `404`) | jeder Metrik-Scraper |
| `stdout` (slog-JSON) | strukturierte Logs, `event="panic"` u. a. | — | jeder Log-Collector |

### `GET /api/healthz`

```json
{ "status": "ok", "db": "ok", "disk_free_pct": 42, "scheduler_age_sec": 35 }
```

`status`=`ok|degraded`, `db`=`ok|fail` (bei `fail` → HTTP `503`), `disk_free_pct` (−1 = unbekannt),
`scheduler_age_sec` (−1 = noch kein Heartbeat). Payload ist bewusst PII-frei.

### `GET /api/metrics`

Nur aktiv, wenn `METRICS_TOKEN` gesetzt ist. Exponierte Metriken:

| Metrik | Typ | Bedeutung |
|---|---|---|
| `teamwerk_up` | gauge | 1 = Prozess antwortet |
| `teamwerk_db_up` | gauge | 1 = DB erreichbar |
| `teamwerk_disk_free_ratio` | gauge | freier Anteil DB-Dateisystem (0..1, −1 = unbekannt) |
| `teamwerk_mem_free_ratio` | gauge | freier RAM-Anteil (0..1, **nur Linux**) |
| `teamwerk_scheduler_age_seconds` | gauge | Sek. seit letztem Scheduler-Heartbeat (−1 = nie) |
| `teamwerk_panics_total` | counter | abgefangene HTTP-Panics seit Start |
| `teamwerk_uptime_seconds` | gauge | Prozess-Laufzeit |

## Tier-Abdeckung & Beispiel-Schwellen

| Tier | Signal | Beispiel-Alarmregel (im Monitor) |
|---|---|---|
| 0 Erreichbarkeit | `/healthz` HTTP-Status | `≠ 200` |
| 1 App+DB | `/healthz` `db` / `503` | `db != "ok"` |
| 2 Cron lebt | `scheduler_age_sec` | `> 180` (Cron läuft 1×/min) |
| 3 Panics | `teamwerk_panics_total` / Log `event=panic` | Counter steigt |
| 4 Disk/RAM | `disk_free_pct` / `*_free_ratio` | Disk `< 15`, RAM `< 0.1` |
| + Cert | (App terminiert kein TLS) | Domain-Cert `< 14 Tage` |

## Referenz-Konsument: mittwald-Cron (austauschbar)

Die Vereins-Homepage läuft auf mittwald-Webhosting (eigene Failure-Domain, bereits bezahlt,
punktgenauer Cron). Das folgende PHP-Script ist die **Referenz**-Implementierung — es liegt
**auf mittwald, nicht im Repo** (so bleibt die interne URL privat). Jedes andere System
(Better Stack, Grafana Cloud, UptimeRobot, eigenes Prometheus) konsumiert dieselben Endpunkte
genauso; der Monitor ist ohne App-Änderung austauschbar.

Einrichtung in mStudio: Cron-Job `*/2 * * * *` → `php /pfad/zu/teamwerk-monitor.php`.

```php
<?php
// teamwerk-monitor.php — externes Auge, läuft auf mittwald (VPS-unabhängig).
// Alarmiert per mail(); kein Push (Push hinge am evtl. toten VPS).
$BASE      = 'https://internal.team-stuttgart.org';
$HOST      = 'internal.team-stuttgart.org';
$ALERT_TO  = 'vorstand@team-stuttgart.org';
$DISK_MIN  = 15;   // Prozent
$SCHED_MAX = 180;  // Sekunden
$CERT_MIN  = 14;   // Tage

$alerts = [];

// Tier 0/1/2/4 — /api/healthz
$ctx  = stream_context_create(['http' => ['timeout' => 10, 'ignore_errors' => true]]);
$body = @file_get_contents("$BASE/api/healthz", false, $ctx);
$code = 0;
if (isset($http_response_header[0]) && preg_match('#\s(\d{3})\s#', $http_response_header[0], $m)) {
    $code = (int) $m[1];
}
if ($body === false || ($code !== 200 && $code !== 503)) {
    $alerts[] = "healthz nicht erreichbar (HTTP $code)";
} else {
    $h = json_decode($body, true) ?: [];
    if (($h['db'] ?? '') !== 'ok')                         $alerts[] = "DB nicht ok: " . ($h['db'] ?? '?');
    if (isset($h['disk_free_pct']) && $h['disk_free_pct'] >= 0 && $h['disk_free_pct'] < $DISK_MIN)
                                                           $alerts[] = "Disk frei {$h['disk_free_pct']}% (< $DISK_MIN%)";
    if (($h['scheduler_age_sec'] ?? -1) > $SCHED_MAX)      $alerts[] = "Scheduler seit {$h['scheduler_age_sec']}s stumm (Cron tot?)";
    if (($h['scheduler_age_sec'] ?? 0) === -1)             $alerts[] = "Scheduler hat noch nie einen Heartbeat geschrieben";
}

// Cert — TLS-Handshake gegen die Domain
$sctx = stream_context_create(['ssl' => ['capture_peer_cert' => true]]);
$sock = @stream_socket_client("ssl://$HOST:443", $e, $es, 10, STREAM_CLIENT_CONNECT, $sctx);
if ($sock) {
    $params = stream_context_get_params($sock);
    $cert   = openssl_x509_parse($params['options']['ssl']['peer_certificate']);
    $days   = (int) (($cert['validTo_time_t'] - time()) / 86400);
    if ($days < $CERT_MIN) $alerts[] = "TLS-Zertifikat läuft in $days Tagen ab";
    fclose($sock);
}
// Solange Domain/Certbot noch ausstehen: Cert-Block bei Bedarf auskommentieren.

if ($alerts) {
    mail($ALERT_TO, '[TeamWERK] Monitoring-Alarm',
         "Probleme:\n\n- " . implode("\n- ", $alerts) . "\n\nZeit: " . date('c'));
}
```

### Optional: GitHub Actions als zweites, redundantes Auge

Siehe `.github/workflows/uptime.yml` (Schedule). Nutzt die Repo-Variable `HEALTHZ_URL`
(Settings → Secrets and variables → Actions → Variables), damit die interne URL nicht im
öffentlichen Repo steht. Doppeltes Auge schadet nie; GitHub-Cron ist allerdings nicht
minutengenau (Jitter, gelegentlich ausgelassene Ticks) — mittwald bleibt das primäre Auge.

## Beispiel: Metrik-Scraper (Prometheus)

```yaml
scrape_configs:
  - job_name: teamwerk
    scheme: https
    metrics_path: /api/metrics
    authorization:
      type: Bearer
      credentials: "<METRICS_TOKEN>"
    static_configs:
      - targets: ["internal.team-stuttgart.org"]
```

## Beispiel: Log-Collector (Vector auf journald)

Die App schreibt slog-JSON nach stdout → systemd-journal. Ein Collector liest journald und
kann auf `event=panic` alarmieren — die App selbst shippt nichts.

```toml
[sources.teamwerk_journal]
type = "journald"
include_units = ["teamwerk.service"]

[transforms.parse]
type = "remap"
inputs = ["teamwerk_journal"]
source = '. = parse_json!(.message)'

# Sink + Alert-Regel auf .event == "panic" je nach Ziel (Loki/Better Stack/…).
```

## Verifikation

Nach Deploy (manuell, einmalig):

1. `curl -s https://internal.team-stuttgart.org/api/healthz` → `200` + `status:"ok"`.
2. `curl -s -H "Authorization: Bearer $METRICS_TOKEN" https://internal.team-stuttgart.org/api/metrics` → Prometheus-Text.
3. Erzwungener Fehl-Poll (z. B. mittwald-Script kurzzeitig gegen falschen Pfad) → Test-Mail kommt an.
