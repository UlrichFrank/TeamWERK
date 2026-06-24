#!/usr/bin/env bash
# Legt system-dashboard.json via Better-Stack-Telemetry-API an.
#
# Voraussetzungen:
#   - jq, curl
#   - Telemetry-API-Token (NICHT der Vector-Ingestion-Token aus vector.toml!):
#       Better Stack → Team settings → API tokens → "Telemetry API token"
#   - Beim ersten Lauf in Better Stack einmalig prüfen, dass die Metrik-Source
#     existiert und Daten enthält. Nach dem Anlegen im Dashboard manuell die
#     korrekte Source auswählen (Dropdown oben rechts), sonst greift {{source}}
#     ins Leere.
#
# Verwendung:
#   export BETTERSTACK_API_TOKEN=...
#   ./apply.sh                # legt Dashboard + 6 Charts an
#   ./apply.sh --dry-run      # zeigt nur die Payloads
set -euo pipefail

API="https://telemetry.betterstack.com/api/v2"
SPEC="$(dirname "$0")/system-dashboard.json"
DRY_RUN="${1:-}"

[[ -f "$SPEC" ]] || { echo "FEHLT: $SPEC" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq fehlt" >&2; exit 1; }

dashboard_payload="$(jq '.dashboard' "$SPEC")"

if [[ "$DRY_RUN" == "--dry-run" ]]; then
  echo "=== Dashboard ==="
  echo "$dashboard_payload"
  echo "=== Charts ==="
  jq '.charts[]' "$SPEC"
  exit 0
fi

[[ -n "${BETTERSTACK_API_TOKEN:-}" ]] || { echo "Setze BETTERSTACK_API_TOKEN" >&2; exit 1; }

echo "→ Lege Dashboard an…"
dashboard_resp="$(curl -fsS -X POST "$API/dashboards" \
  -H "Authorization: Bearer $BETTERSTACK_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data "$dashboard_payload")"

dashboard_id="$(echo "$dashboard_resp" | jq -r '.data.id')"
[[ -n "$dashboard_id" && "$dashboard_id" != "null" ]] || {
  echo "Dashboard-Anlage fehlgeschlagen:" >&2
  echo "$dashboard_resp" >&2
  exit 1
}
echo "  Dashboard-ID: $dashboard_id"

chart_count="$(jq '.charts | length' "$SPEC")"
for i in $(seq 0 $((chart_count - 1))); do
  chart_payload="$(jq ".charts[$i]" "$SPEC")"
  chart_name="$(echo "$chart_payload" | jq -r '.name')"
  echo "→ Chart $((i+1))/$chart_count: $chart_name"
  curl -fsS -X POST "$API/dashboards/$dashboard_id/charts" \
    -H "Authorization: Bearer $BETTERSTACK_API_TOKEN" \
    -H "Content-Type: application/json" \
    --data "$chart_payload" > /dev/null
done

echo "✓ Fertig — Dashboard #$dashboard_id mit $chart_count Charts angelegt."
echo "  Im Better-Stack-UI öffnen und Source-Dropdown auf die Metric-Source der Vector-Sink stellen."
