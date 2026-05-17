#!/usr/bin/env bash
# Run once on the IONOS VPS to set up the server environment.
# Usage: bash setup-vps.sh

set -euo pipefail

# Install Nginx
apt-get update && apt-get install -y nginx certbot python3-certbot-nginx

# Create app data directory
mkdir -p /var/lib/vereinswerk
chown www-data:www-data /var/lib/vereinswerk

# Create env directory
mkdir -p /etc/vereinswerk
cat > /etc/vereinswerk/env <<'EOF'
PORT=8080
DB_PATH=/var/lib/vereinswerk/vereinswerk.db
JWT_SECRET=REPLACE_WITH_RANDOM_SECRET
SMTP_HOST=smtp.mittwald.de
SMTP_PORT=587
SMTP_USER=vorstand@team-stuttgart.org
SMTP_PASS=REPLACE_WITH_SMTP_PASSWORD
SMTP_FROM=VereinsWerk <vorstand@team-stuttgart.org>
EOF
chmod 600 /etc/vereinswerk/env

# Install systemd service
cp vereinswerk.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable vereinswerk

# Install Nginx config
cp nginx-intern.conf /etc/nginx/sites-available/intern.team-stuttgart.org
ln -sf /etc/nginx/sites-available/intern.team-stuttgart.org /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx

# Obtain SSL certificate
certbot --nginx -d intern.team-stuttgart.org --non-interactive --agree-tos -m webmaster@team-stuttgart.org

# Add scheduler Cronjob
(crontab -l 2>/dev/null; echo "* * * * * /usr/local/bin/vereinswerk scheduler:run >> /var/log/vereinswerk-scheduler.log 2>&1") | crontab -

echo "VPS setup complete. Deploy the binary with: make deploy"
