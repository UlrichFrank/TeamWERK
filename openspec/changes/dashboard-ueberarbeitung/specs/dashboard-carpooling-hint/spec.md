## MODIFIED Requirements

### Requirement: Bestätigte Fahrgemeinschaften im Dashboard

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `carpoolingHint` ein Array der nächsten max. 3 Auswärtsspiele zurückgeben, für die der User bestätigte Paarungen hat.

Pro Auswärtsspiel:
- `gameId`, `date`, `opponent`
- `paarungen`: nur Paarungen mit `status='confirmed'`, an denen der User beteiligt ist, mit Name der Gegenseite

Nicht mehr enthalten: `bieteCount`, `sucheCount`, `myEntry`, `openEntries`.

`carpoolingHint` ist ein Array (nicht ein einzelnes Objekt). Leeres Array wenn keine bestätigten Paarungen in den nächsten 3 Auswärtsspielen.

#### Scenario: User hat bestätigte Paarung für nächstes Auswärtsspiel

- **WHEN** der User eine `confirmed`-Paarung für das nächste Auswärtsspiel hat
- **THEN** enthält `carpoolingHint[0]` dieses Spiel mit der bestätigten Paarung

#### Scenario: Keine bestätigten Paarungen

- **WHEN** der User in keinem der nächsten 3 Auswärtsspiele eine `confirmed`-Paarung hat
- **THEN** ist `carpoolingHint` ein leeres Array

#### Scenario: Offene Einträge werden nicht angezeigt

- **WHEN** andere User offene Angebote oder Gesuche für ein Auswärtsspiel haben
- **THEN** erscheinen diese NICHT in `carpoolingHint` (nur auf der Mitfahrgelegenheiten-Seite sichtbar)

#### Scenario: Bis zu 3 Auswärtsspiele

- **WHEN** der User bestätigte Paarungen für 4 oder mehr Auswärtsspiele hat
- **THEN** gibt `carpoolingHint` nur die nächsten 3 zurück

#### Scenario: Kein Auswärtsspiel in der Zukunft

- **WHEN** keine Auswärtsspiele für die Teams des Users in der aktiven Saison ab heute existieren
- **THEN** ist `carpoolingHint` ein leeres Array
