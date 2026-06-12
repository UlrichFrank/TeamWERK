## Context

Die App hat bereits eine Versionsinfrastruktur: `buildHash` wird per ldflags eingebettet, per SSE an den Client gesendet, und `useVersionCheck` erkennt Versionsunterschiede. Bisher gab es zusätzlich `changes.json` (Delta-Beschreibung seit letztem Deploy) für den Update-Banner. Diese Datei wird abgelöst.

## Goals / Non-Goals

**Goals:**
- Vollständige, dauerhafte Versionshistorie im Frontend zugänglich machen
- Kein neues npm-Package
- `changes.json` konsolidieren (ein Format, ein File)

**Non-Goals:**
- Manuell pflegbares Changelog — es wird immer aus git log generiert
- Semantic Versioning / SemVer-Nummern
- Backend-API-Endpoint für Changelog-Daten

## Decisions

### CHANGELOG.md als statisches File statt JSON

`CHANGELOG.md` wird bei `make build` generiert und als statische Datei in `web/public/` abgelegt. Das Frontend fetcht es per `fetch('/CHANGELOG.md')`. Begründung: Markdown ist menschenlesbar und direkt im Repo sichtbar; da das Format eng definiert ist, braucht es keinen vollständigen MD-Parser.

### Eigener Mini-Parser statt react-markdown

Das Changelog-Format ist vollständig vorhersehbar:
```
## DD.MM.YYYY
- [feat] scope: Beschreibung
- [fix] scope: Beschreibung
```
Ein Parser von ~25 Zeilen Regex reicht aus. Keine neue Dependency (react-markdown ~17kB gzip) für ein einziges Modal.

### updateDescription entfällt, Modal übernimmt

`useVersionCheck` liefert zukünftig nur `{ version, updateAvailable }`. Der Update-Banner zeigt bei Klick auf „Details" das `ChangelogModal`. Dadurch gibt es eine einzige kanonische Ansicht für Änderungen.

### CHANGELOG.md-Format

```markdown
## 09.06.2026

- [feat] games: Mehrtägige Events mit end_date für Turniere
- [fix] duties: Sonstige Dienste nach Datum gruppiert

## 08.06.2026

- [feat] members: CSV-Import mit Adresse und IBAN-Validierung
```

Generierungsbefehl in Makefile (Python-Snippet für JSON-sichere Datums-Aufbereitung):
```bash
git log --format="%ad|%s" --date=format:"%d.%m.%Y" --no-merges \
  | grep -E "\|(feat|fix)(\([^)]*\))?:" \
  | awk -F'|' '{...}' → gruppiert nach Datum → CHANGELOG.md
```

### Mini-Parser-Logik (Frontend)

```
Input: rohes CHANGELOG.md als String
Output: Array<{ date: string, entries: Array<{ type: 'feat'|'fix', scope: string, message: string }> }>

Zeile "## DD.MM.YYYY" → neues Datum-Group
Zeile "- [feat] scope: text" → Entry mit type=feat, scope, message
Zeile "- [fix] scope: text" → Entry mit type=fix, scope, message
Leerzeilen und andere Zeilen → ignorieren
```

## Risks / Trade-offs

**git log im Makefile** → Funktioniert nur wenn `.git/` vorhanden ist. In der Deploy-Pipeline (`make build` auf lokalem Rechner) ist das gegeben. Der VPS selbst baut nicht. Kein Risiko.

**CHANGELOG.md wächst mit der Zeit** → Unkritisch, reine Textdatei, nach Jahren noch < 50kB.

**Altes `changes.json` verschwindet** → `useVersionCheck` muss angepasst werden; kein anderer Code referenziert `updateDescription` außer AppShell.
