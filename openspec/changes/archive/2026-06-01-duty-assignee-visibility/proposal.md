## Why

Nutzer sehen auf der Dienstbörse und auf Event-Detail-Seiten, welche Slots frei oder besetzt sind — aber nicht, wer sich eingetragen hat. Das macht es schwierig, spontan Absprachen zu treffen oder zu prüfen, ob man mit bekannten Leuten zusammenarbeitet.

## What Changes

- Die Dienstbörse (`/dienste`) zeigt unter jedem Slot die Namen der eingetragenen Personen
- Die Event-Detail-Seite (`/kalender/:id`) zeigt dieselben Assignee-Informationen
- Wenn eine Person `photo_visible` freigegeben hat, wird ihr Profilbild als Avatar angezeigt
- Per Hover (Desktop) oder Tap (Mobile) öffnet ein Tooltip mit Telefonnummer(n) (wenn `phones_visible`) und Adresse (wenn `address_visible`)
- Die Sichtbarkeitsregeln werden serverseitig angewendet — der Client bekommt nur Daten, die der Nutzer freigegeben hat
- Alle eingeloggten Nutzer, die einen Slot sehen können, sehen auch dessen Assignees

## Capabilities

### New Capabilities
- `duty-assignee-display`: Anzeige der eingetragenen Personen pro Dienst-Slot mit privacy-gefiltertem Tooltip (Foto, Telefon, Adresse)

### Modified Capabilities
- `duties`: Duty-Board-Response wird um `assignees[]` pro Slot erweitert; neue Anforderung: Assignee-Namen und freigegeben Kontaktdaten sind für alle Auth-Nutzer sichtbar

## Impact

- `internal/duties/handler.go`: `GetBoard`-Handler — JOIN auf `duty_assignments` → `users` → `user_visibility` → `user_phones`; `boardSlot`-Struct um `Assignees []PublicAssignee` erweitern
- `web/src/components/DutySlotList.tsx`: `BoardSlot`-Interface + Sub-Row-Rendering + `AssigneeTooltip`-Komponente
- Keine neuen DB-Tabellen, keine neuen Routen, keine neuen Abhängigkeiten
