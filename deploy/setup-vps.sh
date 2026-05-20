#!/usr/bin/env bash
# Run once on the IONOS VPS to set up the server environment.
# Usage: bash setup-vps.sh

set -euo pipefail

# Install Nginx if not present
apt-get update && apt-get install -y nginx openssl

# Create app data directory
mkdir -p /var/lib/teamwerk
chown www-data:www-data /var/lib/teamwerk

# Create upload directories
mkdir -p /var/lib/teamwerk/storage/uploads/{member-photos,user-photos,sepa-mandats}
chown -R www-data:www-data /var/lib/teamwerk/storage/

# Create env file (skip if already configured)
if [ ! -f /etc/teamwerk/env ]; then
    mkdir -p /etc/teamwerk
    cat > /etc/teamwerk/env <<'EOF'
PORT=8080
DB_PATH=/var/lib/teamwerk/teamwerk.db
JWT_SECRET=REPLACE_WITH_RANDOM_SECRET
UPLOAD_DIR=/var/lib/teamwerk/storage/uploads
SMTP_HOST=mail.agenturserver.de
SMTP_PORT=587
SMTP_USER=p459264p5
SMTP_PASS=REPLACE_WITH_SMTP_PASSWORD
SMTP_FROM=TeamWERK <teamwerk@team-stuttgart.org>
EOF
    chmod 600 /etc/teamwerk/env
    echo "⚠️  Bitte /etc/teamwerk/env mit echten Werten befüllen!"
fi

# Install systemd service
cp teamwerk.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable teamwerk

# Generate self-signed certificate
mkdir -p /etc/ssl/teamwerk
if [ ! -f /etc/ssl/teamwerk/cert.pem ]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout /etc/ssl/teamwerk/key.pem \
        -out /etc/ssl/teamwerk/cert.pem \
        -subj "/CN=intern.team-stuttgart.org"
fi

# Install Nginx vhost config
cp nginx-intern.conf /etc/nginx/sites-available/intern.team-stuttgart.org
ln -sf /etc/nginx/sites-available/intern.team-stuttgart.org /etc/nginx/sites-enabled/intern.team-stuttgart.org
nginx -t
systemctl enable nginx
if systemctl is-active --quiet nginx; then
    systemctl reload nginx
else
    systemctl start nginx
fi

# Add scheduler cronjob (idempotent)
CRONJOB="* * * * * /usr/local/bin/teamwerk scheduler:run >> /var/log/teamwerk-scheduler.log 2>&1"
if ! crontab -l 2>/dev/null | grep -qF "$CRONJOB"; then
    (crontab -l 2>/dev/null; echo "$CRONJOB") | crontab -
fi

echo "VPS setup complete. Deploy the binary with: make deploy"
echo "Hinweis: Self-signed Zertifikat – Browser zeigt Sicherheitswarnung."
