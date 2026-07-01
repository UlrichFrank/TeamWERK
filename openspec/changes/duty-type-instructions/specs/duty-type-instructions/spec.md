## ADDED Requirements

### Requirement: Anleitung als Eigenschaft des Dienst-Typs

Das System SHALL pro Eintrag in `duty_types` genau eine Markdown-Anleitung
speichern. Die Anleitung MUSS am Dienst-Typ hängen, nicht am einzelnen Slot
oder am Auto-Regen-Template.

#### Scenario: Neu angelegter Dienst-Typ

- **WHEN** ein Vorstand einen neuen Dienst-Typ per `POST /api/duty-types` anlegt
- **THEN** ist `instruction_md` initial die leere Zeichenkette
- **AND** ist `instruction_updated_at` NULL
- **AND** ist `instruction_updated_by` NULL

#### Scenario: Anleitung überlebt Slot-Regeneration

- **WHEN** ein Dienst-Typ eine Anleitung hat
- **AND** ein Spiel mit zugeordneten Slots dieses Typs geändert wird und die
  Auto-Duty-Regeneration die Slots neu erzeugt
- **THEN** bleibt `duty_types.instruction_md` unverändert

### Requirement: Nur Vorstand / Admin darf die Anleitung ändern

Das System SHALL Schreib-Zugriff auf die Anleitung auf System-Rolle `admin`
oder Vereinsfunktion `vorstand` beschränken. Der Schreib-Endpoint MUSS
`PUT /api/duty-types/{id}/instruction` sein und den Body
`{"markdown": "..."}` annehmen.

#### Scenario: Vorstand setzt Anleitung

- **WHEN** ein Nutzer mit `club_functions` enthält `vorstand`
  `PUT /api/duty-types/{id}/instruction` mit gültigem Body sendet
- **THEN** antwortet der Server mit HTTP 200
- **AND** ist `instruction_md` in der Datenbank auf den übergebenen Text gesetzt
- **AND** ist `instruction_updated_at` auf den Zeitpunkt der Änderung gesetzt
- **AND** ist `instruction_updated_by` auf die User-ID des Aufrufers gesetzt
- **AND** wird ein SSE-Ereignis `duties` gesendet

#### Scenario: Standard-Nutzer wird abgelehnt

- **WHEN** ein Nutzer mit System-Rolle `standard` und ohne die
  Vereinsfunktion `vorstand` denselben Aufruf sendet
- **THEN** antwortet der Server mit HTTP 403
- **AND** wird die Anleitung in der Datenbank nicht verändert

#### Scenario: Anonymer Aufruf

- **WHEN** der Aufruf ohne gültigen Bearer-JWT erfolgt
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Unbekannter Dienst-Typ

- **WHEN** der Aufrufer ein `id` verwendet, das nicht in `duty_types` existiert
- **THEN** antwortet der Server mit HTTP 404
- **AND** entsteht keine Zeile in `duty_types` durch den Aufruf

#### Scenario: Fehlender Body

- **WHEN** der Body fehlt oder das Feld `markdown` fehlt oder keine
  Zeichenkette ist
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Überlanger Body

- **WHEN** der Wert von `markdown` mehr als 65_536 Byte UTF-8 umfasst
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Anleitung ist Teil der Lese-Antworten

Das System SHALL die Anleitung in bestehende Lese-Endpoints aufnehmen, ohne
einen neuen Read-Endpoint einzuführen.

#### Scenario: Typen-Liste

- **WHEN** ein autorisierter Nutzer `GET /api/duty-types` aufruft
- **THEN** enthält jeder Eintrag die Felder `instruction_md`,
  `instruction_updated_at` und `instruction_updated_by`

#### Scenario: Dienstbörse

- **WHEN** ein authentifizierter Nutzer `GET /api/duty-board` aufruft
- **THEN** enthält jeder Slot das Feld `duty_type_id` (Integer)
- **AND** das Feld `has_instruction` (Boolean), das genau dann `true` ist,
  wenn `duty_types.instruction_md` für den zugehörigen Typ nicht leer ist

### Requirement: Slot-Link erscheint nur bei vorhandener Anleitung

Das Frontend SHALL in der Dienstbörse pro Slot einen Link
„Anleitung ansehen" (Icon `BookOpen`, `aria-label`) genau dann rendern, wenn
`has_instruction === true`.

#### Scenario: Slot mit Anleitung

- **WHEN** ein Slot mit `has_instruction=true` in `DutySlotList` gerendert wird
- **THEN** ist ein Router-Link auf `/dienste/anleitung/<duty_type_id>` sichtbar
- **AND** ein Klick auf den Link löst nicht das Claim-/Unclaim-Verhalten des
  Slots aus

#### Scenario: Slot ohne Anleitung

- **WHEN** ein Slot mit `has_instruction=false` gerendert wird
- **THEN** wird kein Anleitung-Link angezeigt

### Requirement: Sichere Darstellung der Anleitung

Das Frontend SHALL die Anleitung ausschließlich über einen sanitisierten
Markdown-Renderer darstellen. Roher HTML-Inhalt MUSS blockiert werden.

#### Scenario: Sanitisierung

- **WHEN** eine Anleitung den Text `<script>alert(1)</script>` enthält
- **THEN** rendert der Viewer den Skript-Block **nicht** als ausführbares
  Element
- **AND** kein Skript wird beim Öffnen der Anleitung ausgeführt

#### Scenario: Bild-Referenz aus Dokumente-Bereich

- **WHEN** eine Anleitung ein Bild mit dem Muster
  `![Alt](/dokumente/datei/<fileId>)` enthält
- **THEN** wird ein `<img>` mit exakt dieser `src` gerendert
- **AND** die Rechteprüfung auf die Datei erfolgt beim Aufruf der Ziel-URL
  über den bestehenden `DocumentFileLinkPage`-Pfad, nicht über einen neuen
  Endpoint

### Requirement: Beispieltext bei leerer Anleitung

Der Editor SHALL bei einer leeren Anleitung einen festen Beispieltext als
Vorbelegung in die Textarea setzen und Speichern erst nach einer echten
Änderung durch den Benutzer erlauben.

#### Scenario: Öffnen mit leerer Anleitung

- **WHEN** der Vorstand den Editor für einen Dienst-Typ mit
  `instruction_md === ''` öffnet
- **THEN** ist die Textarea mit dem in `dutyInstructionTemplate.ts`
  hinterlegten Beispieltext vorbelegt
- **AND** ist der Speichern-Button disabled

#### Scenario: Nutzer verändert nichts

- **WHEN** der Vorstand den Editor öffnet, den Beispieltext unverändert lässt
  und den Editor schließt
- **THEN** bleibt `instruction_md` leer und `has_instruction` bleibt `false`

#### Scenario: Nutzer verändert Text

- **WHEN** der Vorstand mindestens ein Zeichen tippt oder löscht
- **THEN** wird der Speichern-Button aktiv
- **AND** löst ein Klick auf Speichern `PUT /api/duty-types/{id}/instruction`
  mit dem aktuellen Textareal-Inhalt aus

### Requirement: Live-Aktualisierung nach Änderung

Das Frontend SHALL bei Empfang des SSE-Ereignisses `duties`
sowohl die Dienstbörse als auch die Anleitungs-Ansicht neu laden.

#### Scenario: Anleitung wird neu geschrieben, während ein Nutzer sie liest

- **WHEN** Nutzer A `DutyInstructionPage` für Typ X geöffnet hat
- **AND** Nutzer B (Vorstand) speichert eine geänderte Anleitung für Typ X
- **THEN** ruft Nutzer A's `useLiveUpdates("duties")` einen
  Reload auf
- **AND** wird die aktualisierte Anleitung angezeigt, ohne dass Nutzer A die
  Seite manuell neu laden muss
