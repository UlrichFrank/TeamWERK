#!/usr/bin/env bash
# =============================================================================
# TeamWERK — serverseitiges tägliches Backup (läuft AUF dem VPS selbst)
# =============================================================================
#
# Ergänzt (ersetzt nicht) den externen Pull-Weg vom Mittwald-Host
# (deploy/backup-teamwerk.sh, `make backup`/`make backup-files`): dieses
# Skript sichert DB + Storage-Pfade lokal auf dem VPS nach
# /var/backups/teamwerk/<datum>/, mit 14 Tagen Retention (design.md
# Decision 6, specs/vps-deployment/spec.md "Tägliches Backup"). Es ist kein
# Ersatz für ein Offsite-Backup — beide Wege bleiben aktiv.
#
# Installiert von deploy/setup-vps.sh und bei jedem `make deploy` nach
# /usr/local/bin/teamwerk-backup.sh; Cron-Eintrag 30 3 * * * (Env-Zeitzone
# des VPS, siehe dort).
#
# Gesichert wird:
#   - SQLite-DB (`sqlite3 .backup`, konsistenter Snapshot auch bei aktivem WAL)
#   - Storage-Pfade aus /etc/teamwerk/env (UPLOAD_DIR, FILES_DIR, MEDIA_DIR,
#     BEITRAGSLAUF_DIR, TRAINING_DIARY_DIR, MATCH_REPORT_IMAGE_DIR) als
#     ein gemeinsames tar.gz
#   - NICHT: Videos (VIDEO_STORAGE_DIR) — GB-Bereich, eigener Weg via
#     `make backup-videos`
#
# Restore (Kurzform):
#   systemctl stop teamwerk
#   cp /var/backups/teamwerk/<datum>/teamwerk.db /var/lib/teamwerk/teamwerk.db
#   rm -f /var/lib/teamwerk/teamwerk.db-wal /var/lib/teamwerk/teamwerk.db-shm
#   tar -xzf /var/backups/teamwerk/<datum>/storage.tar.gz -C /
#   chown -R www-data:www-data /var/lib/teamwerk
#   systemctl start teamwerk
# =============================================================================

set -euo pipefail

ENV_FILE=/etc/teamwerk/env
BACKUP_ROOT=/var/backups/teamwerk
RETENTION_DAYS=14

log() { printf '[%s] %s\n' "$(date -Iseconds)" "$*"; }
die() { log "FATAL: $*"; exit 1; }

EXIT_CODE=0
trap 'EXIT_CODE=$?; log "Ende (exit code: $EXIT_CODE)"' EXIT

[[ -r "$ENV_FILE" ]] || die "Env-Datei nicht lesbar: $ENV_FILE"
command -v sqlite3 >/dev/null 2>&1 || die "sqlite3 fehlt (siehe deploy/setup-vps.sh, apt-get install)"
command -v tar >/dev/null 2>&1 || die "tar fehlt"

# shellcheck disable=SC1090
set -a
. "$ENV_FILE"
set +a

: "${DB_PATH:?DB_PATH fehlt in $ENV_FILE}"
[[ -f "$DB_PATH" ]] || die "DB nicht gefunden: $DB_PATH"

DATE="$(date +%Y-%m-%d)"
DEST="$BACKUP_ROOT/$DATE"
mkdir -p "$DEST"

# ---------------------------------------------------------------------------
# 1. Konsistenter DB-Snapshot (`sqlite3 .backup` — WAL-safe, anders als `cp`)
# ---------------------------------------------------------------------------
log "Sichere DB: $DB_PATH -> $DEST/teamwerk.db"
sqlite3 "$DB_PATH" ".backup '$DEST/teamwerk.db.tmp'" || die "sqlite3 .backup fehlgeschlagen"
mv "$DEST/teamwerk.db.tmp" "$DEST/teamwerk.db"

# ---------------------------------------------------------------------------
# 2. Storage-Pfade als ein gemeinsames tar.gz (Videos bewusst ausgenommen)
# ---------------------------------------------------------------------------
STORAGE_DIRS=()
for var in UPLOAD_DIR FILES_DIR MEDIA_DIR BEITRAGSLAUF_DIR TRAINING_DIARY_DIR MATCH_REPORT_IMAGE_DIR; do
    val="${!var:-}"
    if [[ -n "$val" && -d "$val" ]]; then
        STORAGE_DIRS+=("$val")
    fi
done

if [[ ${#STORAGE_DIRS[@]} -gt 0 ]]; then
    log "Packe Storage-Pfade (${#STORAGE_DIRS[@]}): ${STORAGE_DIRS[*]}"
    tar -czf "$DEST/storage.tar.gz.tmp" "${STORAGE_DIRS[@]}" || die "Storage-Tar fehlgeschlagen"
    mv "$DEST/storage.tar.gz.tmp" "$DEST/storage.tar.gz"
else
    log "WARN: keine Storage-Pfade in $ENV_FILE gefunden oder angelegt — storage.tar.gz übersprungen"
fi

chmod 700 "$DEST"

# ---------------------------------------------------------------------------
# 3. Retention: Verzeichnisse älter als RETENTION_DAYS Tage löschen
# ---------------------------------------------------------------------------
log "Retention: entferne Backups älter als $RETENTION_DAYS Tage unter $BACKUP_ROOT"
find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d -mtime +"$RETENTION_DAYS" -exec rm -rf {} +

log "Backup fertig: $DEST ($(du -sh "$DEST" 2>/dev/null | cut -f1))"
