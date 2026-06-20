#!/usr/bin/env bash
#
# gen-icons.sh — erzeugt die generierten PWA-Icon-PNGs aus den Quell-SVGs.
#
# Voraussetzung: rsvg-convert (librsvg).  macOS: `brew install librsvg`.
#
# Erzeugt:
#   web/public/icons/icon-maskable-512.png  — Android maskable Icon (weißer Grund,
#                                             Logo innerhalb der 80 %-Safe-Zone)
#   web/public/icons/badge-96.png           — Android Notification-Badge (transparent,
#                                             monochromfähige Silhouette)
#
# Die erzeugten PNGs werden eingecheckt; dieses Skript dient der Reproduzierbarkeit.
set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v rsvg-convert >/dev/null 2>&1; then
  echo "Fehler: rsvg-convert nicht gefunden. Installieren mit: brew install librsvg" >&2
  exit 1
fi

SRC="web/icon-src"
OUT="web/public/icons"

# Maskable: weißer Hintergrund füllt die transparenten Ecken (robust auf jeder Maske).
rsvg-convert -w 512 -h 512 -b '#FFFFFF' "$SRC/IconAndroid.svg" -o "$OUT/icon-maskable-512.png"

# Badge: transparenter Hintergrund — Android rendert nur den Alpha-Kanal monochrom.
rsvg-convert -w 96 -h 96 "$SRC/Handball.svg" -o "$OUT/badge-96.png"

echo "Erzeugt:"
echo "  $OUT/icon-maskable-512.png"
echo "  $OUT/badge-96.png"
