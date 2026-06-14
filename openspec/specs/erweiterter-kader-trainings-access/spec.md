### Requirement: Erweiterter Kader sieht Trainings des Teams

Ein Spieler, der im erweiterten Kader (`kader_extended_members`) eines Teams eingetragen ist, SHALL in `GET /api/training-sessions` alle Trainingseinheiten dieses Teams sehen — identisch zu Hauptkader-Spielern.

#### Scenario: Erw.-Kader-Spieler listet Trainings

- **WHEN** ein Spieler mit Rolle `spieler` `GET /api/training-sessions` aufruft und nur im erweiterten Kader (nicht im Hauptkader) eines Teams eingetragen ist
- **THEN** erscheinen die Trainingseinheiten dieses Teams in der Antwort

#### Scenario: Hauptkader-Spieler ist nicht betroffen

- **WHEN** ein Spieler im Hauptkader `GET /api/training-sessions` aufruft
- **THEN** ist das Verhalten unverändert gegenüber dem Status quo

### Requirement: Erweiterter Kader erscheint in der Anwesenheitsliste

Ein Spieler im erweiterten Kader SHALL in `GET /api/training-sessions/{id}/attendances` aufgelistet werden.

#### Scenario: Erw.-Kader-Spieler in Anwesenheitsliste

- **WHEN** ein Trainer `GET /api/training-sessions/{id}/attendances` für ein Training seines Teams abruft
- **THEN** sind Spieler aus dem erweiterten Kader des Teams in der Liste enthalten

#### Scenario: RSVP eines Erw.-Kader-Spielers sichtbar

- **WHEN** ein Erw.-Kader-Spieler sein RSVP via `POST /api/training-sessions/{id}/respond` abgegeben hat
- **THEN** ist sein Status in der Anwesenheitsliste sichtbar

### Requirement: Erweiterter Kader erhält Benachrichtigungen bei neuen Trainings

Wenn eine neue Trainingseinheit oder -serie angelegt wird, SHALL das System auch Spieler im erweiterten Kader benachrichtigen (Push/E-Mail), sofern sie einen verknüpften User-Account haben.

#### Scenario: Benachrichtigung bei neuer Trainingseinheit

- **WHEN** ein Trainer `POST /api/training-sessions` oder `POST /api/training-series` ausführt
- **THEN** erhalten Spieler im erweiterten Kader des Teams eine Benachrichtigung (sofern User-Account vorhanden)

#### Scenario: Spieler ohne User-Account wird nicht benachrichtigt

- **WHEN** ein Erw.-Kader-Spieler keinen verknüpften `users`-Datensatz hat (`members.user_id IS NULL`)
- **THEN** wird kein Benachrichtigungsversuch für diesen Spieler unternommen
