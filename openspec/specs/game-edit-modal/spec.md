# game-edit-modal Specification

## Purpose

Diese Spezifikation beschreibt die Capability `game-edit-modal`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: GameEditModal zeigt editierbare Felder eines Spieltags

Das `GameEditModal` SHALL die Felder `opponent`, `date`, `time`, `event_type`, `rsvp_opt_out`
und `rsvp_require_reason` eines bestehenden Spieltags anzeigen und editierbar machen. Es
verwendet das Standard-Modal-Design (`bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`).

#### Scenario: Modal öffnet sich mit vorausgefüllten Daten

- **WHEN** das `GameEditModal` für einen Spieltag geöffnet wird
- **THEN** sind alle Felder mit den aktuellen Werten des Spieltags vorausgefüllt — inklusive der Checkboxen für `rsvp_opt_out` und `rsvp_require_reason`

#### Scenario: Modal zeigt alle editierbaren Felder

- **WHEN** das `GameEditModal` offen ist
- **THEN** enthält es ein Eingabefeld für `opponent` (Gegner)
- **THEN** enthält es ein Datumfeld für `date`
- **THEN** enthält es ein Zeitfeld für `time`
- **THEN** enthält es eine Auswahl für `event_type` (heim / auswärts / generisch)
- **THEN** enthält es eine Checkbox „Alle Spieler standardmäßig zugesagt (Opt-Out)" für `rsvp_opt_out`
- **THEN** enthält es eine Checkbox „Begründung bei Absage erforderlich" für `rsvp_require_reason`

#### Scenario: Generischer Event-Type belegt rsvp_require_reason vor

- **WHEN** ein neuer Spieltag im Modal mit `event_type = 'generisch'` angelegt wird
- **THEN** ist die Checkbox `rsvp_require_reason` mit `false` (= 0) vorbelegt

### Requirement: GameEditModal speichert Änderungen via PUT /api/games/{id}

Das Speichern SHALL `PUT /api/games/{id}` aufrufen — inklusive der Felder `rsvp_opt_out` und
`rsvp_require_reason`. Bei Erfolg schließt sich das Modal und der Kalender aktualisiert sich.

#### Scenario: Erfolgreiches Speichern

- **WHEN** ein berechtigter User Änderungen vornimmt und auf „Speichern" klickt
- **THEN** wird `PUT /api/games/{id}` mit den geänderten Feldern aufgerufen — `rsvp_opt_out` und `rsvp_require_reason` werden immer mitgesendet
- **THEN** schließt sich das Modal nach erfolgreicher Antwort
- **THEN** werden die Spieltag-Daten im Kalender aktualisiert (via Reload oder SSE)

#### Scenario: Fehler beim Speichern

- **WHEN** der Server mit einem Fehler antwortet
- **THEN** zeigt das Modal eine Fehlermeldung (Alert-Fehler-Klasse)
- **THEN** bleibt das Modal geöffnet

#### Scenario: Abbrechen ohne Speichern

- **WHEN** ein User auf „Abbrechen" klickt oder Escape drückt
- **THEN** schließt sich das Modal ohne API-Aufruf
- **THEN** bleiben die Spieltag-Daten im Kalender unverändert

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
