# carpooling-team-filter Specification

## Purpose

Diese Spezifikation beschreibt die Capability `carpooling-team-filter`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: DB-View user_accessible_teams
Das System SHALL eine SQLite-View `user_accessible_teams(user_id, team_id, season_id)` bereitstellen, die alle User-Team-Zuordnungen rollenübergreifend kapselt:
- `spieler`: über `kader_members → kader → members.user_id`
- `elternteil`: über `family_links → kader_members → kader`
- `trainer`: über `kader_trainers → kader → members.user_id`

#### Scenario: Spieler-Zeilen in der View
- **WHEN** ein Spieler (user_id=5) in kader_id=3 (season_id=1, team_id=15) eingetragen ist
- **THEN** enthält die View die Zeile `(user_id=5, team_id=15, season_id=1)`

#### Scenario: Elternteil-Zeilen via family_links
- **WHEN** Elternteil (user_id=4) via family_links mit Kind verbunden ist, das in kader_id=3 (team_id=15, season_id=1) ist
- **THEN** enthält die View die Zeile `(user_id=4, team_id=15, season_id=1)`

#### Scenario: Nutzer ohne Zuordnung
- **WHEN** ein Nutzer keine kader_members, kader_trainers oder family_links-Einträge hat
- **THEN** enthält die View keine Zeilen für diesen user_id

### Requirement: Rollenabhängige Spielliste im Backend
`GET /api/mitfahrgelegenheiten` SHALL Spiele nach Team-Zugehörigkeit des anfragenden Nutzers filtern. Filterlogik:
- `admin`, `vorstand`: alle zukünftigen Spiele (kein Filter, View wird nicht genutzt)
- alle anderen Rollen: nur Spiele, deren team_id in `user_accessible_teams` für den Nutzer und die aktive Saison enthalten ist

#### Scenario: Elternteil sieht nur Team-Spiele
- **WHEN** ein Nutzer mit Rolle `elternteil` `GET /api/mitfahrgelegenheiten` aufruft
- **THEN** werden nur Spiele zurückgegeben, die zum Team mindestens eines verknüpften Kindes gehören (aktive Saison)

#### Scenario: Admin sieht alle Spiele
- **WHEN** ein Nutzer mit Rolle `admin` oder `vorstand` den Endpunkt aufruft
- **THEN** werden alle zukünftigen Spiele zurückgegeben

#### Scenario: Nutzer ohne Team-Zuordnung
- **WHEN** ein Elternteil ohne `family_links` oder ein Spieler ohne Kader-Eintrag die Liste abruft
- **THEN** wird eine leere Spielliste zurückgegeben (HTTP 200, kein Fehler)

### Requirement: Optionaler team_id-Filter
`GET /api/mitfahrgelegenheiten?team_id=X` SHALL die Ergebnisse auf das angegebene Team einschränken. Der Filter wird nur angewendet, wenn `team_id` in den zugänglichen Teams des Nutzers liegt.

#### Scenario: Elternteil filtert auf ein Team
- **WHEN** ein Elternteil mit Kindern in zwei Teams `?team_id=15` übergibt
- **THEN** werden nur Spiele von Team 15 zurückgegeben

#### Scenario: Ungültige team_id (kein Zugriff)
- **WHEN** ein Nutzer eine team_id übergibt, die nicht in seinen zugänglichen Teams liegt
- **THEN** wird eine leere Spielliste zurückgegeben (HTTP 200, kein Fehler, keine Fehlermeldung)
