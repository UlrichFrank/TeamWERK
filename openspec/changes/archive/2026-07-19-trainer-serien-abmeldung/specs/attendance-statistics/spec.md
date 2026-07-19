## ADDED Requirements

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
