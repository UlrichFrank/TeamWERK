## MODIFIED Requirements

### Requirement: Preview vor dem Anlegen

Das System SHALL via `GET /api/absences/preview` alle Events auflisten, die beim Speichern der Abwesenheit auto-declined würden — unabhängig davon, ob bereits eine Response existiert.

Der Endpoint gibt zwei Gruppen zurück:
1. Events mit bestehender `confirmed`-Response (werden von `declined` auf declined geändert)
2. Training-Sessions ohne bisherige Response, bei denen der Member Kader-Mitglied ist (bekommen neu eine `declined`-Response)

Jedes `previewEvent` hat ein Feld `pending: bool` — `false` für bestätigte, `true` für unbeantwortete Sessions.

#### Scenario: Preview ohne Konflikte

- **WHEN** der Nutzer einen Zeitraum abfragt, in dem der Member keine bestätigten Events hat und kein Kader-Mitglied bei Training-Sessions im Zeitraum ist
- **THEN** gibt die API eine leere Liste zurück

#### Scenario: Preview mit bestätigten Events

- **WHEN** der Nutzer einen Zeitraum mit mindestens einer `confirmed` Training- oder Spiel-Zusage abfragt
- **THEN** enthält die Antwort diese Events mit `pending: false`

#### Scenario: Preview mit unbeantworteten Training-Sessions

- **WHEN** der Nutzer einen Zeitraum abfragt und der Member Kader-Mitglied eines Teams mit Training-Sessions in diesem Zeitraum ist, ohne bisherige Response
- **THEN** enthält die Antwort diese Sessions mit `pending: true`

#### Scenario: Preview zeigt beide Gruppen

- **WHEN** der Nutzer einen Zeitraum abfragt mit sowohl bestätigten Events als auch unbeantworteten Sessions
- **THEN** enthält die Antwort alle betroffenen Events (confirmed + pending)
