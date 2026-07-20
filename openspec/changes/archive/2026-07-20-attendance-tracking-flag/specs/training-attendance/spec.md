## ADDED Requirements

### Requirement: Erst-Save aktiviert `attendance_tracked` der Session

`POST /api/training-sessions/{id}/attendances` SHALL nach mindestens einem erfolgreich persistierten Spieler-Upsert innerhalb derselben Transaktion `training_sessions.attendance_tracked` auf `1` setzen. Übersprungene Einträge (Trainer-only, serien-abgemeldete Mitglieder) zählen für diese Bedingung NICHT.

#### Scenario: Erster Save aktiviert Flag

- **WHEN** ein Trainer für eine Session mit `attendance_tracked=0` einen Bulk-Save mit einem Spieler-Eintrag aufruft
- **THEN** ist nach der Response `training_sessions.attendance_tracked=1`

#### Scenario: No-op-Save aktiviert Flag NICHT

- **WHEN** ein Trainer einen Bulk-Save aufruft, dessen sämtliche Einträge übersprungen werden (nur Trainer-only oder nur serien-abgemeldete Mitglieder)
- **THEN** bleibt `attendance_tracked` unverändert (weiterhin `0`, falls vorher `0`)

### Requirement: Reset der Anwesenheits-Erfassung

`DELETE /api/training-sessions/{id}/attendance-tracking` SHALL `training_sessions.attendance_tracked` auf `0` setzen, ohne bestehende `training_attendances`-Rows zu verändern. Die Route SHALL denselben Authz-Check wie `POST /api/training-sessions/{id}/attendances` durchlaufen (Trainer des Teams, `sportliche_leitung`, `admin`). Sie SHALL nach Erfolg `hub.Broadcast("attendance-changed")` senden und HTTP 204 zurückgeben.

#### Scenario: Trainer resettet Erfassung

- **WHEN** ein Trainer `DELETE /api/training-sessions/{id}/attendance-tracking` für eine Session seines Teams aufruft, die `attendance_tracked=1` hat
- **THEN** antwortet die API mit HTTP 204, `attendance_tracked` ist danach `0`, vorhandene `training_attendances`-Rows bleiben unverändert, und der Hub broadcastet `attendance-changed`

#### Scenario: Reset ist idempotent

- **WHEN** ein Trainer die Reset-Route zweimal hintereinander aufruft
- **THEN** ist beide Male die Response HTTP 204 und `attendance_tracked` bleibt `0`

#### Scenario: Fremdes Team abgewiesen

- **WHEN** ein Trainer, der nicht dem Team der Session zugeordnet ist, die Reset-Route aufruft
- **THEN** antwortet die API mit HTTP 403 und `attendance_tracked` bleibt unverändert

#### Scenario: Unbekannte Session

- **WHEN** ein berechtigter Nutzer die Reset-Route für eine nicht existierende `id` aufruft
- **THEN** antwortet die API mit HTTP 404

#### Scenario: Unauthentifizierter Aufruf abgewiesen

- **WHEN** ein nicht eingeloggter Client die Reset-Route aufruft
- **THEN** antwortet die API mit HTTP 401

### Requirement: Re-Aktivierung nach Reset

Nach einem Reset SHALL ein erneuter `POST /api/training-sessions/{id}/attendances` das Flag wieder auf `1` setzen (dieselbe Semantik wie Erst-Save). Die zuvor gespeicherten `training_attendances`-Rows werden dabei durch den Bulk-Upsert überschrieben oder bleiben erhalten (je nach Payload).

#### Scenario: Erneuter Save re-aktiviert Flag

- **WHEN** eine Session `attendance_tracked=0` hat, aber bereits `training_attendances`-Rows aus einer früheren Erfassung enthält, und ein Trainer erneut Bulk-Save aufruft
- **THEN** ist `attendance_tracked` danach wieder `1` und die Statistik verwendet die aktualisierten/beibehaltenen Rows
