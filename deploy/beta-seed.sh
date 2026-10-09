#!/usr/bin/env bash
# Erzeugt LOKAL die Start-DB der Beta-Instanz (beta.teamwerk.team-stuttgart.org).
#
#   1. konsistente Kopie einer lokalen DB (Default ./teamwerk.db, z. B. aus `make pull-db`)
#   2. anonymisieren mit docs/schulung/tools/anon.py (Namen, Adressen, Texte, Tokens)
#   3. für eine öffentlich erreichbare Instanz zusätzlich härten:
#      - Bank-/SEPA-Ciphertexte und den Tresor-Schlüssel entfernen (die Ciphertexte
#        sind echt — wer die Vereins-Passphrase kennt, könnte sie sonst entschlüsseln)
#      - alle Login-Passwörter auf ein ZUFÄLLIGES Beta-Passwort setzen (anon.py setzt
#        ein im Repo dokumentiertes Schulungspasswort)
#      - alle übrigen Tokens (Kalender, Einladungen, Passwort-Reset) löschen
#
# Aufruf: bash deploy/beta-seed.sh <ziel.db>
# Ausgabe: das Beta-Passwort auf stdout (letzte Zeile). Die Prod-DB wird nie gelesen
# oder geschrieben; das Skript verweigert Pfade unter /var/lib/teamwerk.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC_DB="${SRC_DB:-$ROOT/teamwerk.db}"
OUT="${1:?Ziel-DB angeben}"

case "$(cd "$(dirname "$SRC_DB")" 2>/dev/null && pwd)" in
  /var/lib/teamwerk*) echo "beta-seed: verweigert — SRC_DB zeigt auf ein Server-Verzeichnis" >&2; exit 1 ;;
esac
[ -f "$SRC_DB" ] || { echo "beta-seed: Quell-DB fehlt: $SRC_DB (erst 'make pull-db')" >&2; exit 1; }
for tool in sqlite3 python3 htpasswd openssl; do
  command -v "$tool" >/dev/null || { echo "beta-seed: '$tool' nicht gefunden" >&2; exit 1; }
done

rm -f "$OUT" "$OUT-wal" "$OUT-shm"
sqlite3 -readonly "$SRC_DB" ".backup '$OUT'"
python3 "$ROOT/docs/schulung/tools/anon.py" "$OUT" "$(mktemp)" >&2

PW="$(openssl rand -base64 18 | tr -d '/+=' | cut -c1-16)"
# htpasswd erzeugt $2y$, Go-bcrypt liest $2a$ — gleicher Algorithmus, anderes Präfix.
HASH="$(htpasswd -bnBC 10 "" "$PW" | tr -d ':\n' | sed 's/^\$2y\$/$2a$/')"

sqlite3 "$OUT" <<SQL
PRAGMA foreign_keys=ON;
DELETE FROM member_sensitive;
UPDATE members SET sepa_mandat_dek_enc=NULL, sepa_mandat_path=NULL;
UPDATE clubs SET sepa_ciphertext=NULL, sepa_dek_enc=NULL,
  group_public_key=NULL, group_private_key_enc=NULL, vorstand_kdf_salt=NULL, vorstand_key_check=NULL;
DELETE FROM calendar_tokens;
DELETE FROM password_reset_tokens;
DELETE FROM invitation_tokens;
UPDATE users SET password='$HASH', failed_login_count=0, locked_until=NULL WHERE password IS NOT NULL AND password <> '';
PRAGMA wal_checkpoint(TRUNCATE);
SQL
sqlite3 "$OUT" "PRAGMA journal_mode=DELETE;" >/dev/null

echo "beta-seed: $OUT fertig (Personas vorstand@beispiel.de, trainer@beispiel.de)" >&2
echo "$PW"
