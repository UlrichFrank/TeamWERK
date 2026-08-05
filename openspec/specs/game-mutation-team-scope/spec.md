# game-mutation-team-scope Specification

## Purpose

Diese Spezifikation beschreibt die Capability `game-mutation-team-scope`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Reine Trainer sind auf Events ihrer eigenen Mannschaften beschränkt

Das System SHALL für Nutzer mit der Vereinsfunktion `trainer`, die **nicht** zusätzlich
`sportliche_leitung` oder `vorstand` besitzen und **nicht** die System-Rolle `admin` haben
(nachfolgend „reiner Trainer"), bei jeder Event-Mutation prüfen, ob der Nutzer über
`kader_trainers` in der aktiven Saison Trainer mindestens einer beteiligten Mannschaft ist.
Schlägt die Prüfung fehl, antwortet das System mit HTTP 403.

Für `admin`, `vorstand` und `sportliche_leitung` findet keine Team-Scope-Prüfung statt.

#### Scenario: Trainer bearbeitet Event einer fremden Mannschaft

- **WHEN** ein reiner Trainer `PUT /api/games/{id}` auf ein Event aufruft, an dem keine seiner
  Mannschaften beteiligt ist
- **THEN** antwortet das System mit HTTP 403
- **THEN** bleibt das Event unverändert

#### Scenario: Trainer löscht Event einer fremden Mannschaft

- **WHEN** ein reiner Trainer `DELETE /api/games/{id}` auf ein Event aufruft, an dem keine seiner
  Mannschaften beteiligt ist
- **THEN** antwortet das System mit HTTP 403
- **THEN** existiert das Event weiterhin

#### Scenario: Trainer bearbeitet Event der eigenen Mannschaft

- **WHEN** ein reiner Trainer `PUT /api/games/{id}` auf ein Event aufruft, an dem mindestens eine
  seiner Mannschaften beteiligt ist
- **THEN** antwortet das System mit HTTP 200

#### Scenario: Sportliche Leitung ist nicht eingeschränkt

- **WHEN** ein Nutzer mit `sportliche_leitung` `PUT /api/games/{id}` auf ein beliebiges Event
  aufruft
- **THEN** antwortet das System mit HTTP 200, unabhängig von den beteiligten Mannschaften

### Requirement: Zulässige team_ids hängen vom Event-Typ ab

Das System SHALL die in `POST /api/games` bzw. `PUT /api/games/{id}` übergebenen `team_ids` für
reine Trainer typabhängig validieren:

- Bei `event_type` = `heim` oder `auswärts` MUSS **jede** übergebene `team_id` eine eigene
  Mannschaft des Trainers sein.
- Bei `event_type` = `generisch` MUSS **mindestens eine** übergebene `team_id` eine eigene
  Mannschaft des Trainers sein; alle weiteren `team_ids` dürfen beliebige aktive Mannschaften
  des Vereins sein.

Verletzt der Request diese Bedingung, antwortet das System mit HTTP 403 und nimmt keine Änderung
an `games` oder `game_teams` vor.

#### Scenario: Trainer legt generisches Event mit fremden Mannschaften an

- **WHEN** ein reiner Trainer `POST /api/games` mit `event_type='generisch'` und `team_ids`
  aufruft, die eine eigene und zwei fremde Mannschaften enthalten
- **THEN** antwortet das System mit HTTP 201
- **THEN** enthält `game_teams` Einträge für alle drei Mannschaften

#### Scenario: Trainer legt generisches Event ohne eigene Mannschaft an

- **WHEN** ein reiner Trainer `POST /api/games` mit `event_type='generisch'` und ausschließlich
  fremden `team_ids` aufruft
- **THEN** antwortet das System mit HTTP 403
- **THEN** wurde kein Event angelegt

#### Scenario: Trainer legt Heimspiel für fremde Mannschaft an

- **WHEN** ein reiner Trainer `POST /api/games` mit `event_type='heim'` und einer fremden
  `team_id` aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Trainer fügt einem generischen Event eine fremde Mannschaft hinzu

- **WHEN** ein reiner Trainer `PUT /api/games/{id}` auf ein generisches Event seiner Mannschaft
  aufruft und `team_ids` um eine fremde Mannschaft erweitert
- **THEN** antwortet das System mit HTTP 200
- **THEN** ist die fremde Mannschaft in `game_teams` eingetragen

#### Scenario: Trainer entfernt die letzte eigene Mannschaft aus einem Event

- **WHEN** ein reiner Trainer `PUT /api/games/{id}` mit `team_ids` aufruft, die keine seiner
  eigenen Mannschaften mehr enthalten
- **THEN** antwortet das System mit HTTP 403
- **THEN** bleiben die bisherigen `game_teams`-Einträge unverändert

#### Scenario: Vorstand darf beliebige Mannschaften setzen

- **WHEN** ein Nutzer mit `vorstand` `POST /api/games` mit beliebigen `team_ids` und beliebigem
  `event_type` aufruft
- **THEN** antwortet das System mit HTTP 201

### Requirement: Erweiterte Team-Auswahl wirkt nur auf das betroffene Event

Das Hinzufügen einer fremden Mannschaft zu einem generischen Event SHALL dem handelnden Trainer
**keine** Rechte auf den übrigen Terminen, dem Kader oder den Mitgliederdaten dieser Mannschaft
verschaffen. Die Regeln für Event-Sichtbarkeit (`event-team-visibility`), Kader-Verwaltung und
Mitglieder-Zugriff bleiben unverändert.

Innerhalb des geteilten Events gelten die bestehenden ereignisbezogenen Regeln unverändert
weiter — insbesondere erfasst jeder Trainer einer beteiligten Mannschaft die Anwesenheiten
**dieses** Events für alle Teilnehmer (`canRecordGameAttendance`, bestehendes Verhalten für
mannschaftsübergreifende Events). Das ist gewollt: wer das Vereinsfest organisiert, hakt ab, wer
da war. Ein Rückwirken auf die Anwesenheitsstatistik anderer Mannschaften entsteht dadurch
nicht, weil `attendance-statistics` ausschließlich Events vom Typ `heim`/`auswärts` auswertet.

#### Scenario: Kein Zugriff auf andere Events der eingeladenen Mannschaft

- **WHEN** ein reiner Trainer eine fremde Mannschaft zu einem generischen Event hinzugefügt hat
- **THEN** enthält `GET /api/games` für ihn weiterhin nur Events, an denen eine seiner eigenen
  Mannschaften beteiligt ist
- **THEN** antwortet `PUT /api/games/{id}` auf ein anderes Event dieser fremden Mannschaft mit
  HTTP 403

#### Scenario: Keine Statistik-Auswirkung auf die eingeladene Mannschaft

- **WHEN** ein reiner Trainer für ein Mitglied der eingeladenen Mannschaft die Anwesenheit auf
  dem geteilten generischen Event erfasst
- **THEN** erscheint dieser Eintrag nicht in der Anwesenheitsstatistik der eingeladenen
  Mannschaft, da nur `heim`/`auswärts`-Events in die Statistik eingehen

#### Scenario: Kein Zugriff auf Kader und Mitgliederdaten der eingeladenen Mannschaft

- **WHEN** ein reiner Trainer eine fremde Mannschaft zu einem generischen Event hinzugefügt hat
- **THEN** bleiben `GET /api/teams` (eigene Mannschaften) und die Mitglieder-Routen für ihn
  unverändert auf seinen bisherigen Umfang beschränkt
