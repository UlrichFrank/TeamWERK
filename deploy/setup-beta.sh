#!/usr/bin/env bash
# Richtet die Beta-Instanz auf dem VPS ein (idempotent, als root via sudo).
# Aufruf über `make setup-beta`. Berührt Prod nicht:
#   - eigene Pfade:   /usr/local/bin/teamwerk-beta, /etc/teamwerk-beta/env,
#                     /var/lib/teamwerk-beta/ (DB + Storage + Videos)
#   - eigener Dienst: teamwerk-beta (Port 8081)
#   - eigene nginx-Site: sites-available/teamwerk-beta; vor jedem Reload `nginx -t`,
#     bei Fehler wird nur die Beta-Site wieder entfernt und abgebrochen
#   - kein Scheduler-Cron, kein Backup-Cron, keine Änderung an nginx.conf
set -euo pipefail

HOST=beta.teamwerk.team-stuttgart.org
HERE="$(cd "$(dirname "$0")" && pwd)"
DATA=/var/lib/teamwerk-beta
ENV=/etc/teamwerk-beta/env
SITE=/etc/nginx/sites-available/teamwerk-beta
LINK=/etc/nginx/sites-enabled/teamwerk-beta
CERT=/etc/letsencrypt/live/$HOST/fullchain.pem

echo "==> Verzeichnisse"
for d in "$DATA" "$DATA"/{uploads,files,media,beitragslauf-protokolle,match-report-images,training-diary,bwhv-reports,videos}; do
  mkdir -p "$d"
done
chown -R www-data:www-data "$DATA"
mkdir -p /var/www/certbot

echo "==> Env ($ENV)"
if [ ! -f "$ENV" ]; then
  mkdir -p /etc/teamwerk-beta
  umask 077
  cat > "$ENV" <<EOF
PORT=8081
BASE_URL=https://$HOST
DB_PATH=$DATA/teamwerk.db
JWT_SECRET=$(openssl rand -base64 48 | tr -d '\n')
VIDEO_STREAM_SECRET=$(openssl rand -base64 48 | tr -d '\n')
LOG_FORMAT=json
HSTS_ENABLED=false
# Keine Außenwirkung: kein Mailversand, kein Push (keine VAPID-Schlüssel),
# kein Abruf beim Verband.
MAILER_DISABLED=true
BWHV_ORG_ID=0
UPLOAD_DIR=$DATA/uploads
FILES_DIR=$DATA/files
MEDIA_DIR=$DATA/media
BEITRAGSLAUF_DIR=$DATA/beitragslauf-protokolle
MATCH_REPORT_IMAGE_DIR=$DATA/match-report-images
TRAINING_DIARY_DIR=$DATA/training-diary
BWHV_REPORT_DIR=$DATA/bwhv-reports
VIDEO_STORAGE_DIR=$DATA/videos
EOF
  chmod 600 "$ENV"
  echo "  angelegt"
else
  echo "  vorhanden, unverändert"
fi

echo "==> systemd-Dienst teamwerk-beta"
install -m 644 "$HERE/teamwerk-beta.service" /etc/systemd/system/teamwerk-beta.service
systemctl daemon-reload
systemctl enable teamwerk-beta >/dev/null 2>&1

reload_nginx() {
  if nginx -t 2>/dev/null; then
    systemctl reload nginx
  else
    nginx -t || true
    echo "nginx -t fehlgeschlagen — Beta-Site wird entfernt, Prod-Konfiguration bleibt wie sie war" >&2
    rm -f "$LINK" "$SITE"
    nginx -t && systemctl reload nginx
    exit 1
  fi
}

echo "==> nginx"
if [ ! -f "$CERT" ]; then
  # Ohne Zertifikat kann der volle Block (443) nicht laden. Übergangsweise nur
  # Port 80 mit ACME-Webroot, damit certbot das Zertifikat holen kann.
  cat > "$SITE" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name $HOST;
    location /.well-known/acme-challenge/ { root /var/www/certbot; }
    location / { return 503; }
}
EOF
  ln -sf "$SITE" "$LINK"
  reload_nginx

  MYIP="$(curl -fsS -4 https://api.ipify.org 2>/dev/null || true)"
  DNSIP="$(getent ahostsv4 "$HOST" 2>/dev/null | awk 'NR==1{print $1}')"
  if [ -z "$DNSIP" ] || [ "$DNSIP" != "$MYIP" ]; then
    echo "  DNS: $HOST löst nicht auf diesen Server auf (DNS='${DNSIP:-keiner}', Server='$MYIP')."
    echo "  A-Record $HOST -> $MYIP anlegen, dann 'make setup-beta' erneut ausführen."
    exit 0
  fi
  certbot certonly --webroot -w /var/www/certbot -d "$HOST" \
    --non-interactive --agree-tos --register-unsafely-without-email --keep-until-expiring
fi

install -m 644 "$HERE/nginx-teamwerk-beta.conf" "$SITE"
ln -sf "$SITE" "$LINK"
reload_nginx
echo "==> fertig: https://$HOST (Dienst teamwerk-beta, Port 8081)"
