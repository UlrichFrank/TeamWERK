#!/usr/bin/env bash
# Lädt den statischen ffmpeg-Build für eine Zielplattform nach
# internal/ffmpegbin/bin/ (wird dort per go:embed eingebettet) und prüft ihn
# gegen gepinnte SHA-256-Summen. Genutzt vom Release-Workflow und lokal für
# einen release-artigen Build.
#
#   scripts/fetch-ffmpeg.sh <darwin-arm64|darwin-x64|win32-x64>
#
# Quelle: github.com/eugeneware/ffmpeg-static (GPL-Builds inkl. libx264). Beim
# Wechsel auf ein neues Release Tag UND alle Summen anpassen:
#   gh release view <tag> -R eugeneware/ffmpeg-static --json assets \
#     -q '.assets[] | [.name, .digest] | @tsv'
set -euo pipefail

TAG="b6.1.1"
BASE="https://github.com/eugeneware/ffmpeg-static/releases/download/$TAG"

platform="${1:-}"
case "$platform" in
  darwin-arm64)
    sums=("ffmpeg-darwin-arm64.gz 8923876afa8db5585022d7860ec7e589af192f441c56793971276d450ed3bbfa"
          "darwin-arm64.LICENSE cb48bf09a11f5fb576cddb0431c8f5ed0a60157a9ec942adffc13907cbe083f2"
          "darwin-arm64.README 05ba4b92c96605434b1aaae3eedf5a2c280c9607bf78ffca9a5b536d9af2dc6a") ;;
  darwin-x64)
    sums=("ffmpeg-darwin-x64.gz 929b375c1182d956c51f7ac25e0b2b0411fb01f6f407aa15c9758efeb4242106"
          "darwin-x64.LICENSE 2e1d16c72fd74e12063776371da757322f8b77589386532f4fd8634bde7de1af"
          "darwin-x64.README e88a0325f8e5b75210355e37341824f074d3cd82def2125be54c914b62848a36") ;;
  win32-x64)
    sums=("ffmpeg-win32-x64.gz 8883a3dffbd0a16cf4ef95206ea05283f78908dbfb118f73c83f4951dcc06d77"
          "win32-x64.LICENSE 8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903"
          "win32-x64.README a636a7183c58006351acbaf35303c0ed85c6e1320fd4e80de453ba6157de6311") ;;
  *)
    echo "Aufruf: $0 <darwin-arm64|darwin-x64|win32-x64>" >&2
    exit 2 ;;
esac

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

dir="$(cd "$(dirname "$0")/.." && pwd)/internal/ffmpegbin/bin"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

for entry in "${sums[@]}"; do
  name="${entry% *}"
  want="${entry#* }"
  curl -fsSL --retry 3 -o "$tmp/$name" "$BASE/$name"
  got="$(sha256 "$tmp/$name")"
  if [ "$got" != "$want" ]; then
    echo "Prüfsumme falsch für $name: $got (erwartet $want)" >&2
    exit 1
  fi
done

rm -f "$dir"/ffmpeg.*
mv "$tmp/ffmpeg-$platform.gz" "$dir/ffmpeg.gz"
mv "$tmp/$platform.LICENSE"   "$dir/ffmpeg.LICENSE"
mv "$tmp/$platform.README"    "$dir/ffmpeg.README"
cat > "$dir/ffmpeg.SOURCE" <<EOF
Herkunft der gebündelten ffmpeg-Binary:
$BASE/ffmpeg-$platform.gz
SHA-256 (gzip): ${sums[0]#* }
EOF
echo "ffmpeg ($platform, $TAG) nach $dir gelegt."
