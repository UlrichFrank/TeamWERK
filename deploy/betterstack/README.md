# deploy/betterstack

Better Stack Observability — Logs + Metrics via [Vector](https://vector.dev/).

## Architektur

```
teamwerk.service (systemd)
    └─ journald → Vector → Better Stack Logs
/api/metrics (Prometheus)
    └─ Vector scrape → Better Stack Metrics
Host (CPU/RAM/Disk)
    └─ Vector host_metrics → Better Stack Metrics
```

## VPS-Setup

### 1. Vector installieren

```bash
curl -1sLf 'https://repositories.timber.io/public/vector/cfg/setup/bash.deb.sh' | bash
apt-get install vector
```

### 2. Konfiguration deployen

`vector.toml` aus dem Repo ins VPS-Konfigurationsverzeichnis kopieren und die
Platzhalter durch echte Tokens ersetzen:

```bash
cp vector.toml /etc/vector/vector.toml
# Platzhalter ersetzen:
#   BETTERSTACK_LOGS_TOKEN    → Better Stack → Sources → Log-Source → Token
#   BETTERSTACK_METRICS_TOKEN → Better Stack → Sources → Metrics-Source → Token
#   METRICS_TOKEN             → Wert von METRICS_TOKEN aus /etc/teamwerk/env
sed -i 's/BETTERSTACK_LOGS_TOKEN/<echter-token>/' /etc/vector/vector.toml
sed -i 's/BETTERSTACK_METRICS_TOKEN/<echter-token>/' /etc/vector/vector.toml
sed -i 's/METRICS_TOKEN/<echter-token>/' /etc/vector/vector.toml

systemctl enable --now vector
```

### 3. Dashboard anlegen (einmalig)

```bash
export BETTERSTACK_API_TOKEN=<Telemetry-API-Token aus Better Stack Team Settings>
./apply.sh        # legt Dashboard + Charts an
# ./apply.sh --dry-run   # zeigt nur Payloads ohne API-Calls
```

Danach im Better-Stack-UI das Source-Dropdown auf die Metrics-Source stellen.

## Dateien

| Datei | Zweck |
|---|---|
| `vector.toml` | Vector-Konfiguration (Repo-Root); Tokens als Platzhalter |
| `apply.sh` | Einmaliges Dashboard-Setup via Better-Stack-Telemetry-API |
| `system-dashboard.json` | Dashboard- + Chart-Definitionen (Quelle für `apply.sh`) |
