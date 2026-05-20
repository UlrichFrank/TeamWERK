## ADDED Requirements

### Requirement: Admin kann ein globales Heimspiel-Template konfigurieren
Das System SHALL ein globales Template verwalten, das definiert welche Duty-Slot-Typen mit welchem Zeitversatz und welcher Personenanzahl pro Heimspiel generiert werden. Es gibt maximal ein aktives Template.

#### Scenario: Template abrufen
- **WHEN** ein Admin `GET /api/admin/game-template` aufruft
- **THEN** antwortet das System mit dem aktiven Template inkl. aller Items (duty_type_id, duty_type_name, anchor, offset_minutes, slots_count, role_desc) oder einem leeren Template wenn keines existiert

#### Scenario: Template-Items setzen
- **WHEN** ein Admin `PUT /api/admin/game-template` mit einem Array von Items aufruft
- **THEN** werden alle bestehenden Items des Templates gelöscht und durch die neuen ersetzt, HTTP 204

#### Scenario: Template-Item mit ungültigem duty_type_id abgelehnt
- **WHEN** ein Admin `PUT /api/admin/game-template` mit einer nicht existierenden `duty_type_id` aufruft
- **THEN** antwortet das System mit HTTP 400

### Requirement: Template-Items haben Zeitanker und Versatz
Jedes Template-Item SHALL einen Anker (`"start"` = Anpfiff, `"end"` = Spielende) und einen `offset_minutes`-Wert haben. Negative Werte bedeuten vor dem Anker, positive danach.

#### Scenario: Aufbau-Slot vor Anpfiff
- **WHEN** ein Template-Item mit `anchor = "start"`, `offset_minutes = -60` und `slots_count = 2` konfiguriert ist und ein Spiel um 15:00 Uhr angelegt wird
- **THEN** wird der generierte Duty Slot mit Event-Zeit 14:00 Uhr angelegt und `slots_total = 2`
