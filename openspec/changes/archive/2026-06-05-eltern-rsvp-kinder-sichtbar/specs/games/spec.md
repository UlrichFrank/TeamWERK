## ADDED Requirements

### Requirement: Spiel-Detail zeigt vollständige Kaderliste für alle authentifizierten User

`GET /api/games/{id}/responses` SHALL nicht nur Spieler zurückgeben, die bereits geantwortet haben, sondern alle Kader-Mitglieder aller zugeordneten Teams für die aktive Saison. User ohne Antwort erscheinen mit `status: null`.

#### Scenario: Spieler ruft Spiel-Detail ab

- **WHEN** ein User mit Rolle `spieler` `GET /api/games/{id}/responses` aufruft
- **THEN** antwortet der Server mit HTTP 200
- **THEN** enthält die Antwort alle Kader-Mitglieder des Teams, auch solche ohne RSVP

#### Scenario: Elternteil ruft Spiel-Detail ab

- **WHEN** ein User mit `is_parent = true` `GET /api/games/{id}/responses` aufruft
- **THEN** antwortet der Server mit HTTP 200
- **THEN** enthält die Antwort alle Kader-Mitglieder des Teams

---

### Requirement: Kommentar-Sichtbarkeit auf der Spiel-Detail-Seite

`GET /api/games/{id}/responses` SHALL Kommentare (`reason`) rollenabhängig zurückgeben:
- Trainer/Admin: alle Kommentare aller Spieler
- Spieler: nur der eigene Kommentar
- Elternteil: nur Kommentare der eigenen Kinder (via `family_links`)

#### Scenario: Trainer sieht alle Kommentare

- **WHEN** ein Trainer `GET /api/games/{id}/responses` aufruft
- **THEN** enthält jeder Eintrag mit vorhandenem Kommentar das Feld `reason` befüllt

#### Scenario: Spieler sieht nur eigenen Kommentar

- **WHEN** ein Spieler `GET /api/games/{id}/responses` aufruft
- **THEN** ist `reason` nur für den eigenen Eintrag befüllt; alle anderen haben `reason: null`

#### Scenario: Elternteil sieht nur Kinder-Kommentare

- **WHEN** ein Elternteil `GET /api/games/{id}/responses` aufruft
- **THEN** ist `reason` nur für Einträge der eigenen Kinder befüllt
