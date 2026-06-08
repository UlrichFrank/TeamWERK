## MODIFIED Requirements

### Requirement: Eltern können RSVPs für Kinder abgeben
`POST /api/training-sessions/{id}/respond` und `POST /api/games/{id}/respond` SHALL für `elternteil`-User funktionieren, wenn `member_id` (Kind-ID) im Body mitgesendet wird. Das Frontend MUSS `member_id` für jedes Kind separat mitsenden. Durch Proxy-Accounts ist nun sichergestellt, dass `members.user_id` für Kinder gesetzt ist — Sichtbarkeitsfilter, die auf `user_id` basieren, funktionieren dadurch auch für Kinder zuverlässig.

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
