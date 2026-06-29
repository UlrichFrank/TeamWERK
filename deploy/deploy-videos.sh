#!/usr/bin/env bash
# Vollständiges Deployment der Spielvideo-Funktion auf den Produktiv-VPS.
#
# Was es macht (idempotent — kann mehrfach laufen, überschreibt nichts):
#   1. Vorprüfung lokal:   .env vorhanden, REMOTE-Alias gesetzt, SSH erreichbar.
#   2. VPS-Vorbereitung:   ffmpeg installiert, /storage/videos/{uploads,raw,processed}
#                          mit Owner www-data, freie Disk gemeldet.
#   3. Env-Ergänzung:      VIDEO_STREAM_SECRET in /etc/teamwerk/env anhängen,
#                          falls fehlt (Secret wird auf dem VPS erzeugt — verlässt
#                          den Server nicht). Bestehende Werte werden NICHT ersetzt.
#   4. Deploy:             ruft `make deploy` (build → rsync → migrate up →
#                          systemctl restart) im Repo-Root auf.
#   5. Smoke-Tests:        Service-Status, /api/healthz, ffmpeg-Version.
#
# Nicht im Scope:
#   • Disk-Vergrößerung von /storage (manuell beim Provider, siehe Runbook).
#   • Domain/Certbot (separat).
#   • Manueller E2E-Test (Upload → Transcode → Player) — wird am Ende verlinkt.
#
# Usage:
#   bash deploy/deploy-videos.sh           # interaktiv (fragt vor schreibenden Schritten)
#   bash deploy/deploy-videos.sh --yes     # nicht-interaktiv (für CI/Wiederholungslauf)

set -euo pipefail

# ---------------------------------------------------------------------------
# 0. Setup
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

ASSUME_YES=0
if [[ "${1:-}" == "--yes" ]] || [[ "${1:-}" == "-y" ]]; then
    ASSUME_YES=1
fi

# ANSI ohne externe Abhängigkeiten.
RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; CYAN=$'\033[36m'; RESET=$'\033[0m'
step()  { echo "${CYAN}▶ $*${RESET}"; }
ok()    { echo "${GREEN}✓ $*${RESET}"; }
warn()  { echo "${YELLOW}⚠ $*${RESET}"; }
fail()  { echo "${RED}✗ $*${RESET}" >&2; exit 1; }

confirm() {
    local prompt="$1"
    if [[ "$ASSUME_YES" == "1" ]]; then return 0; fi
    read -r -p "$prompt [y/N] " ans
    [[ "$ans" =~ ^[yY]$ ]]
}

# ---------------------------------------------------------------------------
# 1. Lokale Voraussetzungen
# ---------------------------------------------------------------------------
step "Lokale Voraussetzungen prüfen"

[[ -f .env ]] || fail ".env fehlt im Repo-Root."

# REMOTE aus .env ziehen (Format: REMOTE=user@host oder REMOTE=ssh-alias)
REMOTE="$(grep -E '^REMOTE=' .env | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" || true)"
[[ -n "$REMOTE" ]] || fail "REMOTE in .env nicht gesetzt."

command -v ssh  >/dev/null || fail "ssh nicht installiert."
command -v make >/dev/null || fail "make nicht installiert."

# Frühe SSH-Probe (BatchMode=yes → fragt nicht nach Passwort, scheitert sofort).
ssh -o BatchMode=yes -o ConnectTimeout=10 "$REMOTE" "true" \
    || fail "SSH zu '$REMOTE' fehlgeschlagen. Key-Auth eingerichtet?"
ok "SSH zu $REMOTE erreichbar."

# Klare Annahme: Branch ist gepusht. Lokale Uncommitted Changes sind erlaubt
# (make deploy baut aus dem Working Tree), warnen aber.
if ! git diff --quiet || ! git diff --cached --quiet; then
    warn "Uncommitted Änderungen im Working Tree — make deploy baut daraus."
fi

# ---------------------------------------------------------------------------
# 2. VPS-Vorbereitung (ffmpeg, Storage)
# ---------------------------------------------------------------------------
step "VPS-Vorbereitung"

# Single SSH-Session, damit jeder Substep im Log einzeln sichtbar wird.
ssh "$REMOTE" 'bash -se' <<'REMOTE_SETUP'
set -euo pipefail

# 2a. ffmpeg
if ! command -v ffmpeg >/dev/null 2>&1; then
    echo "  ffmpeg fehlt → apt-get install"
    DEBIAN_FRONTEND=noninteractive apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq ffmpeg
fi
FFMPEG_VERSION="$(ffmpeg -version 2>/dev/null | head -1)"
echo "  ffmpeg: $FFMPEG_VERSION"

# 2b. Storage
mkdir -p /storage/videos/uploads /storage/videos/raw /storage/videos/processed
chown -R www-data:www-data /storage/videos
echo "  /storage/videos/{uploads,raw,processed} ✓ (www-data)"

# 2c. Disk-Reality-Check — Warnung, kein Abbruch (Disk-Erweiterung ist manuell).
FREE_GB="$(df -BG --output=avail /storage 2>/dev/null | tail -1 | tr -dc '0-9')"
if [[ -z "$FREE_GB" ]]; then FREE_GB=0; fi
echo "  /storage frei: ${FREE_GB} GB"
if [[ "$FREE_GB" -lt 20 ]]; then
    echo "  ⚠ < 20 GB frei — Faustregel: ~3–4 GB pro vorgehaltener Stunde Video."
    echo "    Volume vergrößern, bevor die Vereinsmitglieder Uploads starten."
fi
REMOTE_SETUP
ok "VPS-Vorbereitung abgeschlossen."

# ---------------------------------------------------------------------------
# 3. VIDEO_STREAM_SECRET in /etc/teamwerk/env eintragen (nur wenn fehlt)
# ---------------------------------------------------------------------------
step "Env-Datei prüfen"

ENV_HAS_SECRET="$(ssh "$REMOTE" "grep -c '^VIDEO_STREAM_SECRET=' /etc/teamwerk/env 2>/dev/null || true")"
ENV_HAS_SECRET="${ENV_HAS_SECRET:-0}"

if [[ "$ENV_HAS_SECRET" -gt 0 ]]; then
    ok "VIDEO_STREAM_SECRET bereits gesetzt — nicht angefasst."
else
    warn "VIDEO_STREAM_SECRET fehlt in /etc/teamwerk/env."
    echo "  Es wird ein neues 32-Byte-Hex-Secret auf dem VPS erzeugt und an die"
    echo "  Env-Datei angehängt. Das Secret verlässt den Server nicht (keine"
    echo "  lokale Anzeige). Rotation invalidiert nur laufende Stream-Sessions,"
    echo "  NICHT die JWTs/Logins."
    if confirm "  Eintragen?"; then
        ssh "$REMOTE" 'bash -se' <<'REMOTE_SECRET'
set -euo pipefail
SECRET="$(openssl rand -hex 32)"
# An /etc/teamwerk/env anhängen (Datei existiert, Mode 600 vom Setup).
{
    echo ""
    echo "# Spielvideo-Stream-Token (HMAC, getrennt von JWT_SECRET)."
    echo "# Rotation: alter Wert weg → ausgestellte Stream-Token brechen sofort."
    echo "VIDEO_STREAM_SECRET=$SECRET"
} >> /etc/teamwerk/env
chmod 600 /etc/teamwerk/env
echo "  VIDEO_STREAM_SECRET in /etc/teamwerk/env eingetragen."
REMOTE_SECRET
        ok "Secret gesetzt."
    else
        fail "Abbruch — ohne VIDEO_STREAM_SECRET startet der Server in Prod nicht."
    fi
fi

# Optionale Defaults nur als Hinweis — Code fällt auf Default zurück, wenn fehlt.
ssh "$REMOTE" '
    if ! grep -q "^VIDEO_STORAGE_DIR="    /etc/teamwerk/env; then echo "  ℹ VIDEO_STORAGE_DIR nicht gesetzt — Default /storage/videos wird verwendet."; fi
    if ! grep -q "^VIDEO_RESERVED_BYTES=" /etc/teamwerk/env; then echo "  ℹ VIDEO_RESERVED_BYTES nicht gesetzt — Default 2 GiB wird verwendet."; fi
'

# ---------------------------------------------------------------------------
# 4. Deploy (Build + Rsync + Migrate + Restart)
# ---------------------------------------------------------------------------
step "Deploy (make deploy)"
echo "  → baut Linux-Binary, rsync auf $REMOTE, migrate up, systemctl restart teamwerk"
if confirm "  Jetzt ausführen?"; then
    make deploy
    ok "Deploy abgeschlossen."
else
    warn "Deploy übersprungen — Setup-Schritte sind bereits angewendet."
    exit 0
fi

# ---------------------------------------------------------------------------
# 5. Smoke-Tests
# ---------------------------------------------------------------------------
step "Smoke-Tests"

# Service muss laufen — wenn LoadConfig nach unserem Env-Edit doch noch streikt,
# fängt das hier den Fehler.
ssh "$REMOTE" 'systemctl is-active --quiet teamwerk' \
    || fail "teamwerk.service ist nicht active. → ssh $REMOTE 'journalctl -u teamwerk -n 50'"
ok "teamwerk.service active."

# /api/healthz lokal auf dem VPS gegen 127.0.0.1 (umgeht TLS/Nginx-Fallstricke).
HEALTH="$(ssh "$REMOTE" 'curl -fsS -m 5 http://127.0.0.1:8080/api/healthz || true')"
[[ -n "$HEALTH" ]] || fail "/api/healthz lieferte nichts. → journalctl -u teamwerk -n 50"
ok "/api/healthz: $HEALTH"

# Schema-Version: 013 (videos) muss applied sein.
DB_VERSION="$(ssh "$REMOTE" "sqlite3 /var/lib/teamwerk/teamwerk.db 'SELECT version, dirty FROM schema_migrations;' 2>/dev/null || echo unknown")"
if [[ "$DB_VERSION" == 13\|0 ]] || [[ "$DB_VERSION" == 13* ]]; then
    ok "Schema-Migration: $DB_VERSION"
else
    warn "Schema-Version unerwartet ($DB_VERSION) — erwartet 13|0."
fi

# ---------------------------------------------------------------------------
# 6. Hinweise auf manuelle Resttests (laut PR-Beschreibung Task 11.4)
# ---------------------------------------------------------------------------
cat <<EOF

${GREEN}✅ Video-Deployment abgeschlossen.${RESET}

Letzter Schritt — manueller E2E-Test (nicht automatisierbar):
  1. https://<DOMAIN>/videos öffnen → leere Liste, keine 500
  2. Test-Video (< 100 MB) hochladen → status=queued → ready
  3. Player abspielen (Chrome + Safari) → HLS-Token greift
  4. Push-Notification beim Uploader/Spielern angekommen
  5. ssh $REMOTE 'df -h /storage' nach dem Test — Pufferreserve?

Rollback bei Bedarf:
  ssh $REMOTE 'journalctl -u teamwerk -f'
  ssh $REMOTE 'systemctl restart teamwerk'

Secret-Rotation (falls Token-Leak):
  ssh $REMOTE 'openssl rand -hex 32'                          # neuer Wert
  ssh $REMOTE 'sed -i s/^VIDEO_STREAM_SECRET=.*/VIDEO_STREAM_SECRET=<neu>/ /etc/teamwerk/env'
  ssh $REMOTE 'systemctl restart teamwerk'                     # laufende Streams brechen
EOF
