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
- **Spieler-Sicht (eigenes Mitglied)** als Tab in der eigenen Profil-Seite `/profil` (oder `/profil/anwesenheit`): zeigt für das eigene Mitglied die drei Zähler + Quote für Trainings und Spiele getrennt, plus eine tabellarische Liste aller Trainings und aller Spiele im Saisonzeitraum mit Datum, Titel, Status und Begründung.
- **Eltern-Sicht (verlinktes Kind)** als Tab auf der jeweiligen Kind-Detailseite `/profil/kind/:memberId`: dieselbe Statistik für genau dieses Kind. Die Anwesenheit eines Kindes liegt bewusst **auf dessen Kind-Seite**, nicht aggregiert im Eltern-Profil.

Die **Spieler-/Eltern-Sicht** SHALL nur Mitglieder mit der Vereinsfunktion `spieler` berücksichtigen:

- Der Tab „Anwesenheit" in `/profil` SHALL sichtbar sein, genau dann wenn `own_member.club_functions` `spieler` enthält. Andernfalls SHALL der Tab nicht in der Tab-Liste erscheinen. Der Tab zeigt ausschließlich die Statistik des eigenen Mitglieds (`ProfilAnwesenheitContent` mit `forcedMemberId=own_member.id`, keine Auswahl-Buttons).
- Der Tab „Anwesenheit" auf `/profil/kind/:memberId` SHALL sichtbar sein, genau dann wenn `member.club_functions` des Kindes `spieler` enthält, und die Statistik dieses Kindes zeigen (`ProfilAnwesenheitContent` mit `forcedMemberId=member.id`).
- Die eigenständige Seite `/profil/anwesenheit` behält für Nutzer mit mehreren eigenen Spieler-Bezügen die Auswahl-Buttons in `ProfilAnwesenheitContent`: `own_member` nur einschließen, wenn dessen `club_functions` `spieler` enthält; ein `children[i]` nur, wenn dessen `club_functions` `spieler` enthält. Default-`selectedId` ist das erste Mitglied dieser gefilterten Liste (Priorität: eigenes Mitglied vor Kindern).
- Der Trainer-Drilldown-Aufruf `/profil/anwesenheit?member=X` (Parameter `forcedMemberId` an `ProfilAnwesenheitContent`) SHALL den Spieler-Filter absichtlich umgehen — der aufrufende Nutzer (Trainer/SL) muss nicht selbst die Funktion `spieler` haben, um die Detailstatistik eines Spielers seines Kaders zu sehen.

Beide Sichten SHALL auf SSE-Event `attendance-changed` neu laden.

#### Scenario: Trainer sieht offene-Erfassungen-Banner

- **WHEN** ein Trainer `/team/:id/anwesenheit` öffnet und `GET /api/teams/{id}/attendance-open` mindestens einen Eintrag liefert
- **THEN** zeigt die Seite oben einen Banner "N offene Erfassungen" mit Link zur Detail-Liste

#### Scenario: Stammkader und erweiterter Kader sind visuell getrennt

- **WHEN** ein Team sowohl Stammkader- als auch erweiterte Kader-Mitglieder hat
- **THEN** zeigt die Trainer-Sicht zwei separate Tabellenblöcke mit eigenen Durchschnittszeilen

#### Scenario: Elternteil öffnet die Anwesenheit eines Spieler-Kindes auf dessen Kind-Seite

- **WHEN** ein Elternteil `/profil/kind/:memberId` eines Kindes mit `club_functions` `spieler` öffnet und den Tab „Anwesenheit" wählt
- **THEN** ist der Tab vorhanden und zeigt die Statistik genau dieses Kindes (ohne Auswahl-Buttons); der Anwesenheit-Tab im eigenen `/profil` des Elternteils bleibt davon unberührt

#### Scenario: Nicht-Spieler-Kind hat keinen Anwesenheit-Tab auf der Kind-Seite

- **WHEN** ein Elternteil `/profil/kind/:memberId` eines Kindes ohne `spieler` in `club_functions` öffnet
- **THEN** enthält die Tab-Liste der Kind-Seite kein „Anwesenheit"

#### Scenario: Elternteil mit mehreren Kindern wechselt das Kind (Standalone-Seite)

- **WHEN** ein Elternteil mit mehreren verlinkten Spieler-Kindern die Standalone-Seite `/profil/anwesenheit` öffnet und ein anderes Kind in der Kind-Auswahl wählt
- **THEN** lädt die Seite die Statistik für die neue `member_id` und ersetzt die Termin-Liste entsprechend

#### Scenario: Live-Update nach Erfassung

- **WHEN** ein Trainer auf der Trainer-Sicht ist und ein anderer Trainer im selben Browser-Cluster `POST /api/games/{id}/attendances` aufruft
- **THEN** sendet der Hub `attendance-changed` und die Seite lädt die Statistik automatisch neu

#### Scenario: Nutzer ohne eigene Spieler-Funktion sieht keinen Anwesenheit-Tab im eigenen Profil

- **WHEN** ein Nutzer, dessen eigenes Mitglied nur `trainer` (oder andere Nicht-Spieler-Funktionen) in `club_functions` hat, `/profil` öffnet
- **THEN** enthält die Tab-Liste kein „Anwesenheit" — unabhängig davon, ob Spieler-Kinder verknüpft sind (deren Anwesenheit liegt auf der jeweiligen Kind-Seite)

#### Scenario: Elternteil-Trainer sieht die Kind-Anwesenheit auf der Kind-Seite, nicht im eigenen Profil

- **WHEN** ein Nutzer mit `own_member.club_functions=[trainer]` und einem verlinkten Kind mit `club_functions=[spieler]` sein eigenes `/profil` öffnet
- **THEN** enthält die Tab-Liste des eigenen Profils kein „Anwesenheit"; die Anwesenheit des Kindes ist stattdessen als Tab auf `/profil/kind/:memberId` erreichbar

#### Scenario: Trainer-Drilldown funktioniert ohne eigene Spieler-Funktion

- **WHEN** ein Trainer über die Team-Sicht per `openMember`-Klick auf `/profil/anwesenheit?member=42` navigiert, obwohl sein eigenes Mitglied nicht `spieler` in `club_functions` führt
- **THEN** rendert die Seite die Statistik für Mitglied 42 direkt (kein 403, keine leere Auswahl)

### Requirement: Serien-Abmeldung schließt Session×Mitglied aus der Bezugsmenge aus

Zusätzlich zur Drei-Säulen-Klassifikation SHALL das System eine Trainings-Session für ein Mitglied vollständig aus present/missed/excused (und damit aus dem Nenner) ausschließen, wenn für dieses Mitglied und die Serie der Session eine greifende Serien-Abmeldung (`serien-abmeldung`-Ableitung) existiert. Der Ausschluss SHALL Vorrang vor der Kategorie ENTSCHULDIGT haben: liegt gleichzeitig eine `declined`-Response mit `absence_id` vor, dominiert der Ausschluss. In der Mitglieds-Detail-Termin-Liste (`GET /api/members/{id}/attendance-stats`) SHALL eine solche Session mit der Kategorie `unavailable` (nullable `reason`) erscheinen und in keiner Zähler-Spalte auftauchen.

#### Scenario: Abgemeldete Session zählt in keiner Säule

- **WHEN** ein Mitglied für eine Trainings-Session eine greifende Serien-Abmeldung hat und die Session im Saisonzeitraum liegt
- **THEN** wird diese Session weder als `training_present` noch `training_missed` noch `training_excused` gezählt

#### Scenario: Ausschluss dominiert eine parallele entschuldigte Absage

- **WHEN** für dieselbe Session sowohl eine greifende Serien-Abmeldung als auch eine `declined`-Response mit gesetzter `absence_id` existiert
- **THEN** wird die Session ausgeschlossen (nicht als `training_excused` gezählt)

#### Scenario: Detail-Liste kennzeichnet die Session als unavailable

- **WHEN** ein Trainer oder Spieler `GET /api/members/{id}/attendance-stats` abruft und eine Session der Serie von einer Abmeldung betroffen ist
- **THEN** enthält die `events`-Liste diesen Termin mit `category: "unavailable"` und dem `reason` der Abmeldung, ohne Beitrag zu einer Zähler-Spalte

#### Scenario: Team-Aggregat verwendet Pro-Spieler-Nenner

- **WHEN** in einem Team einzelne Spieler für bestimmte Serien abgemeldet sind
- **THEN** ist der Nenner jedes Spielers die Summe seiner eigenen present/missed/excused-Termine, und die ausgewiesene Team-Quote ist der Durchschnitt über die Pro-Spieler-Quoten (kein einheitlicher Team-Bruch)

#### Scenario: Nach Löschen der Abmeldung zählt die Session wieder

- **WHEN** eine Abmeldung entfernt wurde und danach die Statistik erneut geladen wird
- **THEN** werden die zuvor ausgeschlossenen Sessions wieder gemäß Drei-Säulen-Klassifikation gezählt
