## MODIFIED Requirements

### Requirement: Offene Mitfahr-Gesuche der eigenen Teams im Dashboard anzeigen

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `carpoolingOpenGroups` die offenen Mitfahr-Gesuche zu den kommenden Spielen anzeigen — sowohl der eigenen Teams als auch **teamübergreifend** zu kolozierten Spielen.

Ausgangspunkt sind die nächsten max. 3 künftigen Spiele der eigenen Teams in der aktiven Saison (alle `event_type`). Zusätzlich SHALL das System Spiele **anderer** Teams einbeziehen, die mit einem dieser Anker-Spiele denselben `date` UND dasselbe `venue_id` teilen, sofern `venue_id` gesetzt ist.

Die Gruppierung SHALL nach (Tag, Venue) erfolgen: Eine Gruppe trägt `date` und `venue.name` und listet alle offenen Gesuche aller dort/dann stattfindenden Spiele, teamübergreifend gemischt. Jeder Gesuch-Eintrag SHALL seinen Spiel-/Team-Kontext mitführen. Spiele ohne `venue_id` bilden eine Gruppe pro Spiel (Label = Gegner) und matchen nicht über Teamgrenzen.

Ein Gesuch (`mitfahrgelegenheiten.typ='suche'`) gilt als **offen**, solange darauf **keine** `mitfahrt_paarungen` mit `status='confirmed'` existiert. Eine nur `pending`-Paarung zählt weiterhin als offen. `carpoolingConfirmed` bleibt unverändert.

#### Scenario: Offenes Gesuch am eigenen kommenden Spiel

- **WHEN** an einem künftigen Spiel eines Teams des Nutzers ein `suche`-Eintrag ohne `confirmed`-Paarung existiert
- **THEN** enthält `carpoolingOpenGroups` eine Gruppe (Tag, Venue) mit diesem Gesuch

#### Scenario: Fremdteam-Gesuch bei gleichem Tag und Ort

- **WHEN** ein Fremdteam-Spiel denselben `date` und dasselbe `venue_id` (nicht NULL) wie ein Anker-Spiel des Nutzers hat und dort ein offenes Gesuch existiert
- **THEN** erscheint dieses Gesuch in derselben (Tag, Venue)-Gruppe — mit Spiel-/Team-Kontext

#### Scenario: Fremdteam-Gesuch bei abweichendem Ort/Tag

- **WHEN** ein Fremdteam-Spiel einen anderen `date` oder ein anderes `venue_id` als alle Anker-Spiele hat
- **THEN** erscheint dessen Gesuch NICHT in `carpoolingOpenGroups`

#### Scenario: Fehlendes Venue verhindert Cross-Team-Match

- **WHEN** das Anker-Spiel oder das potenzielle Partner-Spiel `venue_id IS NULL` hat
- **THEN** erfolgt KEIN Cross-Team-Match
- **AND** das eigene Spiel erscheint als eigene Gruppe pro Spiel (Fallback)

#### Scenario: Pool merged Gesuche mehrerer Spiele

- **WHEN** zwei Spiele dasselbe (Tag, Venue) teilen und beide offene Gesuche haben
- **THEN** erscheinen alle Gesuche unter EINER (Tag, Venue)-Gruppe

#### Scenario: Gesuch mit bestätigter Paarung ist nicht offen

- **WHEN** ein `suche`-Eintrag eine Paarung mit `status='confirmed'` hat
- **THEN** erscheint dieses Gesuch NICHT in `carpoolingOpenGroups`
- **AND** die bestätigte Paarung erscheint weiterhin in `carpoolingConfirmed`

#### Scenario: Gesuch mit nur ausstehender Paarung bleibt offen

- **WHEN** ein `suche`-Eintrag ausschließlich Paarungen mit `status='pending'` hat
- **THEN** erscheint dieses Gesuch weiterhin in `carpoolingOpenGroups`

#### Scenario: Keine offenen Gesuche

- **WHEN** zu den relevanten Spielen (eigene + kolozierte) keine offenen Gesuche existieren
- **THEN** ist `carpoolingOpenGroups` ein leeres Array
