# attendance-statistics Specification

## Purpose
TBD - created by syncing change anwesenheits-statistik. Update Purpose after sync.

## Requirements

### Requirement: Drei-Säulen-Klassifikation pro Termin und Mitglied

Die Statistik SHALL für jede Kombination aus Termin (Trainings-Session oder Spiel) und Kader-Mitglied genau eine der vier Kategorien ermitteln:

- **ANWESEND** wenn `attendance.present = 1`
- **FEHLT** wenn `attendance.present = 0`
- **ENTSCHULDIGT** wenn keine `attendance`-Row existiert UND `response.status = 'declined'` UND `response.absence_id IS NOT NULL`
- **IGNORIERT** in allen anderen Fällen

Cancelled Trainings (`training_sessions.status='cancelled'`) SHALL aus der Bezugsmenge entfernt werden. Spiele haben in TeamWERK keinen Cancellation-Status — abgesagte Spiele werden komplett gelöscht und tauchen folglich nicht mehr in der Bezugsmenge auf.

#### Scenario: Anwesenheit dominiert auto-decline

- **WHEN** ein Mitglied für eine Trainings-Session sowohl `attendance.present = 1` als auch eine `response`-Zeile mit `status='declined'` und gesetzter `absence_id` hat
- **THEN** wird das Mitglied als ANWESEND gezählt (nicht als ENTSCHULDIGT)

#### Scenario: Datenloch wird ignoriert

- **WHEN** ein vergangener Termin keine `attendance`-Row und keine `declined`-Response mit `absence_id` hat
- **THEN** zählt der Termin für dieses Mitglied in keiner der drei Säulen

#### Scenario: Cancelled Training nicht gezählt

- **WHEN** eine Trainings-Session `status='cancelled'` hat
- **THEN** taucht der Termin in keinem `count` der drei Säulen auf

### Requirement: Team-Aggregat-Statistik

Das System SHALL via `GET /api/teams/{id}/attendance-stats?season=<id>` eine Aggregat-Statistik zurückgeben, die je Stammkader-Mitglied und je erweitertem Kader-Mitglied die sechs Zähler `training_present`, `training_excused`, `training_missed`, `game_present`, `game_excused`, `game_missed` enthält, getrennt in zwei Blöcke `regular_members` und `extended_members`, plus Team-Durchschnitte pro Block. Default-Saison ist die aktive Saison. Termine zählen nur, wenn ihr `date` zwischen `season.start_date` und heute (inkl.) liegt.

Authz: Nur Trainer der zugehörigen Teams (`kader_trainers`), Vereinsfunktion `sportliche_leitung` oder Admin.

#### Scenario: Trainer erhält Statistik seines Teams

- **WHEN** ein Trainer `GET /api/teams/{id}/attendance-stats` für ein Team seines Kaders ohne `season`-Parameter aufruft
- **THEN** erhält er HTTP 200 mit der Aggregat-Statistik der aktiven Saison

#### Scenario: Spieler in beiden Kadern wird nicht doppelt aufgeführt

- **WHEN** ein Mitglied sowohl in `kader_members` als auch in `kader_extended_members` desselben Teams ist
- **THEN** erscheint es im Block `regular_members` und nicht in `extended_members`

#### Scenario: Cancelled Trainings fließen nicht in die Aggregation ein

- **WHEN** eine Trainings-Session des Teams mit `status='cancelled'` im Saisonzeitraum liegt
- **THEN** spiegelt sich das in keinem der sechs Zähler eines Mitglieds wider

#### Scenario: Sportliche Leitung erhält jedes Team

- **WHEN** ein Mitglied mit Vereinsfunktion `sportliche_leitung` die Statistik eines beliebigen Teams abruft
- **THEN** erhält es HTTP 200

#### Scenario: Spieler ohne Trainer-Funktion abgewiesen

- **WHEN** ein Spieler ohne `trainer`/`sportliche_leitung`-Funktion `GET /api/teams/{id}/attendance-stats` aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Nicht-existentes Team

- **WHEN** ein berechtigter Nutzer eine `team_id` ohne Datenbank-Eintrag abfragt
- **THEN** antwortet das System mit HTTP 404

### Requirement: Mitglieds-Detailstatistik mit Termin-Liste

Das System SHALL via `GET /api/members/{id}/attendance-stats?season=<id>` die sechs Zähler des Mitglieds **plus** eine vollständige Termin-Liste (alle Trainings + alle Spiele im Saisonzeitraum, an denen das Mitglied über Kader oder erweiterten Kader teilnahmeberechtigt war) zurückgeben. Jeder Termineintrag enthält: `event_type` (`training` oder `game`), `event_id`, `date`, `title`, `category` (`present`, `missed`, `excused`, `unknown`, `cancelled`), `reason` (nullable).

Authz: Eigenes Mitglied (über User-Member-Verknüpfung), Elternteil mit `family_links`-Eintrag, Trainer der zugehörigen Teams, sportliche Leitung, Admin.

#### Scenario: Spieler ruft eigene Statistik ab

- **WHEN** ein eingeloggter Spieler `GET /api/members/{id}/attendance-stats` für sein eigenes Mitglied aufruft
- **THEN** erhält er HTTP 200 mit Zählern und Termin-Liste

#### Scenario: Elternteil ruft Statistik eines verlinkten Kindes ab

- **WHEN** ein Elternteil `GET /api/members/{id}/attendance-stats` für eine `member_id` aufruft, mit der er per `family_links` verbunden ist
- **THEN** erhält er HTTP 200

#### Scenario: Fremder Nutzer abgewiesen

- **WHEN** ein Spieler `GET /api/members/{id}/attendance-stats` für ein anderes, nicht verlinktes Mitglied aufruft und er weder Trainer noch sportliche Leitung noch Admin ist
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Termin-Liste umfasst alle Trainings und Spiele

- **WHEN** ein Trainer die Detailstatistik eines Stammkader-Spielers abruft
- **THEN** enthält `events` jeden nicht-cancelled Trainings-Termin und jedes nicht-cancelled Spiel der Teams im Saisonzeitraum, jeweils mit der korrekten `category`

#### Scenario: Cancelled Trainings als category=cancelled gelistet

- **WHEN** eine Trainings-Session im Saisonzeitraum `status='cancelled'` hat
- **THEN** erscheint sie in der Termin-Liste mit `category: "cancelled"` und zählt in keiner Zähler-Spalte

### Requirement: Offene Erfassungen pro Team

Das System SHALL via `GET /api/teams/{id}/attendance-open` eine Liste der vergangenen Termine (`date < today()`) der aktiven Saison liefern, die noch **keine** einzige `attendance`-Row haben. Trainings mit `status='cancelled'` SHALL ausgeschlossen werden; abgesagte Spiele sind in TeamWERK gelöscht und tauchen daher nicht auf. Pro Termin: `event_type` (`training`/`game`), `event_id`, `date`, `title`. Authz: Trainer der zugehörigen Teams, sportliche Leitung, Admin.

#### Scenario: Vergangenes Training ohne Erfassung erscheint

- **WHEN** ein Trainer `GET /api/teams/{id}/attendance-open` aufruft und eine vergangene, aktive Trainings-Session des Teams keine `training_attendances`-Row hat
- **THEN** ist diese Session in der Antwort enthalten

#### Scenario: Vergangenes Spiel mit mindestens einer Anwesenheit verschwindet

- **WHEN** für ein vergangenes Spiel des Teams bereits mindestens eine `game_attendances`-Row existiert
- **THEN** ist das Spiel **nicht** in der Antwort enthalten

#### Scenario: Cancelled Training nicht enthalten

- **WHEN** eine vergangene Trainings-Session `status='cancelled'` hat
- **THEN** erscheint sie nicht in der Antwort, unabhängig vom Vorhandensein einer `attendance`-Row

#### Scenario: Zukünftiger Termin nicht enthalten

- **WHEN** ein Termin des Teams in der Zukunft liegt
- **THEN** erscheint er nicht in der Antwort

#### Scenario: Spieler ohne Trainer-Funktion abgewiesen

- **WHEN** ein Spieler `GET /api/teams/{id}/attendance-open` aufruft
- **THEN** antwortet das System mit HTTP 403

### Requirement: Trainer- und Spieler-Sichten im Frontend

Das Frontend SHALL zwei Sichten bereitstellen:

- **Trainer-/SL-Sicht** unter `/team/:id/anwesenheit`: zeigt oben einen Banner mit der Anzahl offener Erfassungen (Link zur Detail-Liste), darunter eine Tabelle mit dem Stammkader (Spieler, drei Zähler + Quote je für Trainings und Spiele), darunter einen separat überschriebenen Block "Erweiterter Kader (N)" mit gleichem Layout und einer Team-Durchschnittszeile. Tabellen folgen den Projekt-Conventions (brand-Tokens, `lucide-react`-Icons, Mobile-Card-Layout, Touch-Targets ≥ 44px).
- **Spieler-/Eltern-Sicht** als Tab in der Profil-Seite (oder `/profil/anwesenheit`): zeigt für das eigene Mitglied (bzw. das ausgewählte Kind bei Eltern mit mehreren Kindern) die drei Zähler + Quote für Trainings und Spiele getrennt, plus eine tabellarische Liste aller Trainings und aller Spiele im Saisonzeitraum mit Datum, Titel, Status und Begründung.

Beide Sichten SHALL auf SSE-Event `attendance-changed` neu laden.

#### Scenario: Trainer sieht offene-Erfassungen-Banner

- **WHEN** ein Trainer `/team/:id/anwesenheit` öffnet und `GET /api/teams/{id}/attendance-open` mindestens einen Eintrag liefert
- **THEN** zeigt die Seite oben einen Banner "N offene Erfassungen" mit Link zur Detail-Liste

#### Scenario: Stammkader und erweiterter Kader sind visuell getrennt

- **WHEN** ein Team sowohl Stammkader- als auch erweiterte Kader-Mitglieder hat
- **THEN** zeigt die Trainer-Sicht zwei separate Tabellenblöcke mit eigenen Durchschnittszeilen

#### Scenario: Elternteil mit mehreren Kindern wechselt das Kind

- **WHEN** ein Elternteil mit mehreren verlinkten Kindern die Spieler-Sicht öffnet und ein anderes Kind in der Kind-Auswahl wählt
- **THEN** lädt die Seite die Statistik für die neue `member_id` und ersetzt die Termin-Liste entsprechend

#### Scenario: Live-Update nach Erfassung

- **WHEN** ein Trainer auf der Trainer-Sicht ist und ein anderer Trainer im selben Browser-Cluster `POST /api/games/{id}/attendances` aufruft
- **THEN** sendet der Hub `attendance-changed` und die Seite lädt die Statistik automatisch neu
