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
#   --check   Exit 0 wenn ein Bump anfällt, sonst Exit 1 (keine stdout-Ausgabe).
#
# Ohne vorheriges Tag startet die Versionierung bei v0.1.0 (sofern überhaupt
# Commits in den Range fallen).

set -euo pipefail

CHECK_ONLY=0
case "${1:-}" in
  --check) CHECK_ONLY=1 ;;
  "" ) ;;
  *) echo "usage: $0 [--check]" >&2; exit 2 ;;
esac

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
