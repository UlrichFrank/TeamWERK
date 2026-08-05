## ADDED Requirements

### Requirement: Mannschafts-Picker im GameEditModal folgt dem Event-Typ

Das `GameEditModal` SHALL die auswählbaren Mannschaften abhängig vom `event_type` des
bearbeiteten Events aus unterschiedlichen Quellen beziehen — analog zum Event-Wizard
(`event-wizard`):

- `event_type='generisch'`: Multi-Select über **alle aktiven Mannschaften** der aktiven Saison
  (Quelle `GET /api/teams/names`), unabhängig von der Rolle des Nutzers.
- `event_type='heim'`/`'auswärts'`: Single-Select über die nutzergefilterte Liste
  (Quelle `GET /api/teams`) — für einen reinen Trainer also nur die eigenen Mannschaften.

Alle Mannschaften werden mit dem berechneten Kurznamen aus `buildTeamShortNames` beschriftet;
ein Rückfall auf den rohen DB-Namen findet nicht statt (`team-names-endpoint`).

#### Scenario: Trainer bearbeitet generisches Event

- **WHEN** ein reiner Trainer das `GameEditModal` für ein Event mit `event_type='generisch'`
  öffnet
- **THEN** erscheint für **jede** aktive Mannschaft des Vereins eine Checkbox
- **THEN** sind die aktuell am Event beteiligten Mannschaften vorausgewählt
- **THEN** kann er eine fremde Mannschaft anhaken und speichern

#### Scenario: Trainer bearbeitet Heimspiel

- **WHEN** ein reiner Trainer das `GameEditModal` für ein Event mit `event_type='heim'` öffnet
- **THEN** enthält das Single-Select nur seine eigenen Mannschaften

#### Scenario: Beteiligte Mannschaft bleibt sichtbar

- **WHEN** ein generisches Event Mannschaften enthält, die nicht zu den eigenen des Nutzers
  gehören
- **THEN** sind diese Mannschaften als angehakte Checkbox sichtbar und können gezielt abgewählt
  werden — sie verschwinden nicht unsichtbar aus der Auswahl

#### Scenario: Server lehnt Entfernen der letzten eigenen Mannschaft ab

- **WHEN** ein reiner Trainer alle eigenen Mannschaften abwählt und speichert
- **THEN** antwortet der Server mit HTTP 403 (`game-mutation-team-scope`)
- **THEN** zeigt das Modal die Fehlermeldung im Alert-Fehler-Stil und bleibt geöffnet
