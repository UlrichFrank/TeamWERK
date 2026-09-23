## ADDED Requirements

### Requirement: Abwesenheit bearbeiten
Das System SHALL einen `PUT /api/absences/{id}` Endpoint bereitstellen, über den Typ, Start-/Enddatum und Notiz einer Abwesenheit geändert werden können. Nur der Ersteller (`created_by`) oder ein Admin darf eine Abwesenheit bearbeiten. Ein Überlappungscheck verhindert, dass der neue Zeitraum mit einer anderen Abwesenheit desselben Typs für dasselbe Mitglied kollidiert (eigene ID ausgenommen). Nach dem Speichern werden Auto-Decline-Responses für den alten Zeitraum zurückgesetzt und für den neuen Zeitraum neu angelegt.

#### Scenario: Eigene Abwesenheit erfolgreich bearbeiten
- **WHEN** der Ersteller `PUT /api/absences/{id}` mit gültigem Typ, Start- und Enddatum aufruft
- **THEN** wird die Abwesenheit aktualisiert, Auto-Declines werden neu gesetzt und HTTP 200 zurückgegeben

#### Scenario: Fremde Abwesenheit bearbeiten abgewiesen
- **WHEN** ein Nutzer `PUT /api/absences/{id}` für eine Abwesenheit aufruft, die nicht ihm gehört und er kein Admin ist
- **THEN** antwortet die API mit HTTP 403

#### Scenario: Überlappung mit anderer Abwesenheit gleichen Typs
- **WHEN** der neue Zeitraum mit einer anderen Abwesenheit desselben Typs für dasselbe Mitglied überlappt
- **THEN** antwortet die API mit HTTP 409 und Body `{"error":"overlap"}`

#### Scenario: Überlappung mit sich selbst ist erlaubt
- **WHEN** ein Nutzer `PUT /api/absences/{id}` aufruft ohne Änderung des Zeitraums
- **THEN** wird die Abwesenheit ohne Fehler gespeichert (eigene ID ist vom Overlap-Check ausgenommen)

#### Scenario: Auto-Decline nach Zeitraumänderung
- **WHEN** der Zeitraum einer Abwesenheit verändert wird
- **THEN** werden Responses, die durch die alte Abwesenheit auto-gedeclint wurden, zurückgesetzt; für Events im neuen Zeitraum werden neue Auto-Declines angelegt

### Requirement: Abwesenheit im Kalender anzeigen
Die `KalenderPage` SHALL beim Klick auf einen Abwesenheitsbalken ein Info-Modal öffnen, das Typ, Mitgliedsname, Zeitraum und Notiz anzeigt. Das Modal folgt dem selben Muster wie Spiele und Trainings (`EventInfoModal` mit neuem `'absence'`-Zweig).

#### Scenario: Klick auf Abwesenheitsbalken
- **WHEN** ein Nutzer auf einen Abwesenheitsbalken im Monatskalender klickt
- **THEN** öffnet sich ein Modal mit Typ-Label (Urlaub / Verletzung), Mitgliedsname, Zeitraum (von–bis) und Notiz (falls vorhanden)

#### Scenario: Bearbeiten- und Löschen-Button für Ersteller
- **WHEN** der eingeloggte Nutzer der Ersteller der Abwesenheit ist (oder Admin)
- **THEN** sind Bearbeiten- und Löschen-Button im Modal sichtbar

#### Scenario: Kein Bearbeiten für fremde Abwesenheiten
- **WHEN** der eingeloggte Nutzer nicht der Ersteller ist und kein Admin
- **THEN** sind weder Bearbeiten- noch Löschen-Button sichtbar

#### Scenario: Inline-Edit öffnet sich im selben Modal
- **WHEN** der Nutzer auf Bearbeiten klickt
- **THEN** wechselt das Modal in den Edit-Modus und zeigt ein Formular mit Typ, Start-/Enddatum und Notiz

#### Scenario: Speichern schließt Edit-Modus
- **WHEN** der Nutzer das Formular erfolgreich speichert
- **THEN** kehrt das Modal in den Anzeige-Modus zurück und zeigt die aktualisierten Daten
