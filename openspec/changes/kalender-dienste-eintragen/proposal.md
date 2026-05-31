## Why

Nutzer müssen derzeit zur separaten Seite `/dienste` wechseln, um sich für Dienste einzutragen — auch wenn sie gerade einen konkreten Kalender-Eintrag anschauen. Das erzeugt unnötige Kontextwechsel und verschlechtert die Usability, besonders auf Mobile.

## What Changes

- `GET /duty-board` erhält einen optionalen Query-Parameter `?game_id=<id>`, der die Ergebnisse auf die Slots eines einzelnen Spiels filtert.
- Eine neue gemeinsame Frontend-Komponente `DutySlotList` wird aus `DutyPage` extrahiert und kapselt die vollständige Slot-Interaktion (Eintragen/Austragen, Zuteilungen aufklappen, Erfüllen/Geldersatz, Löschen).
- `DutyPage` wird auf `DutySlotList` umgestellt (Verhalten unverändert).
- `SpieltagDetailPage` lädt zusätzlich `GET /duty-board?game_id={id}` und zeigt die Slots mit `DutySlotList` — inklusive Eintragen/Austragen für alle Nutzer.
- Die bisherige ProgressBar-Ansicht in `SpieltagDetailPage` entfällt; das Board-UI übernimmt die Slot-Darstellung vollständig.
- Add/Edit-Modals für Slots bleiben unverändert in `SpieltagDetailPage`.

## Capabilities

### New Capabilities

- `duty-board-game-filter`: Backend-Filter `?game_id=` auf dem `/duty-board`-Endpunkt — liefert `claimed_by_me` und `vacancies` für einen einzelnen Spieltag.
- `kalender-dienste-panel`: Dienst-Eintragen direkt auf der Kalender-Detailseite via gemeinsamer `DutySlotList`-Komponente.

### Modified Capabilities

<!-- keine bestehenden Spec-Anforderungen ändern sich -->

## Impact

- **Backend:** `internal/duties/handler.go` — `Board()`-Funktion, WHERE-Clause-Builder
- **Frontend neu:** `web/src/components/DutySlotList.tsx`
- **Frontend geändert:** `web/src/pages/DutyPage.tsx`, `web/src/pages/SpieltagDetailPage.tsx`
- **API:** `GET /duty-board` erhält neuen optionalen Query-Parameter (abwärtskompatibel)
- **Keine neuen Abhängigkeiten, keine DB-Migration**
