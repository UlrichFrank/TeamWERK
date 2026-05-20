## Why

Der Kader-Kopier-Workflow enthält einen logischen Bug: Die Richtung der Altersklass-Progression ist invertiert, und es fehlt ein Jahrgangsfilter beim Kopieren — dadurch landen falsche Spieler im falschen Kader. Zusätzlich ist Auto-Assign konzeptuell ein anderer Anwendungsfall als das Kopieren und gehört nicht in denselben Workflow.

## What Changes

- **BUGFIX**: `ageClassBefore` Richtung korrigieren: A←B, B←C, C←D (bisher invertiert: B←A, C←B, D←C)
- **BUGFIX**: `copyMembersFromKader` mit Jahrgangsfilter erweitern — nur Mitglieder übernehmen, deren Geburtsjahr im Bracket der Ziel-Jugend liegt
- **Vereinfachung**: Copy-Modal auf einen einzigen smarten Kopiervorgang reduzieren (Option `auto-assign` entfernen, Optionen `same-age-previous` und `age-before-previous` zusammenführen)
- **Neue Aktion**: Separater „Auto-Assign"-Button neben „Aus vorheriger Saison kopieren" auf der Kader-Seite, mit eigenem Modal zur Kader-Auswahl

## Capabilities

### New Capabilities

- `kader-auto-assign-modal`: Eigenständiges Modal zum Auto-Assign von Mitgliedern per Jahrgang+Geschlecht, mit Checkbox-Auswahl welche Kader befüllt werden sollen

### Modified Capabilities

- `kader-copy-from-season`: Korrekter Smart-Copy-Algorithmus mit Jahrgangsfilter und richtiger Richtung; `auto-assign`-Option entfernt

## Impact

- `internal/kader/copy.go`: `ageClassBefore` + `copyMembersFromKader` + `copyKader`
- `web/src/components/CopyKaderModal.tsx`: `auto-assign` Option entfernen, Logik vereinfachen
- `web/src/pages/AdminKaderPage.tsx`: Zweiten Action-Button hinzufügen
- Neue Komponente: `web/src/components/AutoAssignModal.tsx`
- Keine DB-Schema-Änderungen, keine API-Änderungen
