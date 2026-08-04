## MODIFIED Requirements

### Requirement: Drei-Säulen-Klassifikation pro Termin und Mitglied

Die Statistik SHALL für jede Kombination aus Termin (Trainings-Session oder Spiel) und Kader-Mitglied genau eine der vier Kategorien ermitteln:

- **ANWESEND** wenn `attendance.present = 1` UND die Session/das Spiel hat `attendance_tracked = 1`
- **FEHLT** wenn `attendance.present = 0` UND die Session/das Spiel hat `attendance_tracked = 1`
- **ENTSCHULDIGT** wenn keine wirksame `attendance`-Row existiert (entweder keine Row oder `attendance_tracked = 0`) UND `response.status = 'declined'` — **unabhängig davon, ob `response.absence_id` gesetzt ist**. Jede Absage (automatisch durch eine erfasste Abwesenheit oder manuell durch das Mitglied selbst) zählt gleichermaßen als entschuldigt.
- **IGNORIERT** in allen anderen Fällen

Für Sessions/Spiele mit `attendance_tracked = 0` SHALL das System vorhandene `attendance`-Rows behandeln, als existierten sie nicht. Cancelled Trainings (`training_sessions.status='cancelled'`) SHALL aus der Bezugsmenge entfernt werden. Spiele haben in TeamWERK keinen Cancellation-Status — abgesagte Spiele werden komplett gelöscht und tauchen folglich nicht mehr in der Bezugsmenge auf.

#### Scenario: Anwesenheit dominiert auto-decline

- **WHEN** ein Mitglied für eine Trainings-Session sowohl `attendance.present = 1` als auch eine `response`-Zeile mit `status='declined'` (unabhängig davon, ob `absence_id` gesetzt ist) hat und die Session `attendance_tracked=1` hat
- **THEN** wird das Mitglied als ANWESEND gezählt (nicht als ENTSCHULDIGT)

#### Scenario: Manuelle Absage ohne hinterlegte Abwesenheit zählt als entschuldigt

- **WHEN** ein Mitglied für einen vergangenen Termin `status='declined'` hat, ohne dass eine Abwesenheit hinterlegt ist (`response.absence_id IS NULL`), und keine wirksame `attendance`-Row existiert
- **THEN** wird der Termin für dieses Mitglied als ENTSCHULDIGT gezählt

#### Scenario: Datenloch wird ignoriert

- **WHEN** ein vergangener Termin für ein Mitglied weder eine wirksame `attendance`-Row noch eine `declined`-Response hat (z.B. keine Rückmeldung oder `confirmed`/`maybe` ohne Erfassung)
- **THEN** zählt der Termin für dieses Mitglied in keiner der drei Säulen

#### Scenario: Cancelled Training nicht gezählt

- **WHEN** eine Trainings-Session `status='cancelled'` hat
- **THEN** taucht der Termin in keinem `count` der drei Säulen auf

#### Scenario: attendance_tracked=0 blendet Rows aus

- **WHEN** eine Trainings-Session `attendance_tracked=0` hat, aber `training_attendances`-Rows mit `present=0` für Mitglied M existieren
- **THEN** wird der Termin für M als IGNORIERT klassifiziert (nicht als FEHLT)

#### Scenario: attendance_tracked=0 lässt entschuldigte Absage zählen

- **WHEN** eine Trainings-Session `attendance_tracked=0` hat, gleichzeitig aber eine `declined`-Response (mit oder ohne `absence_id`) für Mitglied M existiert
- **THEN** wird der Termin für M als ENTSCHULDIGT gezählt (die Row wird durch den Filter unsichtbar, die Response bleibt maßgeblich)

### Requirement: Serien-Abmeldung schließt Session×Mitglied aus der Bezugsmenge aus

Zusätzlich zur Drei-Säulen-Klassifikation SHALL das System eine Trainings-Session für ein Mitglied vollständig aus present/missed/excused (und damit aus dem Nenner) ausschließen, wenn für dieses Mitglied und die Serie der Session eine greifende Serien-Abmeldung (`serien-abmeldung`-Ableitung) existiert. Der Ausschluss SHALL Vorrang vor der Kategorie ENTSCHULDIGT haben: liegt gleichzeitig eine `declined`-Response vor (unabhängig davon, ob `absence_id` gesetzt ist), dominiert der Ausschluss. In der Mitglieds-Detail-Termin-Liste (`GET /api/members/{id}/attendance-stats`) SHALL eine solche Session mit der Kategorie `unavailable` (nullable `reason`) erscheinen und in keiner Zähler-Spalte auftauchen. Die Kategorie `unavailable` bleibt eigenständig und wird nicht mit `excused`/ENTSCHULDIGT zusammengelegt — „dauerhaft abmelden" durch einen Trainer ist fachlich von einer individuellen Absage zu unterscheiden.

#### Scenario: Abgemeldete Session zählt in keiner Säule

- **WHEN** ein Mitglied für eine Trainings-Session eine greifende Serien-Abmeldung hat und die Session im Saisonzeitraum liegt
- **THEN** wird diese Session weder als `training_present` noch `training_missed` noch `training_excused` gezählt

#### Scenario: Ausschluss dominiert eine parallele entschuldigte Absage

- **WHEN** für dieselbe Session sowohl eine greifende Serien-Abmeldung als auch eine `declined`-Response (mit oder ohne `absence_id`) existiert
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
