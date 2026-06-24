#!/usr/bin/env bash
#
# deploy-encryption.sh — vollautomatischer Rollout der At-Rest-Verschlüsselung
# (OpenSpec-Change encrypt-bank-sepa-at-rest).
#
# Ablauf (idempotent, mehrfach ausführbar):
#   1. Preflight  — .env/REMOTE, SSH-Erreichbarkeit, sauberer Working-Tree
#   2. Schlüssel  — FIELD_ENCRYPTION_KEY auf dem VPS sicherstellen (nur erzeugen,
#                   wenn noch keiner gesetzt ist) — VOR dem Restart, sonst bootet
#                   der Service nicht (Startup-Check).
#   3. Deploy     — `make deploy` (Build + Binary + Service-Neustart). Ab hier
#                   verschlüsselt jeder NEUE Schreibvorgang.
#   4. Backup     — `make backup` (DB + Uploads lokal) VOR encrypt-pii.
#   5. encrypt-pii— Bestand (DB-Zeilen + vorhandene SEPA-PDFs) verschlüsseln,
#                   danach Dateieigentum auf www-data zurücksetzen + Restart.
#   6. Verifikation — Stichprobe zeigt "v1:"-Prefix.
#
# Verwendung:
#   bash deploy/deploy-encryption.sh          # interaktiv (fragt an Risikopunkten)
#   bash deploy/deploy-encryption.sh --yes     # ohne Rückfragen (CI/automatisiert)
#   make deploy-encrypted [YES=1]
#
set -euo pipefail

# In den Repo-Root wechseln (Skript liegt in deploy/).
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ASSUME_YES=0
for arg in "$@"; do
	case "$arg" in
	-y | --yes) ASSUME_YES=1 ;;
	*)
		echo "Unbekannte Option: $arg" >&2
		exit 2
		;;
	esac
done

# --- Konstanten (entsprechen Makefile / setup-vps.sh) ---
ENV_FILE=/etc/teamwerk/env
DB_PATH=/var/lib/teamwerk/teamwerk.db
DATA_DIR=/var/lib/teamwerk
BINARY=teamwerk

REMOTE="$(grep -E '^REMOTE=' .env 2>/dev/null | cut -d= -f2-)"
REMOTE_DIR="$(grep -E '^REMOTE_DIR=' .env 2>/dev/null | cut -d= -f2-)"
REMOTE_DIR="${REMOTE_DIR:-/usr/local/bin}"

log() { printf '\n\033[1m▸ %s\033[0m\n' "$*"; }
warn() { printf '\033[33m! %s\033[0m\n' "$*"; }
die() {
	printf '\033[31m✗ %s\033[0m\n' "$*" >&2
	exit 1
}

confirm() {
	[ "$ASSUME_YES" = 1 ] && return 0
	local reply
	read -r -p "$1 [y/N] " reply
	[[ "$reply" =~ ^[yYjJ]$ ]] || die "Abgebrochen."
}

# --- 1. Preflight ---------------------------------------------------------
log "1/6 Preflight"
[ -n "$REMOTE" ] || die "REMOTE ist nicht in .env gesetzt (z.B. REMOTE=vServer)."
command -v ssh >/dev/null || die "ssh nicht gefunden."
ssh -o BatchMode=yes -o ConnectTimeout=10 "$REMOTE" true 2>/dev/null ||
	die "SSH zu '$REMOTE' nicht möglich."
ssh "$REMOTE" "sudo test -f $ENV_FILE" ||
	die "$ENV_FILE existiert nicht — zuerst 'make setup-vps' ausführen."

echo "  Remote:  $REMOTE  (Binary: $REMOTE_DIR/$BINARY)"
echo "  Deploy:  $(git rev-parse --abbrev-ref HEAD) @ $(git rev-parse --short HEAD)"
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
	warn "Working-Tree hat uncommittete Änderungen — es wird der HEAD-Stand gebaut."
fi
confirm "Mit diesem Stand deployen?"

# --- 2. Schlüssel sicherstellen (VOR dem Restart) -------------------------
log "2/6 FIELD_ENCRYPTION_KEY sicherstellen"
if ssh "$REMOTE" "sudo grep -qE '^FIELD_ENCRYPTION_KEY=.+' $ENV_FILE"; then
	echo "  Schlüssel bereits gesetzt — Generierung übersprungen (idempotent)."
else
	warn "Kein FIELD_ENCRYPTION_KEY gesetzt — erzeuge einen neuen 32-Byte-Schlüssel."
	# Auf dem VPS erzeugen + anhängen, damit der Klartext nicht durch lokale
	# Shell-History/Prozessliste läuft; einmalig zur Sicherung ausgeben.
	NEW_KEY="$(ssh "$REMOTE" "sudo sh -c '
		K=\$(openssl rand -base64 32);
		printf \"\nFIELD_ENCRYPTION_KEY=%s\n\" \"\$K\" >> $ENV_FILE;
		chmod 600 $ENV_FILE;
		printf %s \"\$K\"
	'")"
	[ -n "$NEW_KEY" ] || die "Schlüsselerzeugung fehlgeschlagen."
	echo "=============================================================="
	echo " NEUER FIELD_ENCRYPTION_KEY — JETZT SEPARAT SICHERN:"
	echo
	echo "     $NEW_KEY"
	echo
	echo " Getrennt vom DB-Backup aufbewahren (Passwort-Manager)."
	echo " SCHLÜSSELVERLUST = DATENVERLUST der Bank-/SEPA-Felder."
	echo "=============================================================="
	confirm "Schlüssel an sicherem Ort gespeichert?"
fi

# --- 3. Deploy (Binary + Restart) -----------------------------------------
log "3/6 Deploy (make deploy)"
echo "  Ab dem Neustart verschlüsselt jeder neue Schreibvorgang automatisch."
make deploy

# --- 4. Backup VOR encrypt-pii --------------------------------------------
log "4/6 Backup (DB + Uploads) vor der Erstverschlüsselung"
make backup

# --- 5. Erstverschlüsselung des Bestands ----------------------------------
log "5/6 Bestand verschlüsseln (encrypt-pii)"
echo "  Verschlüsselt bestehende DB-Zeilen + vorhandene SEPA-PDFs. Idempotent."
confirm "encrypt-pii jetzt ausführen?"
# Als root (liest das 0600-Env), danach Dateieigentum zurück auf www-data:
# encrypt-pii schreibt PDFs via atomic rename und ggf. -wal/-shm neu.
ssh "$REMOTE" "sudo sh -c '
	set -a; . $ENV_FILE; set +a;
	$REMOTE_DIR/$BINARY encrypt-pii
'"
ssh "$REMOTE" "sudo chown -R www-data:www-data $DATA_DIR"
ssh "$REMOTE" "sudo systemctl restart $BINARY"

# --- 6. Verifikation ------------------------------------------------------
log "6/6 Verifikation"
echo "  Stichprobe members.iban (erwartet 'v1:'-Prefix):"
ssh "$REMOTE" "sudo -u www-data sqlite3 $DB_PATH \
	\"SELECT substr(iban,1,3) AS prefix, COUNT(*) FROM members WHERE iban IS NOT NULL GROUP BY 1;\"" ||
	warn "Stichprobe nicht möglich (sqlite3 fehlt?) — manuell prüfen."

log "Fertig."
echo "  Neue Schreibvorgänge + Bestand sind verschlüsselt."
echo "  Rollback bei Bedarf: 'ssh $REMOTE \"sudo sh -c \\\"set -a; . $ENV_FILE; set +a; $REMOTE_DIR/$BINARY decrypt-pii\\\"\"'"
