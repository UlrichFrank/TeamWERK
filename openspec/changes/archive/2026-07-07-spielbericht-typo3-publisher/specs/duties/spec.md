## ADDED Requirements

### Requirement: Spielbericht-Duty-Slot nur für Presseteam sichtbar
Das System SHALL Duty-Slots des Typs „Spielbericht" (identifiziert per `duty_types.name='Spielbericht'` oder dedizierter Flag) im `GET /api/duty-board`-Response nur für User mit `role IN ('presseteam','admin')` ausliefern. Für andere User wird der Slot herausgefiltert, als wäre er nicht vorhanden.

#### Scenario: Standard-User sieht Slot nicht
- **WHEN** ein User mit `role='standard'` `GET /api/duty-board` aufruft
- **THEN** enthält die Response keinen Slot des Typs „Spielbericht"

#### Scenario: Presseteam sieht Slot
- **WHEN** ein User mit `role='presseteam'` `GET /api/duty-board` aufruft
- **THEN** enthält die Response Slots des Typs „Spielbericht"

### Requirement: Spielbericht-Slot-Ziehen prüft Rolle
Das System SHALL beim `POST /api/duty-slots/{id}/take` prüfen: wenn der Slot vom Typ „Spielbericht" ist, MUSS der Requester `role IN ('presseteam','admin')` haben. Andernfalls HTTP 403 mit `{"error":"role_required"}`.

#### Scenario: Standard-User versucht Ziehen
- **WHEN** ein Standard-User einen „Spielbericht"-Slot per direktem API-Call ziehen will
- **THEN** liefert das System HTTP 403 (Backend-Guard, nicht nur UI-Filter)

### Requirement: Spielbericht-Slot wird auto-regeneriert
Das System SHALL bei jedem Anlegen oder Update eines Spiels mit `event_type IN ('heim','auswärts')` und gesetztem `template_id` automatisch einen Duty-Slot vom Typ „Spielbericht" erzeugen, wenn noch keiner existiert. Slot-`due_at` wird als `game.end_time + 24h` gesetzt (oder `game.date 23:59 + 24h` falls kein end_time). Custom-editierte Slots (`is_custom=1`) werden nicht überschrieben.

#### Scenario: Neues Heimspiel
- **WHEN** ein neues Heim-Spiel mit `template_id` angelegt wird
- **THEN** existiert danach genau ein Duty-Slot vom Typ „Spielbericht" für dieses Spiel

#### Scenario: Kein Slot bei template_id=NULL
- **WHEN** ein generisches Event ohne `template_id` angelegt wird
- **THEN** wird kein Spielbericht-Slot erzeugt

### Requirement: Slot-Erledigung durch Publish
Das System SHALL beim erfolgreichen `POST /api/match-reports/{id}/publish` den zugehörigen Duty-Slot (`match_reports.duty_slot_id`) als erledigt markieren und dem Slot-Owner die Dienstkonto-Gutschrift geben — analog zum manuellen „erledigt"-Klick.

#### Scenario: Publish zählt aufs Dienstkonto
- **WHEN** ein Presseteam-User seinen Bericht erfolgreich publisht
- **THEN** ist der Slot in `duty_slots` erledigt UND das Dienstkonto des Users um den Slot-Wert erhöht
