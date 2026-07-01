#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# TeamWERK — Server-Umzug: Bootstrap eines neuen VPS
#
# Zieht die Produktions-DB, alle Storage-Ordner, /etc/teamwerk/env und die
# Better-Stack-Konfiguration vom bisherigen Server ($REMOTE) auf den neuen
# Server ($REMOTE_NEW). Nicht destruktiv gegen die Produktion — alle
# Zugriffe auf den Alt-Host sind read-only (Ausnahme: kurzlebige
# /tmp/teamwerk-migration.db, wird sofort wieder gelöscht).
#
# Idempotent: kann mehrfach laufen. Testdaten auf dem neuen Server werden
# beim Wiederholungslauf überschrieben.
#
# Voraussetzungen (in .env):
#   REMOTE          — SSH-Alias des Alt-Hosts
#   BASE_URL        — https://alt-domain
#   REMOTE_NEW      — SSH-Alias des Ziel-Hosts
#   REMOTE_NEW_DIR  — Binary-Verzeichnis auf Ziel (Default: /usr/local/bin)
#   BASE_URL_NEW    — https://neue-domain (bekommt der neue Host als BASE_URL)
#
# Aufruf: bash deploy/bootstrap-new-server.sh
# ---------------------------------------------------------------------------

set -euo pipefail

log()  { printf '\033[1;34m[bootstrap]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[bootstrap]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[bootstrap]\033[0m %s\n' "$*" >&2; exit 1; }

# --- Repo-Root sicherstellen -----------------------------------------------
if ! command -v git >/dev/null 2>&1; then
	die "git nicht im PATH."
fi
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[ -n "$REPO_ROOT" ] || die "Nicht in einem Git-Repo. Skript aus dem TeamWERK-Repo-Root starten."
cd "$REPO_ROOT"
[ -f Makefile ] && [ -f deploy/setup-vps.sh ] || die "Repo-Root sieht nicht wie TeamWERK aus (Makefile oder deploy/setup-vps.sh fehlt)."

# --- .env lesen -------------------------------------------------------------
[ -f .env ] || die ".env fehlt. Zuerst 'make env' laufen lassen."

env_get() {
	local key="$1"
	grep -E "^${key}=" .env | head -1 | cut -d= -f2- || true
}

REMOTE="$(env_get REMOTE)"
BASE_URL="$(env_get BASE_URL)"
REMOTE_NEW="$(env_get REMOTE_NEW)"
REMOTE_NEW_DIR="$(env_get REMOTE_NEW_DIR)"
BASE_URL_NEW="$(env_get BASE_URL_NEW)"
: "${REMOTE_NEW_DIR:=/usr/local/bin}"

missing=()
[ -n "$REMOTE" ]       || missing+=("REMOTE")
[ -n "$BASE_URL" ]     || missing+=("BASE_URL")
[ -n "$REMOTE_NEW" ]   || missing+=("REMOTE_NEW")
[ -n "$BASE_URL_NEW" ] || missing+=("BASE_URL_NEW")
if [ ${#missing[@]} -gt 0 ]; then
	die "Fehlende .env-Werte: ${missing[*]}. Siehe deploy/server-migration-runbook.md Abschnitt 0."
fi
case "$BASE_URL_NEW" in
	https://*) ;;
	*) die "BASE_URL_NEW muss mit 'https://' beginnen (aktuell: $BASE_URL_NEW)";;
esac

SOURCE_DOMAIN="${BASE_URL#https://}"
NEW_DOMAIN="${BASE_URL_NEW#https://}"

# --- Konfig-Übersicht + Bestätigung ----------------------------------------
cat <<EOF

  Quelle:  $REMOTE                ($BASE_URL)
  Ziel:    $REMOTE_NEW            ($BASE_URL_NEW)
  Ziel-Binary-Ort:  $REMOTE_NEW_DIR/teamwerk

  Ablauf:
    A) SSH-Konnektivität prüfen
    B) setup-vps.sh idempotent auf Ziel
    C) /etc/teamwerk/env klonen (BASE_URL-Rewrite)
    D) Better-Stack-Konfigurationsdateien klonen
    E) Ziel-Service stoppen (falls existent)
    F) sqlite3 .backup → Ziel-DB überschreiben
    G) Storage-Ordner rsync (Direkt zwischen Remotes, Fallback über Laptop)
    H) chown auf Ziel
    I) Binary bauen + deployen (make deploy gegen Ziel)
    J) Smoke-Test /api/healthz
    K) BASE_URL auf Ziel verifizieren

EOF

if [ "${AUTO_YES:-0}" != "1" ]; then
	printf "Fortfahren? [y/N] "
	read -r ans
	case "$ans" in y|Y) ;; *) log "Abgebrochen."; exit 1;; esac
fi

# --- A) SSH-Konnektivität --------------------------------------------------
log "A) SSH-Konnektivität prüfen"
for host in "$REMOTE" "$REMOTE_NEW"; do
	if ! ssh -o BatchMode=yes -o ConnectTimeout=8 "$host" 'echo ok' >/dev/null 2>&1; then
		die "SSH zu '$host' scheiterte. Alias in ~/.ssh/config prüfen (Key, HostName, User)."
	fi
	log "   ok: $host"
done

# --- A2) rsync auf Ziel sicherstellen (Bootstrap braucht es, setup-vps.sh selbst auch) ---
log "A2) rsync auf $REMOTE_NEW sicherstellen"
if ! ssh "$REMOTE_NEW" 'command -v rsync >/dev/null 2>&1'; then
	log "   rsync fehlt, installiere per apt-get"
	ssh "$REMOTE_NEW" 'sudo apt-get update -qq && sudo apt-get install -y -qq rsync'
else
	log "   rsync bereits vorhanden"
fi

# --- B) setup-vps.sh auf Ziel (ohne Nginx — Nginx macht B2) ---------------
log "B) setup-vps.sh auf $REMOTE_NEW (SKIP_NGINX=1, idempotent)"
rsync -az deploy/ "$REMOTE_NEW:/tmp/teamwerk-deploy/"
ssh "$REMOTE_NEW" 'cd /tmp/teamwerk-deploy && sudo SKIP_NGINX=1 bash setup-vps.sh'

# --- B2) Nginx vhost mit neuer Domain + Self-signed-Cert -------------------
# Erstlauf: neue Config + Self-signed-Cert. Folgelauf (Config existiert bereits
# und referenziert /etc/letsencrypt/live/...): NICHT anfassen, sonst wird der
# von Certbot eingespielte LE-Cert-Pfad überschrieben und der Server läuft
# wieder auf Self-signed (Browser blockt → „nicht erreichbar").
CONF_PATH="/etc/nginx/sites-available/${NEW_DOMAIN}"
if ssh "$REMOTE_NEW" "sudo test -f ${CONF_PATH} && sudo grep -q '/etc/letsencrypt/live/${NEW_DOMAIN}' ${CONF_PATH}"; then
	log "B2) Nginx vhost existiert bereits mit Let's-Encrypt-Cert-Pfaden — Config bleibt unangetastet"
else
	log "B2) Nginx vhost für $NEW_DOMAIN neu anlegen (Self-signed als Übergang bis Certbot)"
	# Alte partielle Config aufräumen (falls ein vorheriger Bootstrap-Lauf sie hinterlegt hat)
	ssh "$REMOTE_NEW" 'sudo rm -f /etc/nginx/sites-enabled/intern.team-stuttgart.org /etc/nginx/sites-available/intern.team-stuttgart.org'
	# Self-signed-Cert mit CN=NEW_DOMAIN (überschreibt den setup-vps-Default-CN)
	ssh "$REMOTE_NEW" "sudo openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
	    -keyout /etc/ssl/teamwerk/key.pem \
	    -out /etc/ssl/teamwerk/cert.pem \
	    -subj '/CN=${NEW_DOMAIN}' 2>/dev/null"
	# vhost-Config aus nginx-intern.conf ableiten:
	#  - server_name → NEW_DOMAIN
	#  - Cert-Pfade → self-signed (Certbot ersetzt sie später mit --nginx automatisch)
	sed \
	    -e "s|internal\.team-stuttgart\.org|${NEW_DOMAIN}|g" \
	    -e "s|/etc/letsencrypt/live/${NEW_DOMAIN}/fullchain.pem|/etc/ssl/teamwerk/cert.pem|g" \
	    -e "s|/etc/letsencrypt/live/${NEW_DOMAIN}/privkey.pem|/etc/ssl/teamwerk/key.pem|g" \
	    deploy/nginx-intern.conf \
	    | ssh "$REMOTE_NEW" "sudo tee ${CONF_PATH} > /dev/null"
	ssh "$REMOTE_NEW" "sudo ln -sf ${CONF_PATH} /etc/nginx/sites-enabled/${NEW_DOMAIN}"
fi
# limit_req_zone im http{}-Kontext ergänzen, wenn noch nicht drin (idempotent)
ssh "$REMOTE_NEW" '
    if ! grep -q "zone=teamwerk_auth" /etc/nginx/nginx.conf; then
        sudo sed -i "/^http {/a \    limit_req_zone \$binary_remote_addr zone=teamwerk_auth:10m rate=20r/m;" /etc/nginx/nginx.conf
    fi
'
ssh "$REMOTE_NEW" 'sudo nginx -t'
ssh "$REMOTE_NEW" '
    if systemctl is-active --quiet nginx; then
        sudo systemctl reload nginx
    else
        sudo systemctl start nginx
    fi
'

# --- C) Env klonen mit BASE_URL-Rewrite -----------------------------------
log "C) /etc/teamwerk/env klonen (BASE_URL → $BASE_URL_NEW)"
ssh "$REMOTE" 'sudo cat /etc/teamwerk/env' \
	| sed -E "s|^BASE_URL=.*|BASE_URL=${BASE_URL_NEW}|" \
	| ssh "$REMOTE_NEW" 'sudo tee /etc/teamwerk/env > /dev/null && sudo chmod 600 /etc/teamwerk/env'

# --- D) Better-Stack-Konfig-Dateien ----------------------------------------
log "D) Better-Stack-Konfig-Dateien klonen"
for f in heartbeat-url betterstack-logs-token betterstack-metrics-token betterstack-metrics-endpoint; do
	if ssh "$REMOTE" "sudo test -f /etc/teamwerk/$f"; then
		ssh "$REMOTE" "sudo cat /etc/teamwerk/$f" \
			| ssh "$REMOTE_NEW" "sudo tee /etc/teamwerk/$f > /dev/null && sudo chmod 600 /etc/teamwerk/$f"
		log "   kopiert: $f"
	else
		log "   übersprungen (Quelle hat kein /etc/teamwerk/$f): $f"
	fi
done

# --- E) Ziel-Service stoppen ------------------------------------------------
log "E) Ziel-Service stoppen (falls schon vorhanden)"
ssh "$REMOTE_NEW" 'sudo systemctl stop teamwerk 2>/dev/null || true'

# --- F) DB-Snapshot ---------------------------------------------------------
DB_PATH=/var/lib/teamwerk/teamwerk.db
log "F) DB-Snapshot $REMOTE:$DB_PATH → $REMOTE_NEW:$DB_PATH"
ssh "$REMOTE" "sudo sqlite3 $DB_PATH '.backup /tmp/teamwerk-migration.db' && sudo chmod 644 /tmp/teamwerk-migration.db"
ssh "$REMOTE" 'sudo cat /tmp/teamwerk-migration.db' \
	| ssh "$REMOTE_NEW" "sudo mkdir -p $(dirname $DB_PATH) && sudo tee $DB_PATH > /dev/null && sudo rm -f ${DB_PATH}-wal ${DB_PATH}-shm"
ssh "$REMOTE" 'sudo rm -f /tmp/teamwerk-migration.db'

# --- G) Storage-Ordner ------------------------------------------------------
log "G) Storage-Ordner synchronisieren"
STORAGE_DIRS=(
	/var/lib/teamwerk/uploads
	/var/lib/teamwerk/files
	/var/lib/teamwerk/beitragslauf-protokolle
	/storage/videos
)
for d in "${STORAGE_DIRS[@]}"; do
	if ! ssh "$REMOTE" "sudo test -d $d"; then
		log "   übersprungen (Quelle hat kein $d)"
		continue
	fi
	log "   rsync $d"
	if ssh "$REMOTE" "sudo rsync -az -e ssh $d/ $REMOTE_NEW:$d/" 2>/dev/null; then
		continue
	fi
	warn "   Direkt-Rsync fehlgeschlagen, fallback über Laptop-Disk (~kann dauern und puffert transient)"
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064
	trap "rm -rf '$tmp'" EXIT
	rsync -az "$REMOTE:$d/" "$tmp/"
	rsync -az "$tmp/" "$REMOTE_NEW:$d/"
	rm -rf "$tmp"
	trap - EXIT
done

# --- H) Owner-Fix -----------------------------------------------------------
log "H) chown -R www-data:www-data auf Ziel"
ssh "$REMOTE_NEW" '
	sudo chown -R www-data:www-data /var/lib/teamwerk 2>/dev/null || true
	sudo chown -R www-data:www-data /storage 2>/dev/null || true
'

# --- I) Binary deployen (nutzt bestehendes make-deploy) --------------------
log "I) Binary bauen + auf $REMOTE_NEW deployen (make deploy)"
make deploy REMOTE="$REMOTE_NEW" REMOTE_DIR="$REMOTE_NEW_DIR"

# --- J) Smoke-Test ----------------------------------------------------------
log "J) Smoke-Test /api/healthz (über Ziel-Loopback + Host-Header)"
resp="$(ssh "$REMOTE_NEW" "curl -k -s -H 'Host: $NEW_DOMAIN' https://localhost/api/healthz" || true)"
log "   Response: $resp"
if ! echo "$resp" | grep -q '"status":"ok"' || ! echo "$resp" | grep -q '"db":"ok"'; then
	die "Smoke-Test fehlgeschlagen. Journal prüfen: ssh $REMOTE_NEW 'journalctl -u teamwerk -n 50 --no-pager'"
fi

# --- K) BASE_URL verifizieren -----------------------------------------------
log "K) BASE_URL auf Ziel:"
ssh "$REMOTE_NEW" "sudo grep '^BASE_URL=' /etc/teamwerk/env"

cat <<EOF

  Bootstrap fertig. Alt-Host ($REMOTE) läuft unverändert weiter.

  Nächste Schritte (manuell, wenn du bereit bist):

    1. Testphase: lokale /etc/hosts-Zeile setzen, damit nur dein Browser
       auf den neuen Server läuft:

         echo "\$(ssh -G $REMOTE_NEW | awk '/^hostname /{print \$2}')  $NEW_DOMAIN" \\
           | sudo tee -a /etc/hosts

    2. Frischen Daten-Sync (jederzeit wiederholbar, überschreibt Testdaten):

         bash deploy/bootstrap-new-server.sh   # dasselbe Skript

       oder für einen schlanken Sync ohne setup-vps:

         make server-sync-data NEW_REMOTE=$REMOTE_NEW

    3. Wenn du bereit für den echten Umzug bist:
       - DNS-A-Record $NEW_DOMAIN → Ziel-IP im Provider-Panel
       - Certbot auf Ziel:
           ssh $REMOTE_NEW "certbot --nginx -d $NEW_DOMAIN --non-interactive --agree-tos -m vorstand@team-stuttgart.org"
       - Cutover: make server-cutover NEW_REMOTE=$REMOTE_NEW
       - Runbook: deploy/server-migration-runbook.md

EOF
