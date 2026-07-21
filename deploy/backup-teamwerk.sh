#!/usr/bin/env bash
# =============================================================================
# TeamWERK — externes Backup vom Mittwald-Host (Pull-Modell)
# =============================================================================
#
# Läuft täglich per Cron auf einem Mittwald-Host, zieht sich per SSH einen
# konsistenten Snapshot vom Prod-VPS (teamwerk.team-stuttgart.org) und legt ihn
# lokal mit GFS-Retention ab.
#
# Was wird gesichert
#   - SQLite-DB (/var/lib/teamwerk/teamwerk.db) — konsistent via `sqlite3 .backup`
#   - PII/Documents/Uploads/Protokolle (/var/lib/teamwerk/*)
#   - Vereins-Config (/etc/teamwerk/env — enthält Secrets, Datei-Permissions am Ziel = 600)
#   - Video-Raws (/storage/videos/raw) — separat, nur EIN aktueller Stand
#
# Retention
#   Kleine Daten (DB + PII/Docs/Uploads/Protokolle + Config):
#     daily/    → 7 Stück (letzte 7 Tage)
#     monthly/  → 6 Stück (jeweils Backup vom 1. eines Monats)
#     yearly/   → 3 Stück (jeweils Backup vom 1. Januar)
#   Videos (Raw-Uploads):
#     videos-latest/ → 1 rsync-Stand, wird bei jedem Lauf überschrieben.
#                      Wenn der letzte erfolgreiche Sync älter als 7 Tage ist,
#                      wird das Verzeichnis geleert (kein falsches Sicherheitsgefühl).
#
# Setup (einmalig)
#   VPS-Seite ist bereits vorbereitet: User `tw-backup` (UID 1000) hat
#   - Group www-data → Read auf /var/lib/teamwerk und /storage
#   - ACL u:tw-backup:r auf /etc/teamwerk/env (ausschließlich env, keine
#     anderen Secrets im Ordner)
#   - Kein Write, kein sudo, keine anderen Ordner-Zugriffe
#   - SSH nur via Key (kein Passwort gesetzt)
#
#   Auf Mittwald noch zu erledigen:
#   1. SSH-Keypaar erzeugen:
#        ssh-keygen -t ed25519 -f ~/.ssh/teamwerk_backup -N ''
#   2. Public Key auf dem VPS eintragen (als root):
#        cat ~/.ssh/teamwerk_backup.pub | ssh root@teamwerk.team-stuttgart.org \
#          "install -m 600 -o tw-backup -g tw-backup /dev/stdin \
#           /home/tw-backup/.ssh/authorized_keys"
#   3. Verbindung testen:
#        ssh -i ~/.ssh/teamwerk_backup tw-backup@teamwerk.team-stuttgart.org \
#          'sqlite3 -readonly /var/lib/teamwerk/teamwerk.db "SELECT COUNT(*) FROM members"'
#   4. Ziel-Verzeichnis auf Mittwald: mkdir -p ~/teamwerk-backup
#   5. Cronjob auf Mittwald (Web-UI oder crontab -e):
#        15 3 * * *  BACKUP_ROOT=/backup bash /backup/backup-teamwerk.sh >> /backup/backup.log 2>&1
#      HINWEIS: Mittwald-Home /backup ist noexec — Skript daher explizit via
#      `bash /pfad/...` starten (NICHT direkt ausführen).
#   6. Nach dem ersten Lauf: Restore einmal auf Test-VPS durchspielen.
#      Ungetestete Backups sind keine Backups.
#
# Restore (Kurzform, Details am Ende der Datei)
#   scp teamwerk-YYYY-MM-DD.tar.gz  root@vps:/tmp/
#   ssh root@vps 'systemctl stop teamwerk && tar -xzf /tmp/teamwerk-*.tar.gz -C / \
#     && systemctl start teamwerk'
#
# Konfiguration via Env-Variablen (oder oben im Skript defaulten)
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Konfiguration
# ---------------------------------------------------------------------------
: "${VPS_SSH:=tw-backup@teamwerk.team-stuttgart.org}" # SSH-Ziel (least-privilege user)
: "${SSH_KEY:=$HOME/.ssh/teamwerk_backup}"            # Private Key
: "${BACKUP_ROOT:=$HOME/teamwerk-backup}"             # Lokale Backup-Wurzel
: "${VPS_DB:=/var/lib/teamwerk/teamwerk.db}"          # Pfad DB auf VPS
: "${VPS_VARLIB:=/var/lib/teamwerk}"                  # PII/Docs/Uploads/Protokolle
: "${VPS_ENVFILE:=/etc/teamwerk/env}"                 # Config mit Secrets
: "${VPS_VIDEO_RAW:=/storage/videos/raw}"             # Video-Rohdaten
: "${DAILY_KEEP:=7}"
: "${MONTHLY_KEEP:=6}"
: "${YEARLY_KEEP:=3}"
: "${VIDEO_MAX_AGE_DAYS:=7}"                          # Frische-Garantie

SSH_OPTS=(-i "$SSH_KEY" -o BatchMode=yes -o StrictHostKeyChecking=accept-new)
RSYNC_SSH="ssh ${SSH_OPTS[*]}"

TODAY=$(date +%Y-%m-%d)
DOM=$(date +%d)     # Tag im Monat, 01..31
DOY_MONTH=$(date +%m)

log() { printf '[%s] %s\n' "$(date -Iseconds)" "$*"; }
die() { log "FATAL: $*"; exit 1; }

# ---------------------------------------------------------------------------
# Vorbedingungen
# ---------------------------------------------------------------------------
command -v rsync   >/dev/null || die "rsync fehlt (apt install rsync)"
command -v ssh     >/dev/null || die "ssh fehlt"
command -v tar     >/dev/null || die "tar fehlt"
[[ -r "$SSH_KEY" ]] || die "SSH-Key nicht lesbar: $SSH_KEY"

mkdir -p "$BACKUP_ROOT"/{daily,monthly,yearly,videos-latest,tmp}

# ---------------------------------------------------------------------------
# 1. Konsistenten DB-Snapshot auf dem VPS erzeugen
# ---------------------------------------------------------------------------
# `sqlite3 .backup` erzeugt einen atomaren Snapshot auch bei aktivem WAL —
# einfaches `cp` würde inkonsistent kopieren (WAL nicht mit-committed).
# `-readonly` erzwingt Read-Only-Handle → tw-backup hat kein Write-Recht auf
# /var/lib/teamwerk (siehe Setup unten), das ist so gewollt.
REMOTE_SNAPSHOT="/tmp/teamwerk-snapshot-$$.db"
log "Erzeuge SQLite-Snapshot auf VPS: $REMOTE_SNAPSHOT"
ssh "${SSH_OPTS[@]}" "$VPS_SSH" \
    "sqlite3 -readonly '$VPS_DB' \".backup '$REMOTE_SNAPSHOT'\" && chmod 600 '$REMOTE_SNAPSHOT'" \
    || die "SQLite-Snapshot fehlgeschlagen"

# Snapshot am Ende (auch bei Fehler) vom VPS entfernen.
trap 'ssh "${SSH_OPTS[@]}" "$VPS_SSH" "rm -f \"$REMOTE_SNAPSHOT\"" || true' EXIT

# ---------------------------------------------------------------------------
# 2. DB + /var/lib/teamwerk + Config nach lokalem Staging holen
# ---------------------------------------------------------------------------
STAGE="$BACKUP_ROOT/tmp/stage-$TODAY"
rm -rf "$STAGE"
# Struktur = spätere Zielpfade → tar kann relativ packen, Restore-Extraktion
# mit `tar -xzf … -C /` legt die Files direkt an der richtigen Stelle ab.
mkdir -p "$STAGE/var/lib/teamwerk" "$STAGE/etc/teamwerk"

log "Rsync: DB-Snapshot → Stage"
rsync -e "$RSYNC_SSH" -a --info=stats1 \
    "$VPS_SSH:$REMOTE_SNAPSHOT" \
    "$STAGE/var/lib/teamwerk/teamwerk.db" \
    || die "Rsync DB fehlgeschlagen"

log "Rsync: /var/lib/teamwerk (ohne DB-Kopie) → Stage"
# --exclude verhindert Doppelung + inkonsistente Roh-DB
rsync -e "$RSYNC_SSH" -a --info=stats1 --delete \
    --exclude 'teamwerk.db' --exclude 'teamwerk.db-*' \
    "$VPS_SSH:$VPS_VARLIB/" "$STAGE/var/lib/teamwerk/" \
    || die "Rsync /var/lib/teamwerk fehlgeschlagen"

log "Rsync: /etc/teamwerk/env → Stage (Secrets)"
rsync -e "$RSYNC_SSH" -a --info=stats1 \
    "$VPS_SSH:$VPS_ENVFILE" "$STAGE/etc/teamwerk/env" \
    || die "Rsync Config fehlgeschlagen"
chmod 600 "$STAGE/etc/teamwerk/env"

# ---------------------------------------------------------------------------
# 3. Tar.gz-Archiv bauen (relative Pfade var/… etc/… → Restore mit `tar -C /`)
# ---------------------------------------------------------------------------
ARCHIVE="$BACKUP_ROOT/daily/teamwerk-$TODAY.tar.gz"
log "Packe Archiv: $ARCHIVE"
tar -czf "$ARCHIVE.tmp" -C "$STAGE" var etc \
    || die "Tar fehlgeschlagen"
mv "$ARCHIVE.tmp" "$ARCHIVE"
chmod 600 "$ARCHIVE"

# Checksum für Integritätsprüfung
sha256sum "$ARCHIVE" > "$ARCHIVE.sha256"

# Stage wieder abräumen
rm -rf "$STAGE"

log "Archiv fertig: $(du -h "$ARCHIVE" | cut -f1) — $ARCHIVE"

# ---------------------------------------------------------------------------
# 4. GFS-Retention: monthly (1. des Monats) und yearly (1.1.) verlinken
# ---------------------------------------------------------------------------
if [[ "$DOM" == "01" ]]; then
    MONTH_LINK="$BACKUP_ROOT/monthly/teamwerk-$(date +%Y-%m).tar.gz"
    log "Erzeuge Monthly-Snapshot: $MONTH_LINK"
    cp -al "$ARCHIVE" "$MONTH_LINK" 2>/dev/null || cp "$ARCHIVE" "$MONTH_LINK"
    cp "$ARCHIVE.sha256" "$MONTH_LINK.sha256"

    if [[ "$DOY_MONTH" == "01" ]]; then
        YEAR_LINK="$BACKUP_ROOT/yearly/teamwerk-$(date +%Y).tar.gz"
        log "Erzeuge Yearly-Snapshot: $YEAR_LINK"
        cp -al "$ARCHIVE" "$YEAR_LINK" 2>/dev/null || cp "$ARCHIVE" "$YEAR_LINK"
        cp "$ARCHIVE.sha256" "$YEAR_LINK.sha256"
    fi
fi

# ---------------------------------------------------------------------------
# 5. Pruning
# ---------------------------------------------------------------------------
prune_dir() {
    local dir="$1" keep="$2"
    # Sortiert nach Name absteigend (YYYY-MM-DD/YYYY-MM/YYYY sortieren
    # lexikographisch == chronologisch); die neuesten `keep` behalten,
    # den Rest inkl. sha256 löschen.
    (cd "$dir" && ls -1 teamwerk-*.tar.gz 2>/dev/null | sort -r | tail -n +$((keep + 1)) | while read -r f; do
        log "Prune $dir/$f"
        rm -f "$f" "$f.sha256"
    done) || true
}

prune_dir "$BACKUP_ROOT/daily"   "$DAILY_KEEP"
prune_dir "$BACKUP_ROOT/monthly" "$MONTHLY_KEEP"
prune_dir "$BACKUP_ROOT/yearly"  "$YEARLY_KEEP"

# ---------------------------------------------------------------------------
# 6. Videos (Raw) — nur ein Stand, überschreibend, mit Frische-Garantie
# ---------------------------------------------------------------------------
VIDEOS_DIR="$BACKUP_ROOT/videos-latest"
log "Rsync Videos (raw) → $VIDEOS_DIR"
if rsync -e "$RSYNC_SSH" -a --info=stats1 --delete \
        "$VPS_SSH:$VPS_VIDEO_RAW/" "$VIDEOS_DIR/"; then
    # Epoch-Sekunden direkt (portabler als ISO-String → date -d ist
    # nicht überall GNU, z.B. Mittwald bash 4.0 mit älterer coreutils).
    date +%s > "$VIDEOS_DIR/.last-sync"
    log "Videos: $(du -sh "$VIDEOS_DIR" | cut -f1)"
else
    log "WARN: Video-Rsync fehlgeschlagen — Frische-Check greift"
fi

# Frische-Garantie: älter als VIDEO_MAX_AGE_DAYS → leeren, damit kein
# stale-Backup Sicherheit vortäuscht.
if [[ -s "$VIDEOS_DIR/.last-sync" ]]; then
    LAST=$(cat "$VIDEOS_DIR/.last-sync")
    NOW=$(date +%s)
    if [[ "$LAST" =~ ^[0-9]+$ ]] && (( NOW > LAST )); then
        AGE_DAYS=$(( (NOW - LAST) / 86400 ))
        if (( AGE_DAYS > VIDEO_MAX_AGE_DAYS )); then
            log "WARN: Video-Backup $AGE_DAYS Tage alt (>$VIDEO_MAX_AGE_DAYS) — leere Verzeichnis"
            find "$VIDEOS_DIR" -mindepth 1 -delete
        fi
    fi
fi

log "Backup fertig. Bestand:"
log "  daily:   $(ls -1 "$BACKUP_ROOT/daily"   2>/dev/null | grep -c '\.tar\.gz$')"
log "  monthly: $(ls -1 "$BACKUP_ROOT/monthly" 2>/dev/null | grep -c '\.tar\.gz$')"
log "  yearly:  $(ls -1 "$BACKUP_ROOT/yearly"  2>/dev/null | grep -c '\.tar\.gz$')"
log "  videos:  $(du -sh "$VIDEOS_DIR" 2>/dev/null | cut -f1)"

# =============================================================================
# Restore-Cheatsheet
# =============================================================================
# Ausgangslage: teamwerk läuft ggf. schon; Restore auf frischen oder alten VPS.
#
#   # 1. Archiv auf VPS kopieren
#   scp teamwerk-YYYY-MM-DD.tar.gz  root@vps:/tmp/
#   scp teamwerk-YYYY-MM-DD.tar.gz.sha256  root@vps:/tmp/
#   ssh root@vps 'cd /tmp && sha256sum -c teamwerk-*.sha256'   # Integrität!
#
#   # 2. Auf VPS
#   systemctl stop teamwerk
#   # Sicherheitskopie des Ist-Zustands anlegen (falls Restore doch schiefgeht)
#   mv /var/lib/teamwerk /var/lib/teamwerk.rollback.$(date +%s)
#   mv /etc/teamwerk    /etc/teamwerk.rollback.$(date +%s)
#   tar -xzf /tmp/teamwerk-*.tar.gz -C /
#   chown -R www-data:www-data /var/lib/teamwerk
#   chmod 600 /etc/teamwerk/env
#   # Videos separat vom Mittwald-Host mit rsync zurückholen
#   #   rsync -av mittwald:teamwerk-backup/videos-latest/ /storage/videos/raw/
#   # Anschließend erneut transcodieren lassen (Admin-Reprocess-Endpoint).
#   systemctl start teamwerk
#
# =============================================================================
