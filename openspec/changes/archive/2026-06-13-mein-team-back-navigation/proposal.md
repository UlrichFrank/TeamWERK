## Why

Die MeinTeamPage (`/mein-team?team=X`) zeigt keinen Zurück-Button, obwohl User vom Dashboard dorthin navigieren. Konsistente Navigation erfordert einen Rückweg — wie ihn z.B. TermineDetailPage bereits bietet.

## What Changes

- `MeinTeamPage` erhält einen Zurück-Button (← navigiert per `navigate(-1)`), der **nur erscheint** wenn `?team=X` in der URL steht (`focusTeamId != null`)
- Kein Zurück-Button wenn User direkt über die Sidebar zu `/mein-team` navigiert

## Capabilities

### New Capabilities

- `mein-team-back-button`: Zurück-Navigation auf MeinTeamPage bei gefilterter Team-Ansicht

### Modified Capabilities

_(keine bestehenden Specs betroffen)_

## Impact

- **Datei:** `web/src/pages/MeinTeamPage.tsx`
- Keine Backend-Änderungen
- Keine neuen Dependencies
