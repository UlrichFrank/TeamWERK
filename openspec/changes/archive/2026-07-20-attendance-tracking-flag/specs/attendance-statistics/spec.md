## MODIFIED Requirements

### Requirement: Drei-Säulen-Klassifikation pro Termin und Mitglied

Die Statistik SHALL für jede Kombination aus Termin (Trainings-Session oder Spiel) und Kader-Mitglied genau eine der vier Kategorien ermitteln:

- **ANWESEND** wenn `attendance.present = 1` UND die Session/das Spiel hat `attendance_tracked = 1`
- **FEHLT** wenn `attendance.present = 0` UND die Session/das Spiel hat `attendance_tracked = 1`
- **ENTSCHULDIGT** wenn keine wirksame `attendance`-Row existiert (entweder keine Row oder `attendance_tracked = 0`) UND `response.status = 'declined'` UND `response.absence_id IS NOT NULL`
- **IGNORIERT** in allen anderen Fällen

Für Sessions/Spiele mit `attendance_tracked = 0` SHALL das System vorhandene `attendance`-Rows behandeln, als existierten sie nicht. Cancelled Trainings (`training_sessions.status='cancelled'`) SHALL aus der Bezugsmenge entfernt werden. Spiele haben in TeamWERK keinen Cancellation-Status — abgesagte Spiele werden komplett gelöscht und tauchen folglich nicht mehr in der Bezugsmenge auf.

#### Scenario: Anwesenheit dominiert auto-decline

- **WHEN** ein Mitglied für eine Trainings-Session sowohl `attendance.present = 1` als auch eine `response`-Zeile mit `status='declined'` und gesetzter `absence_id` hat und die Session `attendance_tracked=1` hat
- **THEN** wird das Mitglied als ANWESEND gezählt (nicht als ENTSCHULDIGT)

#### Scenario: Datenloch wird ignoriert

- **WHEN** ein vergangener Termin keine `attendance`-Row und keine `declined`-Response mit `absence_id` hat
- **THEN** zählt der Termin für dieses Mitglied in keiner der drei Säulen

#### Scenario: Cancelled Training nicht gezählt

- **WHEN** eine Trainings-Session `status='cancelled'` hat
- **THEN** taucht der Termin in keinem `count` der drei Säulen auf

#### Scenario: attendance_tracked=0 blendet Rows aus

- **WHEN** eine Trainings-Session `attendance_tracked=0` hat, aber `training_attendances`-Rows mit `present=0` für Mitglied M existieren
- **THEN** wird der Termin für M als IGNORIERT klassifiziert (nicht als FEHLT)

#### Scenario: attendance_tracked=0 lässt entschuldigte Absage zählen

- **WHEN** eine Trainings-Session `attendance_tracked=0` hat, gleichzeitig aber eine `declined`-Response mit gesetzter `absence_id` für Mitglied M existiert
- **THEN** wird der Termin für M als ENTSCHULDIGT gezählt (die Row wird durch den Filter unsichtbar, die Response bleibt maßgeblich)

### Requirement: Offene Erfassungen pro Team

Das System SHALL via `GET /api/teams/{id}/attendance-open` eine Liste der vergangenen Termine (`date < today()`) der aktiven Saison liefern, deren `attendance_tracked = 0` ist. Trainings mit `status='cancelled'` SHALL ausgeschlossen werden; abgesagte Spiele sind in TeamWERK gelöscht und tauchen daher nicht auf. Pro Termin: `event_type` (`training`/`game`), `event_id`, `date`, `title`. Authz: Trainer der zugehörigen Teams, sportliche Leitung, Admin.

#### Scenario: Vergangenes Training ohne Erfassung erscheint

- **WHEN** ein Trainer `GET /api/teams/{id}/attendance-open` aufruft und eine vergangene, aktive Trainings-Session des Teams `attendance_tracked=0` hat
- **THEN** ist diese Session in der Antwort enthalten

#### Scenario: Vergangenes Spiel mit aktiver Erfassung verschwindet

- **WHEN** für ein vergangenes Spiel des Teams `attendance_tracked=1` gesetzt ist
- **THEN** ist das Spiel **nicht** in der Antwort enthalten

#### Scenario: Reset lässt Termin wieder erscheinen

- **WHEN** ein Trainer die Reset-Route für eine zuvor erfasste Session/ein zuvor erfasstes Spiel aufruft
- **THEN** taucht der Termin bei der nächsten `GET /api/teams/{id}/attendance-open`-Antwort wieder auf

#### Scenario: Cancelled Training nicht enthalten

- **WHEN** eine vergangene Trainings-Session `status='cancelled'` hat
- **THEN** erscheint sie nicht in der Antwort, unabhängig vom Flag

#### Scenario: Zukünftiger Termin nicht enthalten

- **WHEN** ein Termin des Teams in der Zukunft liegt
- **THEN** erscheint er nicht in der Antwort

#### Scenario: Spieler ohne Trainer-Funktion abgewiesen

- **WHEN** ein Spieler `GET /api/teams/{id}/attendance-open` aufruft
- **THEN** antwortet das System mit HTTP 403

## ADDED Requirements

### Requirement: Backfill setzt `attendance_tracked` für Bestandsdaten

Die Migration, die `attendance_tracked` einführt, SHALL für Bestands-Sessions/-Spiele mit mindestens einer `training_attendances`- bzw. `game_attendances`-Row `attendance_tracked=1` setzen. Für Sessions/Spiele ohne solche Row bleibt der Default `0`. Damit ist das UI-Verhalten für alle historischen Termine identisch zum Verhalten vor der Migration (sichtbare Statistik-Zeilen bleiben sichtbar, „offen zu erfassen"-Liste bleibt identisch).

#### Scenario: Bestands-Session mit Rows wird tracked

- **WHEN** vor Migration eine Trainings-Session mindestens eine `training_attendances`-Row hat
- **THEN** ist nach Migration `attendance_tracked=1`

#### Scenario: Bestands-Session ohne Rows bleibt untracked

- **WHEN** vor Migration eine Trainings-Session keine `training_attendances`-Row hat
- **THEN** ist nach Migration `attendance_tracked=0` (Default)
