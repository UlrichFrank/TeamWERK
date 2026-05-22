## Why

Die Dienstbörse zeigt heute eine flache Liste aller offenen Slots ohne Bezug zu Spielen oder Teams. Ein Spieler oder Elternteil sieht keinen Kontext: welches Spiel, welche Mannschaft, wann. Das macht es schwer zu entscheiden, wofür man sich einträgt. Außerdem fehlt die Möglichkeit, sich wieder auszutragen.

## What Changes

- `GET /api/duty-board` gibt Dienste gruppiert nach Spiel zurück, gefiltert auf die Teams des eingeloggten Users (eigene Mitgliedschaft + Kinder via `family_links`)
- Jede Gruppe enthält ein `claimed_by_me`-Flag pro Slot
- Neuer Endpunkt `DELETE /api/duty-board/{slotId}/claim` zum Austragen
- Slots ohne Spielbezug (`game_id NULL`) erscheinen als „Sonstige Dienste"-Gruppe pro Team
- Frontend: Kachel pro Spiel mit eingebetteter Tabelle; vergangene Spieltage standardmäßig ausgeblendet, per Knopf einblendbar

## Capabilities

### New Capabilities

- `duty-board-game-view`: Spieltagsgruppierte Dienstbörse mit Team-Filter, Claim/Unclaim, und Vergangenheitssteuerung

### Modified Capabilities

## Impact

- Backend: `internal/duties/handler.go` — `Board`-Handler komplett überarbeitet, neuer `Unclaim`-Handler
- Backend: `cmd/teamwerk/main.go` — neue DELETE-Route
- Frontend: `web/src/pages/DutyBoardPage.tsx` — vollständiges Redesign
- Keine DB-Schema-Änderung
