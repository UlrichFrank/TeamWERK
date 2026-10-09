#!/usr/bin/env bash
# Spiegelt die Prod-Daten auf die Beta-Instanz — läuft AUF DEM VPS (Aufruf über
# `make mirror-beta FORCE=1`). Nichts verlässt den Server.
#
# Gespiegelt:  SQLite-DB, Dokumente (files), Beitragslauf-Protokolle, BWHV-Spielberichte.
# Nicht:       Videos und alle Bild-Ablagen (uploads = Profilfotos/SEPA-Mandate,
#              media = Chat-Bilder, training-diary = Nachweise, match-report-images).
#              In der Beta fehlen diese Dateien bewusst; sie zeigt dort leere Bilder.
#
# Prod wird AUSSCHLIESSLICH GELESEN:
#   - alle Lese- und Kopierschritte laufen als www-data (nie root), damit im
#     Prod-Verzeichnis keine root-eigene Datei entstehen kann
#   - die DB wird über die SQLite-Online-Backup-API aus einer -readonly-Verbindung
#     kopiert (dasselbe Verfahren wie das tägliche Backup), nicht per cp
#   - jedes Schreibziel wird vor der Nutzung gegen das Beta-Präfix geprüft
#   - nice/ionice: die Kopie weicht jeder Prod-Last aus
#   - gestoppt/gestartet wird nur teamwerk-beta
# Nachbearbeitet wird ausschließlich die Kopie (siehe sanitize unten).
set -euo pipefail

readonly PROD_DATA=/var/lib/teamwerk
readonly PROD_DB=$PROD_DATA/teamwerk.db
readonly BETA_DATA=/var/lib/teamwerk-beta
readonly BETA_DB=$BETA_DATA/teamwerk.db
readonly BETA_BIN=/usr/local/bin/teamwerk-beta
readonly MIRROR_DIRS=(files beitragslauf-protokolle bwhv-reports)

as_www() { runuser -u www-data -- nice -n 19 ionice -c3 "$@"; }

beta_only() {
  # Schreibziel muss unter dem Beta-Präfix liegen und darf nicht ins Prod-Verzeichnis zeigen.
  local p; p="$(realpath -m "$1")"
  case "$p" in
    "$BETA_DATA"|"$BETA_DATA"/*) ;;
    *) echo "beta-mirror: verweigert — Schreibziel $p liegt nicht unter $BETA_DATA" >&2; exit 1 ;;
  esac
  case "$p" in
    "$PROD_DATA"|"$PROD_DATA"/*) echo "beta-mirror: verweigert — $p ist ein Prod-Pfad" >&2; exit 1 ;;
  esac
}

[ "$(id -u)" = 0 ] || { echo "beta-mirror: als root (sudo) aufrufen" >&2; exit 1; }
[ -x "$BETA_BIN" ] || { echo "beta-mirror: $BETA_BIN fehlt — erst 'make deploy-beta'" >&2; exit 1; }
[ -f /etc/teamwerk-beta/env ] || { echo "beta-mirror: Beta nicht eingerichtet — erst 'make setup-beta'" >&2; exit 1; }
grep -q '^DB_PATH='"$BETA_DB"'$' /etc/teamwerk-beta/env \
  || { echo "beta-mirror: DB_PATH in /etc/teamwerk-beta/env zeigt nicht auf $BETA_DB" >&2; exit 1; }

TMP_DB=$BETA_DATA/teamwerk.db.mirror
beta_only "$TMP_DB"; beta_only "$BETA_DB"
rm -f "$TMP_DB" "$TMP_DB-wal" "$TMP_DB-shm"

echo "==> 1/5 Prod-DB lesen (Online-Backup, readonly, als www-data)"
as_www sqlite3 -readonly "$PROD_DB" ".backup '$TMP_DB'"
[ "$(as_www sqlite3 "$TMP_DB" 'PRAGMA integrity_check;')" = ok ] \
  || { echo "beta-mirror: Kopie nicht konsistent — Abbruch, Beta unverändert" >&2; rm -f "$TMP_DB"; exit 1; }

echo "==> 2/5 Kopie für die Beta bereinigen"
# Push-Abos: ohne VAPID sendet die Beta ohnehin nicht — gelöscht, damit das auch
# dann gilt, wenn jemand später Schlüssel einträgt. Refresh-Tokens: Sitzungen
# beginnen auf der Beta neu (die Cookies sind ohnehin host-gebunden).
# Wartungsmodus: eine gerade gesperrte Prod soll die Beta nicht mitsperren.
as_www sqlite3 "$TMP_DB" <<'SQL'
DELETE FROM push_subscriptions;
DELETE FROM refresh_tokens;
UPDATE system_settings SET value='off' WHERE key='maintenance_mode';
PRAGMA wal_checkpoint(TRUNCATE);
SQL

echo "==> 3/5 Beta stoppen, DB einsetzen, Migrationen"
systemctl stop teamwerk-beta
rm -f "$BETA_DB" "$BETA_DB-wal" "$BETA_DB-shm"
mv "$TMP_DB" "$BETA_DB"
rm -f "$TMP_DB-wal" "$TMP_DB-shm"
chown www-data:www-data "$BETA_DB"
as_www "$BETA_BIN" migrate up --db "$BETA_DB"

echo "==> 4/5 Dateiablagen spiegeln: ${MIRROR_DIRS[*]}"
for d in "${MIRROR_DIRS[@]}"; do
  beta_only "$BETA_DATA/$d"
  mkdir -p "$BETA_DATA/$d"; chown www-data:www-data "$BETA_DATA/$d"
  # Quelle mit Slash, Ziel fest unter BETA_DATA; --delete wirkt nur auf das Ziel.
  as_www rsync -a --delete "$PROD_DATA/$d/" "$BETA_DATA/$d/"
done
# Nicht gespiegelte Ablagen leeren: Reste eines früheren Seeds gehörten zu anderen IDs.
for d in uploads media training-diary match-report-images; do
  beta_only "$BETA_DATA/$d"
  find "$BETA_DATA/$d" -mindepth 1 -delete 2>/dev/null || true
done

echo "==> 5/5 Beta starten"
systemctl start teamwerk-beta
for _ in $(seq 1 15); do
  if curl -fsS http://127.0.0.1:8081/api/healthz >/dev/null 2>&1; then
    echo "beta-mirror: fertig — $(as_www sqlite3 -readonly "$BETA_DB" 'SELECT COUNT(*) FROM users') Nutzer, $(as_www sqlite3 -readonly "$BETA_DB" 'SELECT COUNT(*) FROM games') Spiele"
    exit 0
  fi
  sleep 2
done
systemctl status teamwerk-beta --no-pager | tail -20 >&2
echo "beta-mirror: Beta antwortet nicht" >&2
exit 1
