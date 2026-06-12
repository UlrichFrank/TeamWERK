## 1. Makefile — CHANGELOG.md-Generierung

- [x] 1.1 `changes.json`-Generierungszeile in `make build` durch `CHANGELOG.md`-Generierung ersetzen: `git log --format="%ad|%s" --date=format:"%d.%m.%Y" --no-merges` → nur `feat`/`fix`-Commits → nach Datum gruppiert → `web/public/CHANGELOG.md`
- [x] 1.2 `web/public/changes.json` löschen (Datei selbst, nicht nur Build-Schritt)

## 2. useVersionCheck — updateDescription entfernen

- [x] 2.1 `changes.json`-Fetch und `updateDescription`-State aus `useVersionCheck.ts` entfernen
- [x] 2.2 Rückgabe-Typ auf `{ version: string | null, updateAvailable: boolean }` reduzieren

## 3. ChangelogModal — neue Komponente

- [x] 3.1 `web/src/components/ChangelogModal.tsx` anlegen: `fetch('/CHANGELOG.md')` beim Mount, Mini-Parser (Regex), Render als Datum-Gruppen mit `[feat]`/`[fix]`-Badges und Scope
- [x] 3.2 Ladeindikator und Fehlerfall implementieren
- [x] 3.3 Schließen per ✕-Button und Escape-Taste (`useEscapeKey`)

## 4. AppShell — Button + Modal verdrahten

- [x] 4.1 Versions-`<span>` in `<button>` umwandeln, `onClick` → `setShowChangelog(true)`
- [x] 4.2 `showChangelog`-State anlegen, `ChangelogModal` einbinden
- [x] 4.3 Update-Banner: `updateDescription`/`showUpdateDetails` entfernen; „Details"-Button → `setShowChangelog(true)`
- [x] 4.4 `updateDescription` aus `useVersionCheck`-Destructuring entfernen
