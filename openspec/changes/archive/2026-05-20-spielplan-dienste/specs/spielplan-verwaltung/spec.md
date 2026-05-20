## ADDED Requirements

### Requirement: Admin kann Heimspiele anlegen
Das System SHALL Admins erlauben, Heimspiele mit Datum, Uhrzeit, Gegner, Mannschaft und Saison anzulegen. Jedes Spiel erhält `source = "manual"` und eine optionale Verknüpfung zu einem Template.

#### Scenario: Heimspiel erfolgreich anlegen
- **WHEN** ein Admin `POST /api/admin/games` mit gültigen Feldern (date, time, opponent, team_id, season_id) aufruft
- **THEN** wird das Spiel in der Datenbank gespeichert und die Response enthält die neue `game_id` mit HTTP 201

#### Scenario: Spiel ohne Pflichtfeld wird abgelehnt
- **WHEN** ein Admin `POST /api/admin/games` ohne `date` oder `team_id` aufruft
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Nicht-Admin kann keine Spiele anlegen
- **WHEN** ein Nutzer mit Rolle `trainer`, `elternteil` oder `spieler` `POST /api/admin/games` aufruft
- **THEN** antwortet das System mit HTTP 403

### Requirement: Admin kann Heimspiele bearbeiten und löschen
Das System SHALL Admins erlauben, bestehende Heimspiele zu bearbeiten (Datum, Uhrzeit, Gegner) und zu löschen. Beim Löschen werden verknüpfte `duty_slots.game_id` auf NULL gesetzt (ON DELETE SET NULL).

#### Scenario: Heimspiel bearbeiten
- **WHEN** ein Admin `PUT /api/admin/games/{id}` mit geänderten Feldern aufruft
- **THEN** werden die Felder aktualisiert und HTTP 204 zurückgegeben

#### Scenario: Heimspiel löschen — belegte Slots bleiben erhalten
- **WHEN** ein Admin `DELETE /api/admin/games/{id}` aufruft und das Spiel Duty Slots mit `fulfilled_at IS NOT NULL` hat
- **THEN** wird das Spiel gelöscht, die Slots bleiben erhalten mit `game_id = NULL`, HTTP 204

#### Scenario: Heimspiel löschen — nicht gefunden
- **WHEN** ein Admin `DELETE /api/admin/games/{id}` mit einer nicht existierenden ID aufruft
- **THEN** antwortet das System mit HTTP 404

### Requirement: Spielplan-Liste abrufbar
Das System SHALL eine Liste aller Heimspiele der aktiven Saison zurückgeben, gefiltert nach Saison und optional nach Team.

#### Scenario: Spielplan der aktiven Saison abrufen
- **WHEN** ein authentifizierter Nutzer `GET /api/games?season_id=<id>` aufruft
- **THEN** antwortet das System mit HTTP 200 und einem Array von Spielen mit Feldern: id, date, time, opponent, team_id, team_name, slot_count, filled_count
