## ADDED Requirements

### Requirement: Push bei Spiel-Ereignissen
Das System SHALL allen berechtigten Team-Mitgliedern und deren Elternteilen eine Push Notification senden, wenn ein Spiel erstellt, geändert oder gelöscht wird — sofern der Nutzer Push für die Kategorie `games` nicht deaktiviert hat.

#### Scenario: Neues Spiel erstellt
- **WHEN** ein Admin oder Trainer ein neues Spiel über `POST /api/admin/games` anlegt
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification mit Titel „Neues Spiel" und der Gegnerinfo

#### Scenario: Spiel verschoben oder geändert
- **WHEN** ein Admin oder Trainer ein Spiel über `PUT /api/admin/games/{id}` aktualisiert (Datum, Zeit oder Ort geändert)
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification „Spielinfo geändert"

#### Scenario: Spiel abgesagt (gelöscht)
- **WHEN** ein Admin oder Trainer ein Spiel über `DELETE /api/admin/games/{id}` löscht
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification „Spiel abgesagt"

#### Scenario: Nutzer mit deaktiviertem Push erhält keine Notification
- **WHEN** ein Spiel-Ereignis eintritt und ein Nutzer hat `push_enabled=0` für Kategorie `games` in `notification_preferences`
- **THEN** erhält dieser Nutzer keine Push Notification für dieses Ereignis
