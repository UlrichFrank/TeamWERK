## ADDED Requirements

### Requirement: Anwesenheitserfassung überspringt abgemeldete Spieler

Bei der Bulk-Anwesenheitserfassung (`POST /api/training-sessions/{id}/attendances`) SHALL das System für jedes übergebene Mitglied, das für die Serie der Session eine greifende Serien-Abmeldung (`serien-abmeldung`-Ableitung) hat, **keine** `training_attendances`-Zeile schreiben — analog zur bestehenden Ausnahme für trainer-only-Mitglieder. Der übrige Speichervorgang für nicht abgemeldete Mitglieder SHALL unbeeinflusst bleiben (kein Abbruch des gesamten Requests). Dadurch entsteht für abgemeldete Spieler kein Attendance-Record und sie bleiben aus dem Statistik-Nenner ausgeschlossen.

#### Scenario: Abgemeldeter Spieler bekommt keine Attendance-Zeile

- **WHEN** ein Trainer `POST /api/training-sessions/{id}/attendances` mit einer Liste sendet, die ein für diese Serie abgemeldetes Mitglied enthält
- **THEN** wird für dieses Mitglied keine `training_attendances`-Zeile angelegt oder aktualisiert

#### Scenario: Restliche Erfassung bleibt erfolgreich

- **WHEN** dieselbe Anfrage neben dem abgemeldeten Mitglied weitere, nicht abgemeldete Mitglieder enthält
- **THEN** werden deren Anwesenheitswerte normal gespeichert und die Anfrage liefert HTTP 200

#### Scenario: Bereits erfasste Anwesenheit bei nachträglicher Abmeldung wird ausgeschlossen

- **WHEN** für ein Mitglied bereits eine `training_attendances`-Zeile existiert und danach eine greifende Serien-Abmeldung angelegt wird
- **THEN** wird die Session für dieses Mitglied in der Statistik dennoch ausgeschlossen (der Abmelde-Ausschluss hat Vorrang, siehe Capability `attendance-statistics`)
