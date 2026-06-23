#!/usr/bin/env bash
# Run once on einem frischen IONOS VPS, um die Server-Umgebung aufzusetzen.
# Usage: bash setup-vps.sh
#
# Idempotent — kann mehrfach laufen, überschreibt nichts Bestehendes.
# Vor dem Lauf: deploy/vps-setup-runbook.md lesen (manuelle Vorbedingungen).

set -euo pipefail

# ---------------------------------------------------------------------------
# 1. System-Pakete
# ---------------------------------------------------------------------------
apt-get update
apt-get install -y nginx openssl curl ca-certificates gnupg logrotate

# ---------------------------------------------------------------------------
# 2. Verzeichnisse
# ---------------------------------------------------------------------------
mkdir -p /var/lib/teamwerk/{uploads,files}
chown -R www-data:www-data /var/lib/teamwerk

# ---------------------------------------------------------------------------
# 3. Env-Datei mit Platzhaltern (nur anlegen, wenn nicht vorhanden)
# ---------------------------------------------------------------------------
if [ ! -f /etc/teamwerk/env ]; then
    mkdir -p /etc/teamwerk
    JWT_SECRET=$(openssl rand -hex 32)
    METRICS_TOKEN=$(openssl rand -hex 32)
    cat > /etc/teamwerk/env <<EOF
PORT=8080
BASE_URL=https://REPLACE_WITH_DOMAIN
DB_PATH=/var/lib/teamwerk/teamwerk.db
UPLOAD_DIR=/var/lib/teamwerk/uploads
FILES_DIR=/var/lib/teamwerk/files
JWT_SECRET=$JWT_SECRET
SMTP_HOST=mail.agenturserver.de
SMTP_PORT=587
SMTP_USER=REPLACE_WITH_SMTP_USER
SMTP_PASS=REPLACE_WITH_SMTP_PASSWORD
SMTP_FROM="TeamWERK <teamwerk@team-stuttgart.org>"
# Push-Notifications — Keys mit \`teamwerk gen-vapid\` erzeugen
VAPID_PUBLIC_KEY=REPLACE_WITH_VAPID_PUBLIC
VAPID_PRIVATE_KEY=REPLACE_WITH_VAPID_PRIVATE
VAPID_EMAIL=teamwerk@team-stuttgart.org
# Monitoring
LOG_FORMAT=json
METRICS_TOKEN=$METRICS_TOKEN
EOF
    chmod 600 /etc/teamwerk/env
    echo "⚠️  /etc/teamwerk/env angelegt — REPLACE_*-Werte ersetzen!"
fi

# ---------------------------------------------------------------------------
# 4. systemd-Service
# ---------------------------------------------------------------------------
cp teamwerk.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable teamwerk

# ---------------------------------------------------------------------------
# 5. Self-signed Cert (Übergang bis Domain + Certbot)
# ---------------------------------------------------------------------------
mkdir -p /etc/ssl/teamwerk
if [ ! -f /etc/ssl/teamwerk/cert.pem ]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout /etc/ssl/teamwerk/key.pem \
        -out /etc/ssl/teamwerk/cert.pem \
        -subj "/CN=intern.team-stuttgart.org"
fi

# ---------------------------------------------------------------------------
# 6. Nginx vhost
# ---------------------------------------------------------------------------
cp nginx-intern.conf /etc/nginx/sites-available/intern.team-stuttgart.org
ln -sf /etc/nginx/sites-available/intern.team-stuttgart.org /etc/nginx/sites-enabled/intern.team-stuttgart.org
nginx -t
systemctl enable nginx
if systemctl is-active --quiet nginx; then
    systemctl reload nginx
else
    systemctl start nginx
fi

# ---------------------------------------------------------------------------
# 7. Scheduler-Wrapper (lädt Env, sendet Heartbeat bei Erfolg)
# ---------------------------------------------------------------------------
HEARTBEAT_URL_FILE=/etc/teamwerk/heartbeat-url
if [ ! -f "$HEARTBEAT_URL_FILE" ]; then
    echo "REPLACE_WITH_BETTERSTACK_HEARTBEAT_URL" > "$HEARTBEAT_URL_FILE"
    chmod 600 "$HEARTBEAT_URL_FILE"
    echo "⚠️  $HEARTBEAT_URL_FILE angelegt — Better-Stack-Heartbeat-URL eintragen!"
fi

cat > /usr/local/bin/teamwerk-scheduler.sh <<'EOF'
#!/bin/bash
# Wrapper für cron — lädt Env aus /etc/teamwerk/env und sendet
# Better-Stack-Heartbeat nur bei erfolgreichem scheduler:run.
set -e
set -a
. /etc/teamwerk/env
set +a
/usr/local/bin/teamwerk scheduler:run
HEARTBEAT_URL=$(cat /etc/teamwerk/heartbeat-url 2>/dev/null || true)
if [ -n "$HEARTBEAT_URL" ] && [ "$HEARTBEAT_URL" != "REPLACE_WITH_BETTERSTACK_HEARTBEAT_URL" ]; then
    curl -fsS --retry 3 "$HEARTBEAT_URL" > /dev/null
fi
EOF
chmod +x /usr/local/bin/teamwerk-scheduler.sh

# ---------------------------------------------------------------------------
# 8. Cronjob (idempotent)
# ---------------------------------------------------------------------------
CRONJOB="* * * * * /usr/local/bin/teamwerk-scheduler.sh >> /var/log/teamwerk-scheduler.log 2>&1"
if ! crontab -l 2>/dev/null | grep -qF "/usr/local/bin/teamwerk-scheduler.sh"; then
    (crontab -l 2>/dev/null | grep -v "/usr/local/bin/teamwerk scheduler:run"; echo "$CRONJOB") | crontab -
fi

# ---------------------------------------------------------------------------
# 9. Logrotate (Scheduler-Log)
# ---------------------------------------------------------------------------
cat > /etc/logrotate.d/teamwerk <<'EOF'
/var/log/teamwerk-scheduler.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
EOF

# ---------------------------------------------------------------------------
# 10. Vector (Log-Shipper für Better Stack Logs) — optional
# ---------------------------------------------------------------------------
if ! command -v vector >/dev/null 2>&1; then
    curl -1sLf 'https://setup.vector.dev' | bash
    apt-get install -y vector
fi

BETTERSTACK_TOKEN_FILE=/etc/teamwerk/betterstack-logs-token
if [ ! -f "$BETTERSTACK_TOKEN_FILE" ]; then
    echo "REPLACE_WITH_BETTERSTACK_SOURCE_TOKEN" > "$BETTERSTACK_TOKEN_FILE"
    chmod 600 "$BETTERSTACK_TOKEN_FILE"
    echo "⚠️  $BETTERSTACK_TOKEN_FILE angelegt — Better-Stack-Logs-Source-Token eintragen!"
fi

BETTERSTACK_METRICS_TOKEN_FILE=/etc/teamwerk/betterstack-metrics-token
if [ ! -f "$BETTERSTACK_METRICS_TOKEN_FILE" ]; then
    echo "REPLACE_WITH_BETTERSTACK_METRICS_TOKEN" > "$BETTERSTACK_METRICS_TOKEN_FILE"
    chmod 600 "$BETTERSTACK_METRICS_TOKEN_FILE"
    echo "⚠️  $BETTERSTACK_METRICS_TOKEN_FILE angelegt — Better-Stack-Telemetry-Metrics-Token eintragen!"
fi

# Ingesting-Host der Telemetry-Source (pro Better-Stack-Source individuell; aus der UI kopieren).
BETTERSTACK_METRICS_ENDPOINT_FILE=/etc/teamwerk/betterstack-metrics-endpoint
if [ ! -f "$BETTERSTACK_METRICS_ENDPOINT_FILE" ]; then
    echo "REPLACE_WITH_BETTERSTACK_METRICS_INGESTING_HOST" > "$BETTERSTACK_METRICS_ENDPOINT_FILE"
    chmod 600 "$BETTERSTACK_METRICS_ENDPOINT_FILE"
    echo "⚠️  $BETTERSTACK_METRICS_ENDPOINT_FILE angelegt — Better-Stack-Telemetry-Ingesting-Host eintragen (z. B. s12345.eu-fsn-3.betterstackdata.com)!"
fi

BS_TOKEN=$(cat "$BETTERSTACK_TOKEN_FILE")
BS_METRICS_TOKEN=$(cat "$BETTERSTACK_METRICS_TOKEN_FILE")
BS_METRICS_HOST=$(cat "$BETTERSTACK_METRICS_ENDPOINT_FILE")
# METRICS_TOKEN aus /etc/teamwerk/env ziehen (von Vector für Prometheus-Scrape gegen /api/metrics gebraucht).
APP_METRICS_TOKEN=$(grep -E '^METRICS_TOKEN=' /etc/teamwerk/env | cut -d= -f2-)

cat > /etc/vector/vector.toml <<EOF
# === Logs (bestehend) =====================================================
[sources.teamwerk]
type = "journald"
include_units = ["teamwerk.service"]

[sinks.betterstack]
type = "http"
inputs = ["teamwerk"]
uri = "https://in.logs.betterstack.com"
encoding.codec = "json"

[sinks.betterstack.auth]
strategy = "bearer"
token = "$BS_TOKEN"

# === Metrics (Host + App via Prometheus-Scrape) ===========================
# Host-Telemetrie: füllt CPU/Memory/Network/Disk/Swap-Charts im Better-Stack-
# "Host(Vector)"-Dashboard.
[sources.host]
type = "host_metrics"
scrape_interval_secs = 30
collectors = ["cpu", "memory", "disk", "network", "filesystem", "load"]

# App-Metriken: scrapet teamwerk_* aus /api/metrics (Bearer-Token aus /etc/teamwerk/env).
[sources.teamwerk_app]
type = "prometheus_scrape"
endpoints = ["http://127.0.0.1:8080/api/metrics"]
scrape_interval_secs = 30
auth.strategy = "bearer"
auth.token = "$APP_METRICS_TOKEN"

# Beide Metrik-Streams in den austauschbaren Better-Stack-Telemetry-Sink.
[sinks.betterstack_metrics]
type = "prometheus_remote_write"
inputs = ["host", "teamwerk_app"]
endpoint = "https://$BS_METRICS_HOST"

[sinks.betterstack_metrics.auth]
strategy = "bearer"
token = "$BS_METRICS_TOKEN"
EOF

if ! grep -q "^VECTOR_CONFIG=" /etc/default/vector 2>/dev/null; then
    echo "VECTOR_CONFIG=/etc/vector/vector.toml" >> /etc/default/vector
fi

if [ "$BS_TOKEN" != "REPLACE_WITH_BETTERSTACK_SOURCE_TOKEN" ] \
    && [ "$BS_METRICS_TOKEN" != "REPLACE_WITH_BETTERSTACK_METRICS_TOKEN" ] \
    && [ "$BS_METRICS_HOST" != "REPLACE_WITH_BETTERSTACK_METRICS_INGESTING_HOST" ]; then
    systemctl enable --now vector
    systemctl restart vector
else
    echo "⚠️  Vector noch nicht gestartet — zuerst Logs-Token, Metrics-Token UND Metrics-Endpoint eintragen, dann: systemctl enable --now vector"
fi

# ---------------------------------------------------------------------------
echo ""
echo "✅ VPS-Setup abgeschlossen."
echo ""
echo "Nächste Schritte:"
echo "  1. /etc/teamwerk/env editieren — alle REPLACE_*-Werte ersetzen"
echo "     VAPID-Keys erzeugen: /usr/local/bin/teamwerk gen-vapid (nach erstem Deploy)"
echo "  2. /etc/teamwerk/heartbeat-url mit Better-Stack-URL füllen"
echo "  3. /etc/teamwerk/betterstack-logs-token mit Better-Stack-Logs-Source-Token füllen"
echo "  3b. /etc/teamwerk/betterstack-metrics-token mit Better-Stack-Telemetry-Metrics-Token füllen"
echo "      (in Better Stack: Telemetry → Sources → Vector / prometheus_remote_write)"
echo "  3c. /etc/teamwerk/betterstack-metrics-endpoint mit Ingesting-Host der Telemetry-Source"
echo "      füllen (z. B. s12345.eu-fsn-3.betterstackdata.com — Better Stack zeigt ihn"
echo "      pro Source unten neben dem Token an)"
echo "  4. Lokal: make deploy"
echo "  5. Auf VPS: /usr/local/bin/teamwerk migrate up --db /var/lib/teamwerk/teamwerk.db"
echo "  6. Admin anlegen: make create-admin-remote EMAIL=… PASSWORD=… NAME=…"
echo "  7. Smoke-Test: curl https://<DOMAIN>/api/healthz"
echo ""
echo "Hinweis: Self-signed Zertifikat aktiv — Domain + Certbot separat einrichten."
