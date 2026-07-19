## ADDED Requirements

### Requirement: RSVP für abgemeldete Session gesperrt

Das System SHALL eine RSVP-Antwort (`POST /api/training-sessions/{id}/respond`) mit HTTP 403 ablehnen, wenn für das betroffene Mitglied und die Serie der Session eine greifende Serien-Abmeldung (`serien-abmeldung`-Ableitung) existiert. Dies gilt unabhängig davon, ob der Spieler selbst oder ein Elternteil für ein Kind antwortet. Die Prüfung erfolgt live gegen `member_series_unavailabilities`; es werden keine `training_responses`-Zeilen vorab angelegt.

#### Scenario: Spieler kann für abgemeldete Serie nicht antworten

- **WHEN** ein Spieler `POST /api/training-sessions/{id}/respond` für eine Session aufruft, die von einer greifenden Serien-Abmeldung erfasst ist
- **THEN** antwortet das System mit HTTP 403 und legt/ändert keine `training_responses`-Zeile

#### Scenario: Elternteil kann für abgemeldetes Kind nicht antworten

- **WHEN** ein Elternteil für ein verlinktes, für diese Serie abgemeldetes Kind antworten will
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Nicht betroffene Session bleibt beantwortbar

- **WHEN** die Session außerhalb des Abmelde-Fensters liegt oder keine Abmeldung existiert
- **THEN** funktioniert die RSVP wie bisher (HTTP 200/201)
