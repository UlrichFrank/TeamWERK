### Requirement: Spieler kann zu einem Spiel RSVP abgeben

Das System SHALL es Spielern und Eltern ermöglichen, zu jedem Spiel ihres Teams
eine Rückmeldung (confirmed / declined / maybe) mit optionalem Grund abzugeben.

#### Scenario: Spieler sagt zu
- **WHEN** ein User mit Rolle `spieler` `POST /api/games/{id}/respond` mit `{"status": "confirmed"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist der Eintrag in `game_responses` mit `status = 'confirmed'` gespeichert

#### Scenario: Spieler sagt ab mit Grund
- **WHEN** ein User `POST /api/games/{id}/respond` mit `{"status": "declined", "reason": "Urlaub"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist der Grund in `game_responses.reason` gespeichert

#### Scenario: Elternteil sagt für Kind ab
- **WHEN** ein User mit Rolle `elternteil` `POST /api/games/{id}/respond` mit `{"member_id": 42, "status": "declined"}` aufruft
- **AND** member_id 42 ist via `family_links` mit dem User verknüpft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Elternteil ohne Verknüpfung wird abgelehnt
- **WHEN** ein User mit Rolle `elternteil` `POST /api/games/{id}/respond` mit einer `member_id` aufruft, die nicht zu seinen Kindern gehört
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: RSVP-Update überschreibt alten Eintrag
- **WHEN** ein User für dasselbe Spiel erneut `POST /api/games/{id}/respond` aufruft
- **THEN** wird der bestehende Eintrag via UPSERT aktualisiert

#### Scenario: Ungültiger Status wird abgelehnt
- **WHEN** `POST /api/games/{id}/respond` mit einem Status außerhalb von `confirmed/declined/maybe` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: Kein Default-Status bei Spielen

Das System SHALL für Spiele keinen vorausgewählten RSVP-Status setzen.
Spieler müssen aktiv eine Auswahl treffen.

#### Scenario: Neues Spiel hat keinen RSVP-Status
- **WHEN** ein User die `/termine`-Seite aufruft und ein Spiel noch keine Rückmeldung hat
- **THEN** sind alle drei RSVP-Buttons (Zusagen/Vielleicht/Absagen) inaktiv (kein Button ist hervorgehoben)

---

### Requirement: User-gefilterte Spielliste mit RSVP-Daten

`GET /api/games/my` SHALL Spiele des eigenen Teams zurückgeben, inklusive
`my_rsvp`, `confirmed_count`, `declined_count`, `maybe_count` pro Spiel.

#### Scenario: Spieler sieht nur eigene Teamspiele
- **WHEN** ein User mit Rolle `spieler` `GET /api/games/my` aufruft
- **THEN** enthält die Antwort nur Spiele, bei denen sein Team über `game_teams` beteiligt ist

#### Scenario: RSVP-Counts in der Liste
- **WHEN** `GET /api/games/my` aufgerufen wird
- **THEN** enthält jedes Spiel-Objekt die Felder `confirmed_count`, `declined_count`, `maybe_count` und `my_rsvp`

#### Scenario: my_rsvp ist null wenn keine Antwort
- **WHEN** ein User noch keine Rückmeldung für ein Spiel abgegeben hat
- **THEN** ist `my_rsvp` im Response `null`

---

### Requirement: Trainer sieht Rückmeldungs-Übersicht pro Spiel

`GET /api/games/{id}/responses` SHALL für Trainer und Admins alle Rückmeldungen
zu einem Spiel zurückgeben (member_name, status, reason).

#### Scenario: Trainer ruft Übersicht ab
- **WHEN** ein User mit Rolle `trainer` oder `admin` `GET /api/games/{id}/responses` aufruft
- **THEN** antwortet der Server mit HTTP 200 und einer Liste aller Rückmeldungen

#### Scenario: Spieler kann Übersicht nicht abrufen
- **WHEN** ein User mit Rolle `spieler` `GET /api/games/{id}/responses` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Grund nur für eigene oder eigene Kinder sichtbar
- **WHEN** ein User mit Rolle `elternteil` die Detailseite aufruft
- **THEN** sind Gründe nur für seine eigenen Kinder sichtbar, nicht für andere Mitglieder
