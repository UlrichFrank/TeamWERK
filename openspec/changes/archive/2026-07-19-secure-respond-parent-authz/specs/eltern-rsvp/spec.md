## MODIFIED Requirements

### Requirement: Eltern können RSVPs für Kinder abgeben

`POST /api/training-sessions/{id}/respond` und `POST /api/games/{id}/respond` SHALL eine Rückmeldung für ein per `member_id` benanntes Kind entgegennehmen, wenn der aufrufende User (System-Rolle `standard`) über `family_links` mit diesem Member verknüpft ist. Das Frontend MUSS `member_id` für jedes Kind separat mitsenden. Durch Proxy-Accounts ist sichergestellt, dass `members.user_id` für Kinder gesetzt ist — Sichtbarkeitsfilter, die auf `user_id` basieren, funktionieren dadurch auch für Kinder zuverlässig.

Die Autorisierung SHALL **durchgesetzt** sein (die frühere Verzweigung über `claims.Role == "elternteil"`/`"spieler"` war wirkungslos, da `users.role` nur `admin|standard` kennt). Es gilt:

- Ein User darf immer für sein **eigenes** Member-Record antworten.
- Ein User darf für ein **fremdes** `member_id` nur antworten, wenn er entweder Manage-Berechtigung für das Event-Team besitzt (admin oder trainer/sportliche_leitung/vorstand des Teams) **oder** über `family_links` Elternteil dieses Members ist.
- Jede andere Kombination (fremdes Member, keine Verknüpfung, keine Manage-Berechtigung) SHALL mit **HTTP 403** abgelehnt werden.

#### Scenario: Elternteil sagt für eigenes Kind ab

- **WHEN** ein Elternteil `POST /api/training-sessions/42/respond` mit `{ member_id: 17, status: "declined", reason: "Krank" }` aufruft und Member 17 via `family_links` mit ihm verknüpft ist
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist in `training_responses` ein Eintrag für `member_id = 17` mit `status = "declined"` vorhanden

#### Scenario: User antwortet für eigenes Member-Record

- **WHEN** ein User ohne `member_id` (oder mit seinem eigenen `member_id`) antwortet und sein Account mit einem Member verknüpft ist
- **THEN** antwortet der Server mit HTTP 204 für das eigene Member-Record

#### Scenario: User versucht fremdes Member zu melden

- **WHEN** ein User ein `member_id` sendet, das weder sein eigenes Member-Record noch ein über `family_links` verknüpftes Kind ist, und er keine Manage-Berechtigung für das Event-Team hat
- **THEN** antwortet der Server mit HTTP 403
- **THEN** wird keine Zeile in `training_responses`/`game_responses` angelegt oder geändert

#### Scenario: Manage-Berechtigte dürfen für beliebiges Teammitglied antworten

- **WHEN** ein admin oder ein Trainer/Vorstand des Event-Teams `member_id` eines beliebigen Mitglieds sendet
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Kind mit Proxy-Account — Sichtbarkeitsfilter greift korrekt

- **WHEN** ein Kind über einen Proxy-Account verfügt (`members.user_id` ist gesetzt)
- **THEN** werden eventuelle `user_id`-basierte Sichtbarkeitsabfragen für dieses Kind korrekt ausgewertet (kein NULL-Handling-Fallback nötig)
