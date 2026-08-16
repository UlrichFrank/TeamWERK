#!/usr/bin/env bash
# next-version.sh — leitet die nächste Semver-Version aus Conventional Commits ab.
#
# Sucht das jüngste vX.Y.Z-Tag und klassifiziert alle Commits seither:
#   BREAKING CHANGE (Body) oder `<type>!:` (Subject) → major
#   feat                                              → minor
#   fix | perf                                        → patch
#   sonst                                             → kein Bump
#
# Ausgabe (stdout): die nächste Version als `vX.Y.Z`, oder das bestehende
# Tag falls kein Bump anfällt.
#
# Flags:
#   --check       Exit 0 wenn ein Bump anfällt, sonst Exit 1 (keine stdout-Ausgabe).
#   --min LEVEL   Untergrenze für den Bump (patch|minor|major). Ergibt die
#                 Commit-Analyse weniger (inkl. „kein Bump"), wird LEVEL benutzt;
#                 ergibt sie mehr, gewinnt die Analyse. Damit erzeugt ein
#                 manueller Release-Lauf auch dann ein Tag, wenn die Commits seit
#                 dem letzten Tag nicht conventional formatiert sind (typisch bei
#                 Squash-Merges mit Branch-Namen als Titel).
#
# Ohne vorheriges Tag startet die Versionierung bei v0.1.0 (sofern überhaupt
# Commits in den Range fallen).

set -euo pipefail

CHECK_ONLY=0
MIN_BUMP=none
while [ $# -gt 0 ]; do
  case "$1" in
    --check) CHECK_ONLY=1; shift ;;
    --min)
      MIN_BUMP="${2:-}"
      case "$MIN_BUMP" in
        patch|minor|major) ;;
        *) echo "usage: $0 [--check] [--min patch|minor|major]" >&2; exit 2 ;;
      esac
      shift 2
      ;;
    *) echo "usage: $0 [--check] [--min patch|minor|major]" >&2; exit 2 ;;
  esac
done

# Rangfolge für den Vergleich „Analyse vs. --min".
bump_rank() {
  case "$1" in
    major) echo 3 ;;
    minor) echo 2 ;;
    patch) echo 1 ;;
    *)     echo 0 ;;
  esac
}

LAST_TAG="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1 || true)"

if [ -z "$LAST_TAG" ]; then
  MAJOR=0; MINOR=0; PATCH=0
  RANGE="HEAD"
  HAS_BASE=0
else
  IFS='.' read -r MAJOR MINOR PATCH <<<"${LAST_TAG#v}"
  RANGE="${LAST_TAG}..HEAD"
  HAS_BASE=1
fi

SUBJECTS="$(git log "$RANGE" --no-merges --format='%s' || true)"
BODIES="$(git log "$RANGE" --no-merges --format='%b' || true)"

BUMP=none
if printf '%s\n' "$BODIES" | grep -qE '^BREAKING[ -]CHANGE:'; then
  BUMP=major
elif printf '%s\n' "$SUBJECTS" | grep -qE '^[a-z]+(\([^)]+\))?!:'; then
  BUMP=major
elif printf '%s\n' "$SUBJECTS" | grep -qE '^feat(\([^)]+\))?:'; then
  BUMP=minor
elif printf '%s\n' "$SUBJECTS" | grep -qE '^(fix|perf)(\([^)]+\))?:'; then
  BUMP=patch
fi

# --min hebt einen zu schwachen Analyse-Bump an, senkt einen stärkeren aber nie
# ab: ein `feat!:` bleibt major, auch wenn nur `--min patch` verlangt wurde.
if [ "$(bump_rank "$MIN_BUMP")" -gt "$(bump_rank "$BUMP")" ]; then
  BUMP="$MIN_BUMP"
fi

if [ "$BUMP" = "none" ]; then
  if [ "$HAS_BASE" = "0" ] && [ -n "$SUBJECTS" ]; then
    # Erst-Release: irgendwas ist da, aber kein feat/fix → 0.1.0
    BUMP=minor
  else
    [ "$CHECK_ONLY" = "1" ] && exit 1
    [ -n "$LAST_TAG" ] && echo "$LAST_TAG" || echo "v0.0.0"
    exit 0
  fi
fi

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

[ "$CHECK_ONLY" = "1" ] && exit 0
echo "v${MAJOR}.${MINOR}.${PATCH}"
