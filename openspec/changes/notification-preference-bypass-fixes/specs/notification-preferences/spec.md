## ADDED Requirements

### Requirement: Kategorien `operativ` und `sonstiges`
Das System SHALL die Präferenz-Kategorien `operativ` (Vereins-/Funktionärs-Erinnerungen) und `sonstiges` („Sonstige Events") kennen — persistierbar in `notification_preferences.category` (Migration 027) und Teil von `push.ValidCategories`. Beide defaulten auf `push_enabled=true`.

#### Scenario: Neue Kategorien speicherbar
- **WHEN** ein Nutzer `PUT /api/profile/notification-preferences` mit `operativ: { push: false }` oder `sonstiges: { push: false }` aufruft
- **THEN** wird die Zeile gespeichert (kein CHECK-Fehler) und 204 zurückgegeben

#### Scenario: Defaults enthalten neue Kategorien
- **WHEN** `GET /api/profile/notification-preferences` ohne gespeicherte Zeilen aufgerufen wird
- **THEN** enthält die Antwort `operativ` und `sonstiges` mit `push=true`

### Requirement: Funktionärs-Reminder respektieren `operativ`
Die Push-Trigger match-report-review-reminder, attendance-reminder und match-report-submitted SHALL nur an Empfänger senden, die `operativ` nicht deaktiviert haben.

#### Scenario: Opt-out unterdrückt den Push
- **WHEN** ein Empfänger `operativ` `push_enabled=0` gesetzt hat und einer dieser Trigger feuert
- **THEN** erhält der Empfänger KEINEN Push

#### Scenario: Default sendet weiterhin
- **WHEN** ein Empfänger keine `operativ`-Zeile hat und ein Trigger feuert
- **THEN** erhält der Empfänger den Push (Default an)

### Requirement: video-ready respektiert `sonstiges`
Die „Video ist bereit"-Benachrichtigung SHALL nur an Empfänger senden, die `sonstiges` nicht deaktiviert haben.

#### Scenario: Opt-out unterdrückt den Video-Push
- **WHEN** ein Empfänger `sonstiges` `push_enabled=0` gesetzt hat und ein Video fertig transkodiert wird
- **THEN** erhält der Empfänger KEINEN Push

### Requirement: Mitfahranfrage respektiert `carpooling`
`RequestPairing` SHALL die Mitfahranfrage nur senden, wenn der Empfänger `carpooling` nicht deaktiviert hat (konsistent mit `ConfirmPairing`/`RejectPairing`).

#### Scenario: Opt-out unterdrückt die Anfrage-Push
- **WHEN** der angefragte Nutzer `carpooling` `push_enabled=0` gesetzt hat und eine Mitfahranfrage gestellt wird
- **THEN** erhält er KEINEN Push

### Requirement: Datenverlust-Warnung bleibt unabschaltbar
Die video-retention-Warnung (Video wird in 7 Tagen gelöscht) SHALL unabhängig von jeder Push-Präferenz zugestellt werden.

#### Scenario: Warnung ignoriert Opt-out
- **WHEN** ein Team-Trainer beliebige Push-Kategorien deaktiviert hat und ein Video die T-7-Löschgrenze erreicht
- **THEN** erhält der Trainer die Löschwarnung dennoch
