# event-team-visibility Specification

## Purpose
TBD - created by archiving change event-team-visibility. Update Purpose after archive.
## Requirements
### Requirement: Event-Sichtbarkeit pro Nutzer

Das System SHALL einen zentralen Helper `auth.UserCanSeeGame(ctx, db, userID, gameID)` und `auth.GameIDsVisibleToUser(ctx, db, userID, seasonID)` bereitstellen. Beide SHALL die Sichtbarkeit eines Events nach folgender Regel bestimmen:

Ein Nutzer N SHALL ein Event E sehen genau dann, wenn:

1. N hat die Funktion `admin`, `trainer`, `sportliche_leitung` ODER `vorstand` (Bypass), ODER
2. N selbst ist Mitglied (über `members.user_id`) eines Teams T mit T ∈ `game_teams(E)` und ein Eintrag in `kader_members` oder `kader_extended_members` zu einem `kader` mit `team_id = T` und `season_id = games(E).season_id` existiert, ODER
3. ein Kind von N (via `family_links`) erfüllt die Bedingung 2.

`kassierer` und `vorstand_beisitzer` haben **keinen** Bypass.

#### Scenario: Funktionsträger sieht alles

- **WHEN** `UserCanSeeGame(ctx, db, userID, gameID)` mit User mit `trainer`-Funktion
- **THEN** liefert `true` für jedes Game

#### Scenario: Eltern sehen Events der Kinder-Teams

- **WHEN** ein Elternteil ohne eigene Team-Mitgliedschaft, dessen Kind in Team A spielt
- **THEN** `UserCanSeeGame` liefert `true` für Events mit Team A in `game_teams`

#### Scenario: Fremder Nutzer ohne Team-Bezug

- **WHEN** ein Standard-Nutzer ohne Team-Mitgliedschaft (eigen oder Kinder) zu einem Event
- **THEN** `UserCanSeeGame` liefert `false`

### Requirement: Listen-Routen filtern nach Sichtbarkeit

Alle Listen-Routen, die Events ausliefern, SHALL die Antwort auf die für den Caller sichtbaren Events einschränken. Dies SHALL gelten für:

- `GET /api/games`
- `GET /api/dashboard` (alle Blöcke mit Game-Bezug)
- `GET /api/calendar` (bzw. die zuständige Calendar-Route)
- `GET /api/mitfahrgelegenheiten` (sofern teamübergreifend)

Funktionsträger SHALL kein Filter erhalten.

#### Scenario: Spieler sieht nur eigene Team-Games in /api/games

- **WHEN** ein Spieler aus Team A `GET /api/games` aufruft und Events für Teams A, B, C existieren
- **THEN** enthält die Response nur Events mit Team A in `game_teams`

#### Scenario: Trainer sieht alle Games

- **WHEN** ein Trainer `GET /api/games` aufruft
- **THEN** enthält die Response alle Games der Saison

#### Scenario: Dashboard zeigt nur sichtbare Termine

- **WHEN** ein Spieler aus Team A das Dashboard öffnet
- **THEN** enthält der Block „Nächste Termine" nur Events mit Team A in `game_teams`

### Requirement: Detail-Routen liefern 404 statt 403 bei fehlender Sichtbarkeit

Alle Detail-Routen mit `{game_id}`-Pfadparameter SHALL `404 Not Found` zurückgeben, wenn `UserCanSeeGame` für den Caller `false` liefert. Dies SHALL gelten für:

- `GET /api/games/{id}`
- `GET /api/games/{id}/participants`
- `GET /api/games/{id}/lineup`
- `GET /api/games/{id}/duty-slots`
- `POST /api/mitfahrgelegenheiten` (bei game_id im Body) und alle Mitfahr-Sub-Routen

#### Scenario: Direkter ID-Zugriff auf fremdes Game

- **WHEN** ein Spieler aus Team A `GET /api/games/{id}` für ein Event aufruft, das nur Teams B+C umfasst
- **THEN** antwortet der Server mit 404

#### Scenario: Direkter ID-Zugriff für Trainer

- **WHEN** ein Trainer `GET /api/games/{id}` für ein beliebiges Event aufruft
- **THEN** antwortet der Server mit 200 und liefert die Daten

#### Scenario: Carpooling-Anlage zu fremdem Game

- **WHEN** ein Nutzer ohne Team-Bezug versucht, ein Mitfahr-Gesuch zu einem fremden Event anzulegen
- **THEN** antwortet der Server mit 404

### Requirement: Push-Notifications synchron mit Event-Sichtbarkeit

Push-Notifications zu Events (Erstellung, Änderung, Absage, Carpooling-Events) SHALL ausschließlich an Nutzer gesendet werden, für die `UserCanSeeGame` zum Zeitpunkt des Push-Versands `true` liefert.

Inhaltlich gerichtete Pushes (z.B. „Aufstellung geändert" an Trainer/sL) SHALL weiterhin ihren bestehenden Inhalts-Filter behalten — die Sichtbarkeitsregel ist eine ZUSÄTZLICHE Whitelist-Bedingung.

#### Scenario: Push an Nicht-Berechtigte unterbleibt

- **WHEN** ein Event für Team A geändert wird und ein registrierter Push-User U nur in Team C ist
- **THEN** erhält U keine Push zu diesem Event

#### Scenario: Trainer erhält Push trotz fehlender Team-Mitgliedschaft

- **WHEN** ein Event für Team A geändert wird und Trainer T weder in A noch in einem anderen Team spielt
- **THEN** erhält T weiterhin die organisatorische Push (Funktions-Bypass)

