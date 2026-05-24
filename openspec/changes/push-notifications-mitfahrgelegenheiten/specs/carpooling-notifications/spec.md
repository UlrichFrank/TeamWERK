## ADDED Requirements

### Requirement: Notification bei neuem Angebot
Wenn ein User einen "biete"-Eintrag erstellt oder aktualisiert, MÜSSEN alle User mit einem "suche"-Eintrag für dasselbe Spiel eine Push Notification erhalten.

#### Scenario: Neues Angebot, Suchende vorhanden
- **WHEN** User A `POST /api/mitfahrgelegenheiten` mit `typ: "biete"` für Spiel X aufruft
- **THEN** erhalten alle User mit einem "suche"-Eintrag für Spiel X eine Push Notification mit Titel "Mitfahrgelegenheit" und Text "[Name] bietet Plätze an — [Gegner], [Datum]"
- **THEN** Die Notification enthält eine URL zu `/mitfahrgelegenheiten`

#### Scenario: Neues Angebot, keine Suchenden
- **WHEN** User A `POST /api/mitfahrgelegenheiten` mit `typ: "biete"` postet und niemand "suche" hat
- **THEN** wird kein Push gesendet

#### Scenario: Eigener Eintrag erzeugt keinen Self-Push
- **WHEN** User A einen "biete"-Eintrag erstellt
- **THEN** erhält User A selbst keine Push Notification, auch wenn User A einen "suche"-Eintrag für dasselbe Spiel hätte

### Requirement: Notification bei neuer Suchanfrage
Wenn ein User einen "suche"-Eintrag erstellt oder aktualisiert, MÜSSEN alle User mit einem "biete"-Eintrag für dasselbe Spiel eine Push Notification erhalten.

#### Scenario: Neue Suchanfrage, Anbietende vorhanden
- **WHEN** User B `POST /api/mitfahrgelegenheiten` mit `typ: "suche"` für Spiel X aufruft
- **THEN** erhalten alle User mit einem "biete"-Eintrag für Spiel X eine Push Notification mit Titel "Mitfahrgelegenheit" und Text "[Name] sucht noch einen Platz — [Gegner], [Datum]"
- **THEN** Die Notification enthält eine URL zu `/mitfahrgelegenheiten`

#### Scenario: Neue Suchanfrage, keine Anbietenden
- **WHEN** User B `POST /api/mitfahrgelegenheiten` mit `typ: "suche"` postet und niemand "biete" hat
- **THEN** wird kein Push gesendet

### Requirement: Notification bei zurückgezogenem Angebot
Wenn ein User seinen "biete"-Eintrag löscht, MÜSSEN alle User mit einem "suche"-Eintrag für dasselbe Spiel eine Push Notification erhalten.

#### Scenario: Angebot zurückgezogen, Suchende vorhanden
- **WHEN** User A `DELETE /api/mitfahrgelegenheiten/{id}` für einen "biete"-Eintrag aufruft
- **THEN** erhalten alle User mit einem "suche"-Eintrag für dasselbe Spiel eine Push Notification mit Text "[Name] hat sein Angebot zurückgezogen — [Gegner], [Datum]"

#### Scenario: Gelöschter Eintrag war "suche" — kein Push
- **WHEN** ein User seinen eigenen "suche"-Eintrag löscht
- **THEN** wird kein Push gesendet

### Requirement: Push-Versand ist nicht blockierend
Das Senden von Push Notifications DARF die HTTP-Antwort des Carpooling-Handlers nicht verzögern.

#### Scenario: Push wird asynchron gesendet
- **WHEN** der Carpooling-Upsert oder -Delete erfolgreich in der DB gespeichert wird
- **THEN** antwortet der Handler sofort mit dem entsprechenden HTTP-Statuscode und der Push-Versand läuft in einer separaten Goroutine
