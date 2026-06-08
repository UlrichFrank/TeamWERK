## MODIFIED Requirements

### Requirement: Spiel-Response manuell ändern
Ein Spieler oder berechtigter Elternteil DARF eine Spiel-Response (confirmed/declined/maybe) nur dann manuell ändern, wenn die Response kein gesetztes `absence_id` hat. Ist `absence_id IS NOT NULL`, MUSS die API die Änderung mit HTTP 403 ablehnen. Der Member MUSS stattdessen die zugehörige Abwesenheit löschen.

#### Scenario: Manuelle Änderung ohne Abwesenheit
- **WHEN** ein Nutzer eine Spiel-Response ändert und `absence_id IS NULL`
- **THEN** wird die Änderung akzeptiert

#### Scenario: Manuelle Änderung bei auto-declined Response abgewiesen
- **WHEN** ein Nutzer versucht, eine Spiel-Response mit `absence_id IS NOT NULL` zu ändern
- **THEN** antwortet die API mit HTTP 403

#### Scenario: Trainer kann auto-declined nicht überschreiben
- **WHEN** ein Trainer versucht, eine Response mit `absence_id IS NOT NULL` für ein Kader-Member zu ändern
- **THEN** antwortet die API mit HTTP 403
