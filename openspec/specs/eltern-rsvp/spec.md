# eltern-rsvp Specification

## Purpose
TBD - created by archiving change eltern-rsvp-kinder-sichtbar. Update Purpose after archive.
## Requirements
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

`POST /api/training-sessions/{id}/respond` und `POST /api/games/{id}/respond` SHALL für `elternteil`-User funktionieren, wenn `member_id` (Kind-ID) im Body mitgesendet wird. Das Frontend MUSS `member_id` für jedes Kind separat mitsenden. Durch Proxy-Accounts ist sichergestellt, dass `members.user_id` für Kinder gesetzt ist — Sichtbarkeitsfilter, die auf `user_id` basieren, funktionieren dadurch auch für Kinder zuverlässig.

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

#### Scenario: Kind mit Proxy-Account — Sichtbarkeitsfilter greift korrekt

- **WHEN** ein Kind über einen Proxy-Account verfügt (`members.user_id` ist gesetzt)
- **THEN** werden eventuelle `user_id`-basierte Sichtbarkeitsabfragen für dieses Kind korrekt ausgewertet (kein NULL-Handling-Fallback nötig)

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

### Requirement: children_rsvp umfasst Kinder im erweiterten Kader

`GET /api/training-sessions` und `GET /api/games/my` SHALL im Feld `children_rsvp` auch Kinder des Elternteils einschließen, die für das Event-Team **nur** über `kader_extended_members` (erweiterter Kader) geführt sind — identisch zu Stammkader-Kindern. Steht ein Kind sowohl im Stamm- als auch im erweiterten Kader desselben Teams, erscheint es genau einmal (Stammkader hat Vorrang).

#### Scenario: Elternteil mit Kind nur im erweiterten Kader (Training)

- **WHEN** ein Elternteil ein Kind via `family_links` verknüpft hat, das nur über `kader_extended_members` einem Team zugeordnet ist
- **WHEN** das Elternteil `GET /api/training-sessions` für eine Trainingseinheit dieses Teams abruft
- **THEN** enthält `children_rsvp` einen Eintrag für dieses Kind

#### Scenario: Elternteil mit Kind nur im erweiterten Kader (Spiel)

- **WHEN** ein Elternteil ein Kind verknüpft hat, das nur über `kader_extended_members` einem Team zugeordnet ist
- **WHEN** das Elternteil `GET /api/games/my` für ein Spiel dieses Teams abruft
- **THEN** enthält `children_rsvp` einen Eintrag für dieses Kind

#### Scenario: Kind in Stamm- und erweitertem Kader erscheint einmal

- **WHEN** ein Kind im selben Team sowohl in `kader_members` als auch in `kader_extended_members` steht
- **THEN** enthält `children_rsvp` genau einen Eintrag für dieses Kind

### Requirement: Elternteil kann RSVP für erw.-Kader-Kind abgeben

`POST /api/training-sessions/{id}/respond` und `POST /api/games/{id}/respond` SHALL für ein Elternteil funktionieren, wenn das mitgesendete `member_id` ein über `family_links` verknüpftes Kind ist, das für das Event-Team über `kader_extended_members` geführt ist.

#### Scenario: Elternteil sagt für erw.-Kader-Kind zu

- **WHEN** ein Elternteil `POST /api/training-sessions/{id}/respond` mit `{ member_id: <erw-kader-kind>, status: "confirmed" }` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist in `training_responses` ein Eintrag für dieses `member_id` mit `status = "confirmed"` vorhanden

#### Scenario: Status erscheint danach auf der Detailseite

- **WHEN** das Elternteil für sein erw.-Kader-Kind eine Rückmeldung abgegeben hat
- **THEN** ist der Status in `GET /api/training-sessions/{id}/attendances` für dieses Kind sichtbar (`is_extended: true`)

### Requirement: Kein Auto-Confirm für erw.-Kader-Kinder im children_rsvp

Bei `rsvp_opt_out = 1` eines Events SHALL der automatische `confirmed`-Status im `children_rsvp` ausschließlich für Kinder gelten, die über `kader_members` (Stammkader) im Team sind. Kinder, die für das Event-Team nur über `kader_extended_members` geführt sind, SHALL bei fehlender Rückmeldung `rsvp: null` erhalten (immer explizite Rückmeldung nötig).

#### Scenario: Stammkader-Kind wird bei opt-out auto-confirmed

- **WHEN** ein Kind im Stammkader des Teams ist und das Event `rsvp_opt_out = 1` hat
- **WHEN** für das Kind keine Response-Zeile existiert
- **THEN** ist `rsvp` im `children_rsvp`-Eintrag `"confirmed"`

#### Scenario: Erw.-Kader-Kind erhält KEINEN Auto-Confirm

- **WHEN** ein Kind nur über `kader_extended_members` im Team ist und das Event `rsvp_opt_out = 1` hat
- **WHEN** für das Kind keine Response-Zeile existiert
- **THEN** ist `rsvp` im `children_rsvp`-Eintrag `null`
