## Why

Bei Diensten wie „Kuchen" (Bewirtungsrotation) müssen sich eingetragene Personen
untereinander abstimmen, was sie mitbringen (z. B. „Marmorkuchen", „laktosefrei") —
dafür gibt es aktuell keinen Platz. Das bestehende `duty_slots.role_desc` ist
ungeeignet: es wird vom Ersteller/Vorstand einmalig am Slot gesetzt, nicht von der
Person, die sich einträgt, und ist pro Slot statt pro Zuteilung — ein Slot hat oft
mehrere `slots_total`-Plätze mit unterschiedlichen Personen und unterschiedlichem
Beitrag.

## What Changes

- Neue Tabelle `duty_assignment_comments`: genau ein Freitext-Kommentar pro
  `duty_assignments`-Zeile, geschrieben von der eingetragenen Person selbst
  (bzw. deren Elternteil bei einem Proxy-Kind).
- Neue Endpunkte `PUT`/`DELETE /api/duty-assignments/{id}/comment` (eigenen
  Kommentar setzen/löschen) und `GET /api/duty-slots/{id}/comments` (alle
  Kommentare eines Slots lesen, für jeden mit Board-Zugriff).
- Duty-Board-Response bekommt pro Slot ein `comment_count`-Feld (schlank, kein
  Volltext im Listing).
- `DutySlotList.tsx`: neues Kommentar-Icon neben dem bestehenden Anleitungs-Icon,
  sichtbar nur bei vorhandenen Kommentaren, öffnet ein Modal mit allen
  Kommentaren des Slots.
- Neues ⋮-Aktionsmenü **auch auf Desktop** (bisher nur mobile) mit den Einträgen
  Bearbeiten, Anleitung und neu Kommentieren (nur sichtbar, wenn man selbst
  eingetragen ist); Eintragen/Austragen bleiben eigene, direkt sichtbare Buttons.

## Capabilities

### New Capabilities
- `duty-assignment-comments`: Freitext-Kommentar pro Dienst-Zuteilung — Schreib-/
  Löschrecht nur für die eingetragene Person (kein Admin-Bypass), universelles
  Leserecht, automatisches Aufräumen beim Wegfall der Zuteilung, Board-Anzeige
  über ein Zähl-Feld plus On-Demand-Detailabruf.

### Modified Capabilities
(keine — das bestehende `duty-assignee-display`/`duty-type-instructions` bleiben
unverändert; die Kommentar-Anzeige ist rein additiv)

## Impact

- **Backend:** `internal/db/migrations/062_duty_assignment_comments.{up,down}.sql`,
  `internal/duties/handler.go` (neue Routen + `comment_count` im Board-Query),
  `internal/app/router.go` (Routen-Eintrag).
- **Frontend:** `web/src/components/DutySlotList.tsx` (Icon, Modal, ⋮-Menü auch
  Desktop), `BoardSlot`-Interface um `comment_count` erweitert.
- **Keine Breaking Changes** — rein additiv, bestehende Board-Konsumenten
  (`SpieltagDetailModal` u. a.) ignorieren das neue Feld unverändert.
