## ADDED Requirements

### Requirement: Eltern sehen RSVP-Status ihrer Kinder auf der Terminliste

`GET /api/training-sessions` und `GET /api/games/my` SHALL für User mit `is_parent = true` ein zusätzliches Feld `children_rsvp` zurückgeben. Das Feld enthält pro Kind des Elternteils einen Eintrag mit `member_id`, `name` und `rsvp` (null wenn noch nicht geantwortet).

#### Scenario: Elternteil mit einem Kind ruft Termine ab

- **WHEN** ein User mit `is_parent = true` `GET /api/training-sessions` aufruft
- **THEN** enthält jedes Session-Objekt ein Feld `children_rsvp: [{member_id, name, rsvp}]`
- **THEN** ist `rsvp` der aktuelle Status des Kindes (`confirmed`, `declined`, `maybe` oder `null`)

#### Scenario: Elternteil mit zwei Kindern im selben Team

- **WHEN** ein Elternteil zwei Kinder via `family_links` verknüpft hat und beide im selben Team sind
- **THEN** enthält `children_rsvp` zwei Einträge — einen pro Kind

#### Scenario: Kind hat noch nicht geantwortet

- **WHEN** ein Kind keine `training_responses`-Zeile für die Session hat
- **THEN** ist `rsvp` im entsprechenden `children_rsvp`-Eintrag `null`

#### Scenario: Nicht-Elternteil erhält kein children_rsvp

- **WHEN** ein User mit Rolle `spieler` oder `trainer` die Termine abruft
- **THEN** ist das Feld `children_rsvp` nicht in der Antwort enthalten

---

### Requirement: Eltern können RSVPs für Kinder abgeben

`POST /api/training-sessions/{id}/respond` und `POST /api/games/{id}/respond` SHALL für `elternteil`-User funktionieren, wenn `member_id` (Kind-ID) im Body mitgesendet wird. Das Frontend MUSS `member_id` für jedes Kind separat mitsenden.

#### Scenario: Elternteil sagt für Kind ab

- **WHEN** ein Elternteil `POST /api/training-sessions/42/respond` mit `{ member_id: 17, status: "declined", reason: "Krank" }` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist in `training_responses` ein Eintrag für `member_id = 17` mit `status = "declined"` und `reason = "Krank"` vorhanden

#### Scenario: Elternteil ohne member_id wird abgelehnt

- **WHEN** ein Elternteil `POST /api/training-sessions/42/respond` ohne `member_id` aufruft
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Elternteil versucht fremdes Kind zu melden

- **WHEN** ein Elternteil `member_id` eines Kindes sendet, das nicht in `family_links` verknüpft ist
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: RSVP-Fehler sind für den User sichtbar

Das Frontend SHALL bei einem fehlgeschlagenen RSVP-Submit eine sichtbare Fehlermeldung anzeigen. Ein stiller Fehlschlag (kein Feedback) ist nicht zulässig.

#### Scenario: API gibt 400 zurück

- **WHEN** der Server beim RSVP-Submit HTTP 400 oder 403 zurückgibt
- **THEN** zeigt das Frontend eine Fehlermeldung im betroffenen Termin-Card
- **THEN** wird der angezeigte RSVP-Status nicht optimistisch aktualisiert

---

### Requirement: Kommentare werden korrekt gespeichert und angezeigt

Ein Kommentar (Absage- oder Vielleicht-Grund) SHALL beim RSVP-Submit mitgesendet werden und nach einem Reload für den Verfasser sichtbar sein.

#### Scenario: Spieler gibt Absage mit Kommentar

- **WHEN** ein Spieler auf der Terminliste einen Kommentar eingibt und „Absagen" klickt
- **THEN** wird `POST /api/training-sessions/{id}/respond` mit `{ status: "declined", reason: "<text>" }` aufgerufen
- **THEN** ist der Kommentar in `training_responses.reason` gespeichert
- **THEN** ist der Kommentar auf der Detail-Seite für den Spieler selbst sichtbar

#### Scenario: Elternteil gibt Absage für Kind mit Kommentar

- **WHEN** ein Elternteil für ein Kind einen Kommentar eingibt und „Absagen" klickt
- **THEN** wird `member_id` des Kindes mitgesendet
- **THEN** ist der Kommentar für das Elternteil auf der Detail-Seite sichtbar, nicht für andere Eltern oder Spieler
