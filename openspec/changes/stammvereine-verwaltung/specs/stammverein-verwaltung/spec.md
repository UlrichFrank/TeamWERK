## ADDED Requirements

### Requirement: Stammvereine verwalten
Nutzer mit Vereinsfunktion `vorstand` (sowie System-Rolle `admin`) SHALL die Liste der Stammvereine pflegen können. Über `POST /api/stammvereine` MUST ein neuer Verein mit eindeutigem `name` angelegt werden. Über `PUT /api/stammvereine/{id}` MUST ein Verein umbenannt oder sein `aktiv`-Flag umgeschaltet werden können. Über `DELETE /api/stammvereine/{id}` MUST der Verein **soft-deleted** (`aktiv=0`) werden — niemals physisch gelöscht, solange Mitglieder ihn referenzieren. Jede Mutation MUST `Broadcast("stammvereine")` auslösen.

#### Scenario: Verein anlegen
- **WHEN** ein Vorstand `POST /api/stammvereine` mit `{"name":"SV Beispiel 1900"}` sendet
- **THEN** antwortet der Server mit HTTP 201 und der Verein erscheint in `GET /api/stammvereine`

#### Scenario: Doppelter Name
- **WHEN** ein Vorstand einen Verein mit bereits existierendem `name` anlegt
- **THEN** antwortet der Server mit HTTP 409

#### Scenario: Soft-Delete erhält Mitglieder-Referenz
- **WHEN** ein Verein per `DELETE /api/stammvereine/{id}` deaktiviert wird, dem Mitglieder zugeordnet sind
- **THEN** wird `aktiv=0` gesetzt, die `members.home_club_id`-Referenzen bleiben bestehen, und der Verein verschwindet aus der Standard-Liste (`GET /api/stammvereine` ohne `include_inactive`)

#### Scenario: Zugriff ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `POST /api/stammvereine` aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Stammvereine für Mitglieder-Auswahl bereitstellen
Jeder eingeloggte Nutzer MUST über `GET /api/stammvereine` die Liste der **aktiven** Stammvereine abrufen können (für das Auswahl-Dropdown auf der Mitgliederseite). Mit `?include_inactive=1` MUST für `vorstand`/`admin` auch deaktivierte Vereine geliefert werden.

#### Scenario: Liste nur aktive
- **WHEN** ein eingeloggter Nutzer `GET /api/stammvereine` aufruft
- **THEN** enthält die Antwort nur Vereine mit `aktiv=1`

#### Scenario: Unauthentifiziert
- **WHEN** `GET /api/stammvereine` ohne gültiges Access-Token aufgerufen wird
- **THEN** antwortet der Server mit HTTP 401
