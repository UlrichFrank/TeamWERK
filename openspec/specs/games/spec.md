### Requirement: Spiel anlegen
Spiele (Heim, Auswärts, Generisch) SHALL optional mit einem Venue verknüpft werden können. Bei Heimspielen wird venue_id automatisch auf den Heimhallen-Venue vorausgefüllt, sofern einer als `is_home_venue=true` markiert ist.

#### Scenario: Heimspiel mit Heimhallen-Autofill
- **WHEN** Nutzer öffnet Formular für neues Heimspiel
- **THEN** venue_id wird automatisch auf den is_home_venue-Venue gesetzt; Nutzer kann überschreiben

#### Scenario: Auswärtsspiel ohne Venue
- **WHEN** Nutzer legt Auswärtsspiel ohne Venue-Auswahl an
- **THEN** Spiel wird ohne venue_id gespeichert (null)

#### Scenario: Venue in Response eingebettet
- **WHEN** GET /api/games oder GET /api/games/{id} aufgerufen wird
- **THEN** Response enthält venue-Objekt mit id, name, street, city, postal_code, note (oder null wenn kein Venue)

---

### Requirement: Spielplan für alle eingeloggten User lesbar

`GET /api/games` und `GET /api/games/{id}` SHALL ohne Rollen-Einschränkung für alle authentifizierten User zugänglich sein.

#### Scenario: Spieler ruft Spielplan ab
- **WHEN** ein User mit Rolle spieler oder elternteil `GET /api/games` aufruft
- **THEN** antwortet der Server mit HTTP 200 und der Spielplanliste

#### Scenario: Nav-Eintrag sichtbar
- **WHEN** ein User eingeloggt ist
- **THEN** ist der „Spielplan"-Menüeintrag in der Navigation sichtbar, unabhängig von der Rolle

---

### Requirement: Schreibzugriff für admin, vorstand und trainer

`POST`, `PUT` und `DELETE` auf `/api/games/*` SHALL für die Rollen admin, vorstand und trainer zugänglich sein.

#### Scenario: Vorstand legt Event an
- **WHEN** ein User mit Rolle vorstand `POST /api/games` aufruft
- **THEN** antwortet der Server mit HTTP 201

#### Scenario: Spieler kann kein Event anlegen
- **WHEN** ein User mit Rolle spieler oder elternteil `POST /api/games` aufruft
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: PUT /api/games/{id} erreichbar für trainer und vorstand

`PUT /api/games/{id}` SHALL für die Rollen admin, trainer und vorstand zugänglich sein. Dies ermöglicht dem `GameEditModal` im Kalender das direkte Bearbeiten von Spieltagen durch Trainer.

#### Scenario: Trainer bearbeitet Spieltag via PUT

- **WHEN** ein User mit Rolle trainer `PUT /api/games/{id}` mit gültigen Feldern aufruft
- **THEN** antwortet der Server mit HTTP 200 und den aktualisierten Spieltag-Daten

#### Scenario: Spieler kann Spieltag nicht bearbeiten

- **WHEN** ein User mit Rolle spieler `PUT /api/games/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: Multi-Team-Zuordnung via `game_teams`

Ein Event SHALL einer oder mehreren Mannschaften zugeordnet sein, abgebildet über die Junction-Tabelle `game_teams`.

#### Scenario: Event mit mehreren Teams anlegen
- **WHEN** `POST /api/games` mit `team_ids: [1, 2, 3]` aufgerufen wird
- **THEN** werden in `game_teams` drei Einträge angelegt
- **THEN** wird für jede Mannschaft ein identischer Satz Duty-Slots generiert

#### Scenario: Event ohne Team abgelehnt
- **WHEN** `POST /api/games` ohne `team_ids` oder mit leerem Array aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: `event_type`-Feld

Jedes Event SHALL einen `event_type` haben: `heim`, `auswärts` oder `generisch`.

#### Scenario: Standard-Typ bei fehlendem Feld
- **WHEN** `POST /api/games` ohne `event_type` aufgerufen wird
- **THEN** wird `event_type = 'heim'` gesetzt

#### Scenario: Ungültiger Typ abgelehnt
- **WHEN** `POST /api/games` mit einem ungültigen `event_type` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

---

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

---

### Requirement: Mehrtägige Events mit end_date

Events (heim, auswärts, generisch) SHALL optional ein `end_date` haben, das ein Enddatum (inklusive) für das Event festlegt. Wenn `end_date` gesetzt ist und nach `date` liegt, erstreckt sich das Event über mehrere Tage.

#### Scenario: Event ohne end_date (Standardfall)
- **WHEN** ein Event ohne `end_date` angelegt wird
- **THEN** wird es wie bisher als eintägiges Event behandelt

#### Scenario: Mehrtägiges Event anlegen
- **WHEN** `POST /api/games` mit `end_date` aufgerufen wird und `end_date >= date`
- **THEN** wird das Event mit `end_date` gespeichert und HTTP 201 zurückgegeben

#### Scenario: end_date vor date wird abgelehnt
- **WHEN** `POST /api/games` oder `PUT /api/games/{id}` mit `end_date < date` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: end_date in GET-Response enthalten
- **WHEN** `GET /api/games` oder `GET /api/games/{id}` aufgerufen wird
- **THEN** enthält jedes Event mit gesetztem `end_date` das Feld `end_date` in der Response (ISO-Datum-String)
- **THEN** Events ohne `end_date` liefern `end_date: null`

#### Scenario: Mehrtägiges Event bearbeiten
- **WHEN** `PUT /api/games/{id}` mit neuem `end_date` aufgerufen wird
- **THEN** wird `end_date` aktualisiert
- **WHEN** `PUT /api/games/{id}` mit `end_date: null` aufgerufen wird
- **THEN** wird `end_date` auf NULL gesetzt (Event wird wieder eintägig)
