#!/bin/sh
# Claude-Code PostToolUse-Hook: formatiert eine gerade editierte/geschriebene
# Go-Datei automatisch mit gofmt. Hält den Selbstkorrektur-Loop geschlossen,
# damit der pre-commit-Hook nicht an Formatierung scheitert.
#
# Eingabe: Hook-JSON auf stdin (enthält tool_input.file_path).
# Verdrahtet in .claude/settings.json unter hooks.PostToolUse (Matcher Edit|Write).

f="$(python3 -c 'import sys,json; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' 2>/dev/null)"

case "$f" in
	*.go) ;;
	*) exit 0 ;;
esac
[ -f "$f" ] || exit 0

GOFMT=/usr/local/go/bin/gofmt
[ -x "$GOFMT" ] || GOFMT=gofmt
"$GOFMT" -w "$f" 2>/dev/null || true
exit 0
