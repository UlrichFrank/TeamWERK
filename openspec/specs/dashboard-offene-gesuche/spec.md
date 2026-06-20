## ADDED Requirements

### Requirement: Offene Mitfahr-Gesuche der eigenen Teams im Dashboard anzeigen

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `carpoolingOpenGroups` die offenen Mitfahr-Gesuche zu den kommenden Spielen der Teams des eingeloggten Nutzers anzeigen.

Eine Gruppe entspricht einem Spiel und enthält `date`, eine Bezeichnung (Gegner/Event) und die Liste der offenen Gesuche (Name des Suchenden, Anzahl benötigter Plätze, optional Treffpunkt). Berücksichtigt werden die nächsten max. 3 künftigen Spiele der eigenen Teams in der aktiven Saison, unabhängig vom `event_type` (Heim, Auswärts, generisch).

Ein Gesuch (`mitfahrgelegenheiten.typ='suche'`) gilt als **offen**, solange darauf **keine** `mitfahrt_paarungen` mit `status='confirmed'` existiert. Eine nur `pending`-Paarung zählt weiterhin als offen.

`carpoolingConfirmed` (bestätigte Paarungen) bleibt unverändert und unabhängig von `carpoolingOpenGroups`.

#### Scenario: Offenes Gesuch am eigenen kommenden Spiel

- **WHEN** an einem künftigen Spiel eines Teams des Nutzers ein `suche`-Eintrag ohne `confirmed`-Paarung existiert
- **THEN** enthält `carpoolingOpenGroups` eine Gruppe für dieses Spiel mit dem Gesuch

#### Scenario: Gesuch mit bestätigter Paarung ist nicht offen

- **WHEN** ein `suche`-Eintrag eine Paarung mit `status='confirmed'` hat
- **THEN** erscheint dieses Gesuch NICHT in `carpoolingOpenGroups`
- **AND** die bestätigte Paarung erscheint weiterhin in `carpoolingConfirmed`

#### Scenario: Gesuch mit nur ausstehender Paarung bleibt offen

- **WHEN** ein `suche`-Eintrag ausschließlich Paarungen mit `status='pending'` hat
- **THEN** erscheint dieses Gesuch weiterhin in `carpoolingOpenGroups`

#### Scenario: Gesuch eines fremden Teams wird nicht gezeigt

- **WHEN** ein offenes Gesuch an einem Spiel existiert, das keinem Team des Nutzers gehört
- **THEN** erscheint es NICHT in `carpoolingOpenGroups` (teamübergreifende Anzeige ist nicht Teil dieser Capability)

#### Scenario: Keine offenen Gesuche

- **WHEN** zu den nächsten Spielen der eigenen Teams keine offenen Gesuche existieren
- **THEN** ist `carpoolingOpenGroups` ein leeres Array
