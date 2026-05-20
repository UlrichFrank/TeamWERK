## ADDED Requirements

### Requirement: Vereinsfunktion auf Mitglied hinterlegbar
Ein Mitglied MUSS eine optionale Vereinsfunktion (`club_function`) haben können, unabhängig von der System-Benutzerrolle. Gültige Werte: `trainer`, `vorstand`, `vorstand_beisitzer`. Die Funktion MUSS im Mitglieder-Detailformular anzeigbar und änderbar sein.

#### Scenario: Admin setzt Vereinsfunktion auf Trainer
- **WHEN** Admin im Mitglieder-Detailformular "Trainer" als Vereinsfunktion wählt und speichert
- **THEN** wird `PUT /api/members/{id}` mit `{ club_function: "trainer" }` aufgerufen und die Vereinsfunktion wird gespeichert

#### Scenario: Admin entfernt Vereinsfunktion
- **WHEN** Admin im Mitglieder-Detailformular "– keine –" wählt und speichert
- **THEN** wird `PUT /api/members/{id}` mit `{ club_function: null }` aufgerufen

#### Scenario: Vereinsfunktion in Mitgliederliste sichtbar
- **WHEN** `GET /api/members` oder `GET /api/members/{id}` aufgerufen wird
- **THEN** enthält die Antwort das Feld `club_function` (null oder einer der drei Werte)

#### Scenario: Filterung nach Vereinsfunktion
- **WHEN** `GET /api/members?club_function=trainer` aufgerufen wird
- **THEN** gibt die API nur Mitglieder zurück, bei denen `club_function = 'trainer'` gesetzt ist

### Requirement: Mehrere Trainer pro Kader zuweisbar
Jeder Kader-Eintrag MUSS eine Liste von zugewiesenen Trainern (`trainers: [{id, name}]`) unterstützen. Trainer werden aus Mitgliedern mit `club_function = 'trainer'` gewählt. Es MUSS möglich sein, beliebig viele Trainer hinzuzufügen und einzelne zu entfernen.

#### Scenario: Trainer wird einem Kader hinzugefügt
- **WHEN** Admin im "Trainer hinzufügen"-Select ein Mitglied mit Trainer-Funktion auswählt
- **THEN** wird `PUT /api/admin/kader/{id}` mit `{ trainers_add: [member_id] }` aufgerufen und der Trainer erscheint als Chip in der Karte

#### Scenario: Trainer wird von einem Kader entfernt
- **WHEN** Admin auf × neben einem Trainer-Chip klickt
- **THEN** wird `PUT /api/admin/kader/{id}` mit `{ trainers_remove: [member_id] }` aufgerufen und der Chip verschwindet

#### Scenario: Kader ohne Trainer
- **WHEN** `GET /api/admin/kader` aufgerufen wird und ein Kader keine Trainer hat
- **THEN** enthält das Kader-Objekt `trainers: []`

#### Scenario: Add-Select zeigt nur nicht-zugewiesene Trainer
- **WHEN** Trainer A bereits dem Kader zugewiesen ist
- **THEN** erscheint Trainer A nicht im "Trainer hinzufügen"-Dropdown

### Requirement: Jahrgangs-Dropdown ist nicht durch die Kader-Karte geclippt
Der native `<select>` für die Jahrgangs-Auswahl MUSS vollständig sichtbar sein und DARF nicht durch den Kader-Karten-Container abgeschnitten werden.

#### Scenario: Dropdown öffnet sich über den Kartenrand hinaus
- **WHEN** Admin auf das Jahrgangs-Dropdown klickt
- **THEN** erscheint die native Select-Liste vollständig sichtbar, unabhängig von der Kartenhöhe

#### Scenario: Dropdown öffnet sich beim zweiten Klick
- **WHEN** Admin das Dropdown schließt und erneut darauf klickt
- **THEN** öffnet sich das Dropdown sofort ohne vorheriges Klicken auf ein anderes Element
