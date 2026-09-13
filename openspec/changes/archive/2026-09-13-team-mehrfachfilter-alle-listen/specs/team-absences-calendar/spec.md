## MODIFIED Requirements

### Requirement: Backend liefert Team-Abwesenheiten nur für berechtigte Rollen
`GET /api/absences/calendar?show_team=true` SHALL für berechtigte Nutzer zusätzlich Abwesenheiten von Teammitgliedern mit `absences_public = 1` zurückgeben, die in mindestens einem der eigenen Teams der aktiven Saison sind.

Der optionale Parameter `team_id` SHALL eine **kommaseparierte ID-Liste** annehmen (`team_id=3,7`); eine einzelne ID (`team_id=5`) bleibt gültig. Er schränkt die Team-Abwesenheiten auf Mitglieder der genannten Mannschaften ein — zusätzlich zur bestehenden Prüfung gegen `user_accessible_teams`, die unverändert bleibt. Unbrauchbare Werte werden still verworfen; ein Filter, von dem keine gültige ID übrig bleibt, wirkt wie kein Filter (keine Fehlermeldung, keine leere Antwort).

#### Scenario: Berechtigt — Team-Abwesenheiten werden zurückgegeben
- **WHEN** ein Trainer `GET /api/absences/calendar?show_team=true` aufruft
- **THEN** enthält die Antwort zusätzlich Abwesenheiten von Mitgliedern seiner Teams mit `absences_public = 1`

#### Scenario: Unberechtigt — Parameter wird ignoriert
- **WHEN** ein Spieler `GET /api/absences/calendar?show_team=true` aufruft
- **THEN** enthält die Antwort nur eigene und Kinder-Abwesenheiten (wie ohne Parameter)

#### Scenario: Team-Filter einschränken
- **WHEN** ein Trainer `GET /api/absences/calendar?show_team=true&team_id=5` aufruft
- **THEN** enthält die Antwort nur Team-Abwesenheiten von Mitgliedern des Teams 5

#### Scenario: Mehrere Teams einschränken
- **WHEN** ein Trainer `GET /api/absences/calendar?show_team=true&team_id=5,8` aufruft
- **THEN** enthält die Antwort Team-Abwesenheiten der Teams 5 und 8 und keine anderen

#### Scenario: Unbrauchbarer Filter
- **WHEN** ein Trainer `GET /api/absences/calendar?show_team=true&team_id=abc` aufruft
- **THEN** verhält sich die Antwort wie ohne `team_id` (alle eigenen Teams), ohne Fehler

### Requirement: Team-Abwesenheiten werden mit bestehendem Team-Filter koordiniert
Das System SHALL bei aktivem Mannschafts-Filter nur Abwesenheiten der gewählten Mannschaften laden (`?team_id={ID-Liste}`), wenn Team-Abwesenheiten eingeblendet sind. Der Kalender-Filter ist eine Mehrfachauswahl; die Abwesenheiten sind seine einzige serverseitig gefilterte Datenquelle, alle übrigen (Spiele, Trainings) filtert die Seite clientseitig.

#### Scenario: Team-Filter aktiv
- **WHEN** Trainer wählt im Dropdown nur Team A und aktiviert Team-Abwesenheiten
- **THEN** werden nur Abwesenheiten von Mitgliedern des Teams A angezeigt

#### Scenario: Mehrere Teams gewählt
- **WHEN** Trainer hakt Team A und Team B an (von drei) und aktiviert Team-Abwesenheiten
- **THEN** wird `?team_id=A,B` angefragt und nur deren Abwesenheiten angezeigt

#### Scenario: Kein Team-Filter
- **WHEN** Trainer aktiviert Team-Abwesenheiten ohne Team-Filter
- **THEN** wird kein `team_id` mitgeschickt und Abwesenheiten aller eigenen Teams werden angezeigt
