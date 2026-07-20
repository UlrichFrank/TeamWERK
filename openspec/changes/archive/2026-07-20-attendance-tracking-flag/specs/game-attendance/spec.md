## ADDED Requirements

### Requirement: Erst-Save aktiviert `attendance_tracked` des Spiels

`POST /api/games/{id}/attendances` SHALL nach mindestens einem erfolgreich persistierten Spieler-Upsert innerhalb derselben Transaktion `games.attendance_tracked` auf `1` setzen. Übersprungene Einträge (Trainer-only) zählen für diese Bedingung NICHT.

#### Scenario: Erster Save aktiviert Flag

- **WHEN** ein Trainer für ein Spiel mit `attendance_tracked=0` einen Bulk-Save mit einem Spieler-Eintrag aufruft
- **THEN** ist nach der Response `games.attendance_tracked=1`

#### Scenario: No-op-Save aktiviert Flag NICHT

- **WHEN** ein Trainer einen Bulk-Save aufruft, dessen sämtliche Einträge übersprungen werden (nur Trainer-only)
- **THEN** bleibt `attendance_tracked` unverändert

### Requirement: Reset der Anwesenheits-Erfassung eines Spiels

`DELETE /api/games/{id}/attendance-tracking` SHALL `games.attendance_tracked` auf `0` setzen, ohne bestehende `game_attendances`-Rows zu verändern. Die Route SHALL denselben Authz-Check wie `POST /api/games/{id}/attendances` durchlaufen (Trainer eines beteiligten Teams, `sportliche_leitung`, `admin`). Sie SHALL nach Erfolg `hub.Broadcast("attendance-changed")` senden und HTTP 204 zurückgeben.

#### Scenario: Trainer resettet Erfassung

- **WHEN** ein Trainer `DELETE /api/games/{id}/attendance-tracking` für ein Spiel eines seiner Teams aufruft, das `attendance_tracked=1` hat
- **THEN** antwortet die API mit HTTP 204, `attendance_tracked` ist danach `0`, vorhandene `game_attendances`-Rows bleiben unverändert, und der Hub broadcastet `attendance-changed`

#### Scenario: Reset ist idempotent

- **WHEN** ein Trainer die Reset-Route zweimal hintereinander aufruft
- **THEN** ist beide Male die Response HTTP 204 und `attendance_tracked` bleibt `0`

#### Scenario: Trainer eines fremden Teams abgewiesen

- **WHEN** ein Trainer ohne Trainer-Funktion in einem der Teams des Spiels die Reset-Route aufruft
- **THEN** antwortet die API mit HTTP 403

#### Scenario: Unbekanntes Spiel

- **WHEN** ein berechtigter Nutzer die Reset-Route für eine nicht existierende `id` aufruft
- **THEN** antwortet die API mit HTTP 404

#### Scenario: Unauthentifizierter Aufruf abgewiesen

- **WHEN** ein nicht eingeloggter Client die Reset-Route aufruft
- **THEN** antwortet die API mit HTTP 401

### Requirement: Re-Aktivierung nach Reset (Spiel)

Nach einem Reset SHALL ein erneuter `POST /api/games/{id}/attendances` das Flag wieder auf `1` setzen. Vorhandene `game_attendances`-Rows werden dabei durch den Bulk-Upsert überschrieben oder bleiben erhalten (je nach Payload).

#### Scenario: Erneuter Save re-aktiviert Flag

- **WHEN** ein Spiel `attendance_tracked=0` hat, aber bereits `game_attendances`-Rows aus einer früheren Erfassung enthält, und ein Trainer erneut Bulk-Save aufruft
- **THEN** ist `attendance_tracked` danach wieder `1` und die Statistik verwendet die aktualisierten/beibehaltenen Rows
