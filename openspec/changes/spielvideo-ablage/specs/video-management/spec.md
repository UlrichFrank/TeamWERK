## ADDED Requirements

### Requirement: Video erfassen
Admin und Nutzer mit Trainer-Rolle oder Trainer-Funktion SHALL Videos via `POST /api/videos` erfassen können. Ein Video MUST mindestens `youtube_id` (11 Zeichen), `title` und `visibility` enthalten. Optional: `team_id`, `game_date`, `description`.

#### Scenario: Erfolgreiches Anlegen
- **WHEN** ein Admin oder Trainer `POST /api/videos` mit gültiger `youtube_id` aufruft
- **THEN** wird ein DB-Eintrag angelegt und `201 Created` mit der neuen Video-ID zurückgegeben

#### Scenario: Anlegen ohne Berechtigung
- **WHEN** ein Nutzer mit Rolle `spieler` `POST /api/videos` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Ungültige YouTube-ID
- **WHEN** eine `youtube_id` mit mehr oder weniger als 11 Zeichen übergeben wird
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Video bearbeiten
Admin und Trainer SHALL Video-Metadaten via `PUT /api/videos/:id` aktualisieren können.

#### Scenario: Erfolgreiche Aktualisierung
- **WHEN** ein berechtigter Nutzer `PUT /api/videos/:id` mit neuen Metadaten aufruft
- **THEN** werden die Daten in der DB aktualisiert und `200 OK` zurückgegeben

### Requirement: Video löschen
Admin und Trainer SHALL Videos via `DELETE /api/videos/:id` löschen können. Nur der DB-Eintrag wird gelöscht; das YouTube-Video bleibt unberührt.

#### Scenario: Erfolgreiche Löschung
- **WHEN** ein berechtigter Nutzer `DELETE /api/videos/:id` aufruft
- **THEN** wird der DB-Eintrag entfernt und `204 No Content` zurückgegeben

#### Scenario: Löschen ohne Berechtigung
- **WHEN** ein Nutzer mit Rolle `spieler` `DELETE /api/videos/:id` aufruft
- **THEN** antwortet der Server mit HTTP 403
