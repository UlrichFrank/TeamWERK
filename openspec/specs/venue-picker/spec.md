### Requirement: Venue aus Liste wählen
Der VenuePicker SHALL in allen Event-Formularen (Spiel, Training-Series, Training-Session) einen auswählbaren Venue ermöglichen.

#### Scenario: Venue suchen und auswählen
- **WHEN** Nutzer tippt in das Picker-Feld
- **THEN** Liste der passenden Venues wird clientseitig gefiltert (Name, Stadt); Auswahl setzt venue_id im Formular

#### Scenario: Kein Venue ausgewählt
- **WHEN** Nutzer lässt Ort-Feld leer
- **THEN** venue_id bleibt null; Formular kann trotzdem gespeichert werden (Ort ist optional)

---

### Requirement: Neuen Venue inline anlegen
Der VenuePicker SHALL eine Option „+ Neuen Ort anlegen" anbieten, die ein Modal öffnet.

#### Scenario: Neuen Venue anlegen und direkt wählen
- **WHEN** Nutzer klickt „+ Neuen Ort anlegen", füllt Formular aus und bestätigt
- **THEN** Venue wird via POST /api/admin/venues angelegt; Modal schließt sich; neuer Venue wird im Picker direkt ausgewählt

#### Scenario: Anlage abbrechen
- **WHEN** Nutzer schließt das Modal ohne Speichern
- **THEN** Kein Venue wird angelegt; Picker bleibt unverändert

---

### Requirement: Venue-Auswahl entfernen
Der VenuePicker SHALL eine Möglichkeit bieten, die Venue-Auswahl zu löschen.

#### Scenario: Auswahl löschen
- **WHEN** Nutzer klickt auf „Ort entfernen" oder wählt leere Option
- **THEN** venue_id wird auf null zurückgesetzt
