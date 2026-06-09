## Why

Die Versionsnummer in der Sidebar ist ein toter Text — Nutzer sehen zwar welche Version läuft, aber nicht was sich seit dem letzten Update geändert hat. Gleichzeitig zeigt der Update-Banner nur einen Freitext-Snippet der aktuellen Delta-Änderungen, ohne Kontext über frühere Versionen. Eine zugängliche Versionshistorie erhöht das Vertrauen in Updates und gibt Nutzern Orientierung.

## What Changes

- `make build` generiert `CHANGELOG.md` aus der vollständigen `git log`-Historie (nur `feat`/`fix` Conventional Commits), gruppiert nach Commit-Datum, in `web/public/`
- Die Versionsnummer im Sidebar-Footer (`v abc1234`) wird von einem `<span>` zu einem klickbaren `<button>`
- Klick auf den Versions-Button öffnet ein `ChangelogModal` das `CHANGELOG.md` fetcht und mit einem eigenen Mini-Parser (Regex, keine neue Dependency) als Datum-Gruppen mit `[feat]`/`[fix]`-Badges und Scope rendert
- Der Update-Banner öffnet bei Klick auf „Details" dasselbe Modal statt dem bisherigen Inline-Text
- `changes.json` und `updateDescription` in `useVersionCheck` entfallen; der Hook liefert nur noch `version` und `updateAvailable`

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `version-display`: Versions-Span wird Button; neues `ChangelogModal`
- `deploy-version-detection`: `changes.json`-Generierung entfällt; `useVersionCheck` verliert `updateDescription`; Makefile generiert stattdessen `CHANGELOG.md`

## Impact

- `Makefile` — `changes.json`-Erzeugung durch `CHANGELOG.md`-Generierung ersetzen (git log, alle feat/fix-Commits mit Datum)
- `web/public/CHANGELOG.md` — neue generierte statische Datei
- `web/src/hooks/useVersionCheck.ts` — `updateDescription` / `changes.json`-Fetch entfernen
- `web/src/components/AppShell.tsx` — Versions-Span → Button, Update-Banner Details → Modal öffnen
- `web/src/components/ChangelogModal.tsx` — neue Komponente (fetch + Mini-Parser + Render)
