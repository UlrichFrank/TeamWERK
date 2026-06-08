## Why

Nach dem Speichern einer Abwesenheit (Urlaub/Verletzung) werden Training-Responses im Backend korrekt auf `declined` gesetzt — aber die KalenderPage lädt die Trainings danach nicht neu. Der Nutzer sieht deshalb keine visuelle Rückmeldung über die Auto-Absagen, und andere geöffnete Clients werden nie benachrichtigt.

## What Changes

- `doSaveAbsence()` in KalenderPage ruft nach dem POST auch `loadTrainings()` auf
- `useLiveUpdates` in KalenderPage reagiert neu auf `"trainings"`-Events, damit alle Clients synchronisiert werden
- Der Preview-Endpoint (`GET /api/absences/preview`) gibt zusätzlich zu bestätigten auch alle Training-Sessions im Zeitraum zurück, bei denen der Member Kader-Mitglied ist — unabhängig vom bisherigen Antwortstatus

## Capabilities

### New Capabilities

*(keine neuen Capabilities — nur Bugfixes und UX-Korrekturen)*

### Modified Capabilities

- `absences`: Preview-Endpoint zeigt jetzt alle betroffenen Training-Sessions, nicht nur bereits bestätigte

## Impact

- `web/src/pages/KalenderPage.tsx`: `doSaveAbsence()`, `useLiveUpdates`-Handler
- `internal/absences/handler.go`: `Preview`-Handler, Training-Query erweitert
