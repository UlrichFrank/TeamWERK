#!/usr/bin/env bash
# Erzeugt die Screenshots der Schulungsfolien neu (Aufruf über `make schulung`).
#
#   1. konsistente Kopie der lokalen DB  → docs/schulung/.build/demo.db
#   2. anonymisieren (tools/anon.py)     → Personen ersetzt, Personas vorstand@/trainer@beispiel.de
#   3. Frontend + Binary bauen, Server auf der Kopie starten (leere Medienordner, kein Mail, kein Push)
#   4. Screenshots laut shots.txt        → docs/schulung/folien/img/*.jpg
#
# Optionen (Umgebung):
#   SRC_DB=<pfad>          Quell-DB (Default ./teamwerk.db, z. B. aus `make pull-db`)
#   SCHULUNG_PORT=18090    Port des Demo-Servers
#   SCHULUNG_VORSTAND=10   Nutzer-ID der Vorstand-Persona
#   SCHULUNG_TRAINER=11    Nutzer-ID der Trainer-Persona
#   SKIP_WEB_BUILD=1       vorhandenes Frontend-Build nutzen
#   KEEP=1                 Server nach den Screenshots laufen lassen (zum Durchklicken, Strg+C beendet)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
HERE="$ROOT/docs/schulung"
BUILD="$HERE/.build"
SRC_DB="${SRC_DB:-$ROOT/teamwerk.db}"
PORT="${SCHULUNG_PORT:-18090}"
GO="${GO:-go}"

case "$(cd "$(dirname "$SRC_DB")" 2>/dev/null && pwd)" in
  /var/lib/teamwerk*) echo "schulung: verweigert — SRC_DB zeigt auf das Prod-Verzeichnis" >&2; exit 1 ;;
esac
[ -f "$SRC_DB" ] || { echo "schulung: Quell-DB fehlt: $SRC_DB (erst 'make pull-db' oder SRC_DB=… setzen)" >&2; exit 1; }
for tool in sqlite3 python3 node pnpm; do
  command -v "$tool" >/dev/null || { echo "schulung: '$tool' nicht gefunden" >&2; exit 1; }
done

echo "==> 1/4 DB kopieren und anonymisieren ($SRC_DB)"
rm -rf "$BUILD"
mkdir -p "$BUILD"/store/{uploads,files,media,beitrag,diary,videos,mri} "$HERE/folien/img"
sqlite3 -readonly "$SRC_DB" ".backup '$BUILD/demo.db'"
python3 "$HERE/tools/anon.py" "$BUILD/demo.db" "$BUILD/ids.json"

echo "==> 2/4 Frontend und Binary bauen"
if [ "${SKIP_WEB_BUILD:-}" != "1" ]; then
  pnpm -C "$ROOT/web" build >/dev/null
fi
(cd "$ROOT" && "$GO" build -o "$BUILD/teamwerk" ./cmd/teamwerk)

echo "==> 3/4 Demo-Server starten (http://127.0.0.1:$PORT)"
# Start aus $BUILD heraus: dort liegt keine .env, die DB_PATH & Co. überschreiben könnte.
# AUTH_RATE_LIMIT_PER_MIN=0: jeder Seitenaufruf der SPA löst /api/auth/refresh aus; mit der
# Prod-Drosselung (10/min und IP) scheitert sonst nach gut 20 Screenshots der nächste Login mit 429.
(
  cd "$BUILD"
  exec env PORT="$PORT" DB_PATH="$BUILD/demo.db" BASE_URL="http://127.0.0.1:$PORT" \
    JWT_SECRET="schulung-demo-secret-mindestens-32-bytes" LOG_FORMAT=text MAILER_DISABLED=true \
    AUTH_RATE_LIMIT_PER_MIN=0 \
    VAPID_PUBLIC_KEY= VAPID_PRIVATE_KEY= TYPO3_IMPORT_URL= \
    UPLOAD_DIR="$BUILD/store/uploads" FILES_DIR="$BUILD/store/files" MEDIA_DIR="$BUILD/store/media" \
    BEITRAGSLAUF_DIR="$BUILD/store/beitrag" TRAINING_DIARY_DIR="$BUILD/store/diary" \
    VIDEO_STORAGE_DIR="$BUILD/store/videos" MATCH_REPORT_IMAGE_DIR="$BUILD/store/mri" \
    ./teamwerk
) > "$BUILD/server.log" 2>&1 &
SERVER_PID=$!
trap 'kill $SERVER_PID 2>/dev/null || true' EXIT
for _ in $(seq 1 60); do
  curl -fsS "http://127.0.0.1:$PORT/api/healthz" >/dev/null 2>&1 && break
  kill -0 $SERVER_PID 2>/dev/null || { echo "schulung: Server beendet — siehe $BUILD/server.log" >&2; exit 1; }
  sleep 1
done
curl -fsS "http://127.0.0.1:$PORT/api/healthz" >/dev/null || { echo "schulung: Server antwortet nicht — siehe $BUILD/server.log" >&2; exit 1; }

echo "==> 4/4 Screenshots laut shots.txt"
node "$HERE/tools/shots.mjs" "$BUILD/ids.json" "$HERE/shots.txt" "$HERE/folien/img" "http://127.0.0.1:$PORT"

echo "Fertig: $HERE/folien/index.html im Browser öffnen. Vor dem Teilen die Bilder einmal ansehen."
if [ "${KEEP:-}" = "1" ]; then
  echo "Server läuft weiter auf http://127.0.0.1:$PORT — vorstand@beispiel.de / trainer@beispiel.de, Passwort Schulung2026! (Strg+C beendet)"
  wait $SERVER_PID
fi
