### Requirement: Trainer kann Aufstellung pro Spiel setzen

Das System SHALL einen Endpoint `POST /api/games/{id}/lineup` bereitstellen, der eine Liste von `member_id`-Werten als Aufstellung speichert (bulk upsert + delete-diff). Nur Trainer und Admins dürfen schreiben.

#### Scenario: Trainer setzt Aufstellung

- **WHEN** ein Trainer `POST /api/games/{id}/lineup` mit `{"member_ids": [1, 2, 3]}` sendet
- **THEN** werden genau diese Mitglieder als Aufstellung gespeichert; nicht enthaltene Einträge werden gelöscht

#### Scenario: Spieler kann Aufstellung nicht setzen

- **WHEN** ein Spieler `POST /api/games/{id}/lineup` sendet
- **THEN** antwortet das System mit HTTP 403

### Requirement: Aufstellung ist per Participants-Endpoint abrufbar

Das System SHALL `GET /api/games/{id}/participants` bereitstellen, der alle regulären und erweiterten Kader-Mitglieder des Teams zurückgibt, jeweils mit RSVP-Status (`rsvp_status`, nullable) und Lineup-Status (`in_lineup: bool`).

#### Scenario: Participant-Liste enthält reguläre und erweiterte Mitglieder

- **WHEN** ein Trainer `GET /api/games/{id}/participants` abruft
- **THEN** enthält die Antwort sowohl `kader_members` (mit RSVP-Status) als auch `kader_extended_members` (mit `rsvp_status: null`) des Teams

#### Scenario: Lineup-Status ist korrekt gesetzt

- **WHEN** ein Mitglied in `game_lineup` für dieses Spiel eingetragen ist
- **THEN** enthält sein Eintrag in der Participants-Antwort `in_lineup: true`

### Requirement: Spieldetail zeigt Aufstellungs-Spalte

Das System SHALL auf `/termine/spiel/{id}` in der Teilnahme-Tabelle eine Spalte „Aufstellung" anzeigen. Trainer sehen Checkboxen (editierbar). Spieler und Eltern sehen read-only-Indikatoren.

#### Scenario: Trainer kann Aufstellung über Checkbox setzen

- **WHEN** ein Trainer die Checkbox eines Mitglieds in der Aufstellungs-Spalte aktiviert
- **THEN** wird dieses Mitglied in die Aufstellung aufgenommen (optimistic update + API-Call)

#### Scenario: Spieler sieht Aufstellung read-only

- **WHEN** ein Spieler die Spieldetail-Seite öffnet
- **THEN** sieht er in der Aufstellungs-Spalte Häkchen (nominiert) oder Striche (nicht nominiert), ohne editierbare Checkboxen

#### Scenario: Erweitertes Kader-Mitglied erscheint in Aufstellungs-Tabelle

- **WHEN** ein erweitertes Kader-Mitglied für das Team eingetragen ist
- **THEN** erscheint es in der Teilnahme-Tabelle ohne RSVP-Status, aber mit Aufstellungs-Checkbox (Trainer) bzw. Indikator (andere)
