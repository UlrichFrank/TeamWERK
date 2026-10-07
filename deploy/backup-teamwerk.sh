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
#   - Fertige Videos (/storage/videos/processed, HLS) — separat, nur EIN aktueller Stand
#
# Retention
#   Kleine Daten (DB + PII/Docs/Uploads/Protokolle + Config):
#     daily/    → 7 Stück (letzte 7 Tage)
#     monthly/  → 6 Stück (jeweils Backup vom 1. eines Monats)
#     yearly/   → 3 Stück (jeweils Backup vom 1. Januar)
#   Videos (fertig umgepackte HLS-Renditions, ~GB-Bereich):
#     videos-latest/ → 1 rsync-Spiegel, wird bei jedem Lauf nachgezogen
#                      (--delete: ein auf dem VPS gelöschtes Video verschwindet
#                      beim nächsten Lauf auch hier). Rohdateien gibt es nicht
#                      mehr — der Worker löscht sie nach dem Umpacken.
#                      Ist der letzte erfolgreiche Sync älter als 7 Tage, gibt
#                      es eine WARN-Zeile; der Bestand bleibt liegen (er ist
#                      die einzige Kopie, Leeren hieße Datenverlust).
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
#   4. Ziel-Verzeichnis auf Mittwald: mkdir -p /backup
#   5. Cronjob auf Mittwald (mStudio-Cronjob-UI, Tab "Benutzerdefinierter Aufruf"):
#      Dateipfad /backup/backup-teamwerk.sh, Interpreter /bin/bash, Parameter leer,
#      Zeitplan z.B. 02:00 täglich.
#      HINWEIS: Die mStudio-UI hat kein Feld für Env-Variablen — BACKUP_ROOT steht
#      deshalb unten als fester Default auf /backup (nicht mehr per Env-Var
#      injizierbar); wer's lokal woanders hinlegen will, exportiert BACKUP_ROOT
#      trotzdem vor dem Aufruf, das Skript respektiert einen gesetzten Wert weiter.
#      HINWEIS: Mittwald-Home /backup ist noexec — Skript daher explizit via
#      `bash /pfad/...` starten (NICHT direkt ausführen).
#   6. Better-Stack-Heartbeat-URLs hinterlegen (Monitoring → Heartbeats):
#        mkdir -p ~/.config/teamwerk-backup && chmod 700 ~/.config/teamwerk-backup
#        echo '<URL TeamWERK Backup (Mittwald)>'         > ~/.config/teamwerk-backup/heartbeat-backup
#        echo '<URL TeamWERK Backup-Speicher (Mittwald)>' > ~/.config/teamwerk-backup/heartbeat-storage
#        chmod 600 ~/.config/teamwerk-backup/heartbeat-*
#   7. Nach dem ersten Lauf: Restore einmal auf Test-VPS durchspielen.
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
: "${BACKUP_ROOT:=/backup}"                           # Lokale Backup-Wurzel (mStudio-Cron kann kein Env setzen)
: "${VPS_DB:=/var/lib/teamwerk/teamwerk.db}"          # Pfad DB auf VPS
: "${VPS_VARLIB:=/var/lib/teamwerk}"                  # PII/Docs/Uploads/Protokolle
: "${VPS_ENVFILE:=/etc/teamwerk/env}"                 # Config mit Secrets
: "${VPS_VIDEO_DIR:=/storage/videos/processed}"       # fertige Videos (HLS)
: "${DAILY_KEEP:=7}"
: "${MONTHLY_KEEP:=6}"
: "${YEARLY_KEEP:=3}"
: "${VIDEO_MAX_AGE_DAYS:=7}"                          # Frische-Warnung
: "${QUOTA_PATH:=$HOME}"                              # zählt gegen das Projekt-Kontingent
: "${QUOTA_LIMIT_GB:=50}"                             # mStudio-Tarif des Projekts p459264
: "${QUOTA_WARN_PCT:=85}"                             # ab hier schlägt der Speicher-Heartbeat fehl
# Better-Stack-Heartbeats (URLs sind Zugangsdaten → nicht im Repo, Datei 600):
#   heartbeat-backup  → „TeamWERK Backup (Mittwald)":  Ping nur bei vollständigem Lauf
#   heartbeat-storage → „TeamWERK Backup-Speicher (Mittwald)": /fail ab QUOTA_WARN_PCT
: "${HEARTBEAT_DIR:=$HOME/.config/teamwerk-backup}"

SSH_OPTS=(-i "$SSH_KEY" -o BatchMode=yes -o StrictHostKeyChecking=accept-new)
RSYNC_SSH="ssh ${SSH_OPTS[*]}"

TODAY=$(date +%Y-%m-%d)
DOM=$(date +%d)     # Tag im Monat, 01..31
DOY_MONTH=$(date +%m)

log() { printf '[%s] %s\n' "$(date -Iseconds)" "$*"; }

# heartbeat <name> [fail <meldung>] — Ping an Better Stack. Fehlt die
# URL-Datei, wird nur geloggt; ein Monitoring-Fehler bricht nie das Backup ab.
heartbeat() {
    local file="$HEARTBEAT_DIR/heartbeat-$1" url
    [[ -s "$file" ]] || { log "WARN: $file fehlt — kein Heartbeat"; return 0; }
    url=$(<"$file")
    if [[ "${2:-}" == "fail" ]]; then
        curl -fsS -m 30 --retry 3 -X POST --data-raw "${3:-}" "$url/fail" >/dev/null \
            || log "WARN: Heartbeat $1/fail nicht zustellbar"
    else
        curl -fsS -m 30 --retry 3 "$url" >/dev/null \
            || log "WARN: Heartbeat $1 nicht zustellbar"
    fi
}

die() { log "FATAL: $*"; heartbeat backup fail "FATAL: $*"; exit 1; }
# Alles, was set -e außerhalb eines `|| die` abbricht, meldet sich ebenfalls
# sofort — nicht erst, wenn nach 27 h der ausbleibende Ping auffällt.
trap 'die "Unerwarteter Abbruch in Zeile $LINENO"' ERR

# ---------------------------------------------------------------------------
# Vorbedingungen
# ---------------------------------------------------------------------------
command -v rsync   >/dev/null || die "rsync fehlt (apt install rsync)"
command -v ssh     >/dev/null || die "ssh fehlt"
command -v tar     >/dev/null || die "tar fehlt"
command -v curl    >/dev/null || die "curl fehlt"
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
# Auch Stages abgebrochener Vortage räumen: ein FATAL beendet das Skript vor
# dem Aufräumen, und jede Leiche hält eine volle Kopie von /var/lib/teamwerk.
rm -rf "$BACKUP_ROOT"/tmp/stage-*
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

# ls statt du: der Mittwald-Pool ist ZFS, du meldet dort die komprimierte und
# direkt nach dem Schreiben noch unvollständig verbuchte Belegung (63M statt 188M).
log "Archiv fertig: $(ls -lh "$ARCHIVE" | awk '{print $5}') — $ARCHIVE"

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
# 6. Videos (fertige HLS-Renditions) — ein Spiegel, mit Frische-Warnung
# ---------------------------------------------------------------------------
# Nicht ins tar: die Segmente sind bereits komprimiert, und 7 Tagesarchive à
# Video-Bestand sprengten den Speicher. .last-sync wird vom --delete
# verschont (--exclude), sonst löschte der Spiegel ihn bei jedem Lauf.
VIDEOS_DIR="$BACKUP_ROOT/videos-latest"
VIDEO_OK=1
log "Rsync Videos ($VPS_VIDEO_DIR) → $VIDEOS_DIR"
if rsync -e "$RSYNC_SSH" -a --info=stats1 --delete --exclude '.last-sync' \
        "$VPS_SSH:$VPS_VIDEO_DIR/" "$VIDEOS_DIR/"; then
    # Epoch-Sekunden direkt (portabler als ISO-String → date -d ist
    # nicht überall GNU, z.B. Mittwald bash 4.0 mit älterer coreutils).
    date +%s > "$VIDEOS_DIR/.last-sync"
    log "Videos: $(du -sh "$VIDEOS_DIR" | cut -f1)"
else
    VIDEO_OK=0
    log "WARN: Video-Rsync fehlgeschlagen — Frische-Check greift"
fi

# Frische-Warnung: älter als VIDEO_MAX_AGE_DAYS → laut melden, aber NICHT
# leeren — der Spiegel ist die einzige Kopie der Videos außerhalb des VPS.
if [[ -s "$VIDEOS_DIR/.last-sync" ]]; then
    LAST=$(cat "$VIDEOS_DIR/.last-sync")
    NOW=$(date +%s)
    if [[ "$LAST" =~ ^[0-9]+$ ]] && (( NOW > LAST )); then
        AGE_DAYS=$(( (NOW - LAST) / 86400 ))
        if (( AGE_DAYS > VIDEO_MAX_AGE_DAYS )); then
            log "WARN: Video-Backup $AGE_DAYS Tage alt (>$VIDEO_MAX_AGE_DAYS) — Sync prüfen!"
        fi
    fi
fi

log "Backup fertig. Bestand:"
log "  daily:   $(ls -1 "$BACKUP_ROOT/daily"   2>/dev/null | grep -c '\.tar\.gz$')"
log "  monthly: $(ls -1 "$BACKUP_ROOT/monthly" 2>/dev/null | grep -c '\.tar\.gz$')"
log "  yearly:  $(ls -1 "$BACKUP_ROOT/yearly"  2>/dev/null | grep -c '\.tar\.gz$')"
log "  videos:  $(du -sh "$VIDEOS_DIR" 2>/dev/null | cut -f1)"

# ---------------------------------------------------------------------------
# 7. Speicherplatz gegen das Projekt-Kontingent
# ---------------------------------------------------------------------------
# Der Mittwald-Pool (ZFS) meldet per statvfs nur den geteilten Pool, nicht das
# Tarif-Kontingent — deshalb zählt du die Belegung des Projekts selbst.
USED_KB=$(du -sk "$QUOTA_PATH" 2>/dev/null | cut -f1) || true
if [[ "$USED_KB" =~ ^[0-9]+$ ]]; then
    LIMIT_KB=$(( QUOTA_LIMIT_GB * 1024 * 1024 ))
    USED_PCT=$(( USED_KB * 100 / LIMIT_KB ))
    USED_GB=$(( USED_KB / 1024 / 1024 ))
    log "Speicher: ${USED_GB} GB von ${QUOTA_LIMIT_GB} GB belegt (${USED_PCT} %)"
    if (( USED_PCT >= QUOTA_WARN_PCT )); then
        heartbeat storage fail "Mittwald-Projekt: ${USED_GB} GB von ${QUOTA_LIMIT_GB} GB belegt (${USED_PCT} % >= ${QUOTA_WARN_PCT} %)"
    else
        heartbeat storage
    fi
else
    log "WARN: Belegung von $QUOTA_PATH nicht ermittelbar"
    heartbeat storage fail "Belegung von $QUOTA_PATH nicht ermittelbar"
fi

# Erst ganz am Ende: Archiv UND Video-Spiegel müssen geklappt haben.
if (( VIDEO_OK )); then
    heartbeat backup
else
    heartbeat backup fail "Archiv ok, aber Video-Rsync fehlgeschlagen"
fi

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
#   # Videos separat vom Mittwald-Host mit rsync zurückholen (fertige HLS,
#   # kein erneutes Umpacken nötig — DB-Zeilen zeigen auf processed/{id})
#   #   rsync -av --exclude .last-sync mittwald:/backup/videos-latest/ /storage/videos/processed/
#   #   chown -R www-data:www-data /storage/videos/processed
#   systemctl start teamwerk
#
# =============================================================================
