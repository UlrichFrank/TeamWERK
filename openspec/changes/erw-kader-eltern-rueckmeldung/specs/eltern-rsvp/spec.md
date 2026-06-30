## ADDED Requirements

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
