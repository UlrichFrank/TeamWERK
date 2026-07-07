## MODIFIED Requirements

### Requirement: Anleitung ist Teil der Lese-Antworten

Das System SHALL die Anleitung in bestehende Lese-Endpoints aufnehmen, ohne
einen neuen Read-Endpoint einzuführen. Der potenziell große Markdown-Volltext
(`instruction_md`) SHALL dabei NICHT in der Typen-**Liste** (`GET /api/duty-types`)
ausgeliefert werden, sondern nur über den Detail-Pfad; die Liste transportiert
stattdessen ein `has_instruction`-Flag.

#### Scenario: Typen-Liste liefert Flag statt Volltext

- **WHEN** ein autorisierter Nutzer `GET /api/duty-types` aufruft
- **THEN** enthält jeder Eintrag das Feld `has_instruction` (Boolean), das genau
  dann `true` ist, wenn `duty_types.instruction_md` nicht leer ist
- **AND** die Einträge enthalten KEIN `instruction_md`-Feld

#### Scenario: Detail-Pfad behält den Volltext

- **WHEN** ein autorisierter Nutzer den Anleitungs-Detail-Pfad eines Dienst-Typs
  aufruft
- **THEN** enthält die Antwort `instruction_md`, `instruction_updated_at` und
  `instruction_updated_by`

#### Scenario: Dienstbörse

- **WHEN** ein authentifizierter Nutzer `GET /api/duty-board` aufruft
- **THEN** enthält jeder Slot das Feld `duty_type_id` (Integer)
- **AND** das Feld `has_instruction` (Boolean), das genau dann `true` ist,
  wenn `duty_types.instruction_md` für den zugehörigen Typ nicht leer ist
