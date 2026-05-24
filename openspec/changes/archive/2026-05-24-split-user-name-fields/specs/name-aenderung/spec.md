## MODIFIED Requirements

### Requirement: Eingeloggter Nutzer kann Vorname und Nachname ändern
Das System SHALL jedem authentifizierten Nutzer erlauben, seinen eigenen Vornamen (`users.first_name`) und Nachnamen (`users.last_name`) separat zu ändern. Diese Felder gehören zur Kontoidentität und sind unabhängig von `members.first_name`/`last_name` (Vereinsmitgliedsdaten). Eine Änderung hier löst keinen Mitgliedsdaten-Workflow aus und berührt die `members`-Tabelle nicht. Eine Passwort-Verifikation ist nicht erforderlich.

#### Scenario: Vor- und Nachname erfolgreich ändern
- **WHEN** ein eingeloggter Nutzer `PUT /api/profile/account` mit `{ "first_name": "Maria", "last_name": "Muster" }` aufruft
- **THEN** werden `users.first_name` und `users.last_name` für den aufrufenden Nutzer aktualisiert, HTTP 204 zurückgegeben, und die `members`-Tabelle bleibt unverändert

#### Scenario: Leerer Vorname wird abgelehnt
- **WHEN** `PUT /api/profile/account` mit leerem oder fehlendem `first_name`-Feld aufgerufen wird
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Nicht eingeloggte Anfrage wird abgelehnt
- **WHEN** `PUT /api/profile/account` ohne gültigen Bearer-Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 401

### Requirement: Profilseite zeigt zwei bearbeitbare Namensfelder
Das Frontend SHALL auf der Profilseite separate Eingabefelder für Vorname und Nachname anzeigen, die mit den aktuellen Werten vorbelegt sind und per Speichern-Button gespeichert werden können.

#### Scenario: Namensfelder vorbelegt
- **WHEN** ein Nutzer die Profilseite aufruft
- **THEN** sind Vorname-Feld mit `first_name` und Nachname-Feld mit `last_name` vorbelegt

#### Scenario: Speichern zeigt Bestätigung
- **WHEN** ein Nutzer Vor- oder Nachname ändert und speichert
- **THEN** erscheint eine kurze Erfolgsmeldung; die neuen Werte bleiben in den Feldern stehen
