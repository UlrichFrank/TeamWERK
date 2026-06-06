# game-deletion-cascade Specification

## Purpose
TBD - created by archiving change dienste-kaskadiert-loeschen. Update Purpose after archive.
## Requirements
### Requirement: Dienste werden beim Löschen eines Termins automatisch mitgelöscht
Wenn ein Spiel oder Sonstiger Termin gelöscht wird, MÜSSEN alle verknüpften `duty_slots` (und deren `duty_assignments` via vorhandenem CASCADE) automatisch gelöscht werden. Es gibt keinen opt-out.

#### Scenario: Löschen eines Termins mit verknüpften Diensten
- **WHEN** ein Admin einen Termin (Spiel oder Sonstiger Termin) löscht
- **THEN** werden alle `duty_slots` mit dieser `game_id` gelöscht
- **THEN** werden alle `duty_assignments` dieser Slots gelöscht
- **THEN** existiert kein Dienst mehr mit Bezug auf den gelöschten Termin

#### Scenario: Löschen eines Termins ohne Dienste
- **WHEN** ein Admin einen Termin ohne verknüpfte Dienste löscht
- **THEN** wird nur der Termin gelöscht, keine Fehler

#### Scenario: Kaskadierung greift auf DB-Ebene
- **WHEN** ein Termin über jeden Frontend-Pfad (GameEditModal oder SpieltagDetailPage) gelöscht wird
- **THEN** sind die Dienste immer gelöscht, unabhängig von Query-Parametern

### Requirement: Keine "Dienste behalten"-Option beim Löschen
Das Frontend DARF keine Möglichkeit anbieten, Dienste beim Löschen eines Termins zu behalten. Die bisherige Checkbox in SpieltagDetailPage SHALL entfernt werden.

#### Scenario: Delete-Dialog ohne Checkbox
- **WHEN** ein Admin die Löschen-Bestätigung für einen Termin mit Diensten öffnet
- **THEN** gibt es keine Checkbox "Verknüpfte Dienste ebenfalls löschen"
- **THEN** zeigt der Dialog an, wie viele Dienste mitgelöscht werden

