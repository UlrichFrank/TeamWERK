# training-attendance Specification

## Purpose
TBD - created by archiving change trainingsplanung. Update Purpose after archive.
## Requirements
### Requirement: Trainer kann Anwesenheit nach dem Training erfassen
Ein Trainer oder Admin SHALL nach einem Training die tatsächliche Anwesenheit aller Mitglieder des Teams als Bulk-Operation erfassen können. Bestehende Einträge werden überschrieben.

#### Scenario: Anwesenheit erfassen
- **WHEN** ein Trainer POST `/api/training-sessions/{id}/attendances` mit einem Array `[{member_id: 5, present: true}, {member_id: 7, present: false}]` aufruft
- **THEN** werden für alle angegebenen Mitglieder `training_attendances`-Rows angelegt oder aktualisiert (Upsert auf UNIQUE(training_id, member_id))

#### Scenario: Trainer kann nur für eigenes Team erfassen
- **WHEN** ein User mit `role='trainer'` versucht, Anwesenheit für eine Session eines anderen Teams zu erfassen
- **THEN** antwortet das System mit HTTP 403

### Requirement: Trainer kann Anwesenheitsliste einer Session abrufen
Ein Trainer oder Admin SHALL die Anwesenheitsliste einer Session abrufen können, die beide Dimensionen zeigt: RSVP-Status (was angesagt wurde) und tatsächliche Anwesenheit. Jedes Element der Liste SHALL ein `is_extended`-Feld enthalten, das anzeigt, ob das Mitglied zum primären Kader (`false`) oder zum erweiterten Kader (`true`) gehört. Mitglieder, die in beiden Kadern sind, SHALL nur einmal erscheinen und gelten als primäres Kader-Mitglied (`is_extended: false`). Für primäre Kader-Mitglieder ohne explizite Rückmeldung gilt `rsvp_opt_out` der Session: ist es aktiv, wird ihr Status als `confirmed` ausgewiesen. Für erweiterte Kader-Mitglieder gilt `rsvp_opt_out` NICHT — ihr Status ist `null` wenn keine explizite Rückmeldung vorliegt, unabhängig von der Session-Konfiguration.

#### Scenario: Anwesenheitsliste abrufen
- **WHEN** ein Trainer GET `/api/training-sessions/{id}/attendances` aufruft
- **THEN** erhält er eine Liste aller Teammitglieder mit jeweils `member_id`, `member_name`, `rsvp_status`, `reason`, `present` und `is_extended` (bool)

#### Scenario: Primärer Kader korrekt markiert
- **WHEN** ein Mitglied via `kader_members` zum Team gehört
- **THEN** hat es `is_extended: false` in der Response

#### Scenario: Erweiterter Kader korrekt markiert
- **WHEN** ein Mitglied nur via `kader_extended_members` zum Team gehört (nicht im primären Kader)
- **THEN** hat es `is_extended: true` in der Response

#### Scenario: Kein Duplikat bei Overlap
- **WHEN** ein Mitglied sowohl im primären als auch im erweiterten Kader ist
- **THEN** erscheint es genau einmal in der Liste mit `is_extended: false`

#### Scenario: Primärer Kader mit rsvp_opt_out auto-confirmed
- **WHEN** eine Session `rsvp_opt_out = true` hat und ein primäres Kader-Mitglied keine Rückmeldung abgegeben hat
- **THEN** hat dieses Mitglied `rsvp_status: "confirmed"` in der Response

#### Scenario: Erweiterter Kader NICHT auto-confirmed
- **WHEN** eine Session `rsvp_opt_out = true` hat und ein erweitertes Kader-Mitglied keine Rückmeldung abgegeben hat
- **THEN** hat dieses Mitglied `rsvp_status: null` in der Response

#### Scenario: Erweiterter Kader mit expliziter Rückmeldung
- **WHEN** ein erweitertes Kader-Mitglied explizit `confirmed` oder `declined` geantwortet hat
- **THEN** wird dieser Status korrekt zurückgegeben, unabhängig von `rsvp_opt_out`

#### Scenario: Diskrepanz sichtbar
- **WHEN** ein Mitglied `rsvp_status='confirmed'` hat, aber `present=false`
- **THEN** sind beide Werte in der Liste sichtbar, sodass Trainer Zusagen ohne Erscheinen erkennen kann

### Requirement: Anwesenheitserfassung nur für vergangene oder aktuelle Sessions
Das System SHALL verhindern, dass Anwesenheit für Sessions in der Zukunft erfasst wird.

#### Scenario: Zukunfts-Session blockiert
- **WHEN** ein Trainer POST `/api/training-sessions/{id}/attendances` für eine Session aufruft, deren `date` in der Zukunft liegt
- **THEN** antwortet das System mit HTTP 422 und der Meldung, dass Anwesenheit erst nach dem Termin erfasst werden kann

### Requirement: Anwesenheit-Speichern ignoriert Trainer-Roster-Einträge
`POST /api/training-sessions/{id}/attendances` SHALL Einträge, die sich auf einen
Trainer-only-Member beziehen (im Kader-Trainerstab, nicht als Spieler im Kader), still
überspringen und die verbleibenden Spieler-Einträge speichern. Ein einzelner Trainer-Eintrag
im Paket darf das Speichern der Spieler-Einträge NICHT verhindern. Die fachliche Regel
„Trainer haben keine Anwesenheitserfassung" bleibt bestehen — für Trainer wird keine
`training_attendances`-Zeile geschrieben.

#### Scenario: Paket mit Trainer und Spieler speichert den Spieler
- **WHEN** ein Trainer `POST /api/training-sessions/{id}/attendances` für ein vergangenes Training mit einem Paket aufruft, das sowohl einen Spieler (`present=true`) als auch einen Trainer des Teams enthält
- **THEN** antwortet die API mit HTTP 204, die `present`-Angabe des Spielers ist persistiert und für den Trainer existiert keine `training_attendances`-Zeile

#### Scenario: Paket nur mit Trainer-Eintrag ist ein No-op
- **WHEN** das Paket ausschließlich Trainer-Einträge enthält
- **THEN** antwortet die API mit HTTP 204 und schreibt keine `training_attendances`-Zeile

#### Scenario: Fremdes Team weiterhin abgewiesen
- **WHEN** ein Nutzer ohne Trainer-Zugriff auf das Team die Route aufruft
- **THEN** antwortet die API mit HTTP 403

### Requirement: Anwesenheitserfassung überspringt abgemeldete Spieler

Bei der Bulk-Anwesenheitserfassung (`POST /api/training-sessions/{id}/attendances`) SHALL das System für jedes übergebene Mitglied, das für die Serie der Session eine greifende Serien-Abmeldung (`serien-abmeldung`-Ableitung) hat, **keine** `training_attendances`-Zeile schreiben — analog zur bestehenden Ausnahme für trainer-only-Mitglieder. Der übrige Speichervorgang für nicht abgemeldete Mitglieder SHALL unbeeinflusst bleiben (kein Abbruch des gesamten Requests). Dadurch entsteht für abgemeldete Spieler kein Attendance-Record und sie bleiben aus dem Statistik-Nenner ausgeschlossen.

#### Scenario: Abgemeldeter Spieler bekommt keine Attendance-Zeile

- **WHEN** ein Trainer `POST /api/training-sessions/{id}/attendances` mit einer Liste sendet, die ein für diese Serie abgemeldetes Mitglied enthält
- **THEN** wird für dieses Mitglied keine `training_attendances`-Zeile angelegt oder aktualisiert

#### Scenario: Restliche Erfassung bleibt erfolgreich

- **WHEN** dieselbe Anfrage neben dem abgemeldeten Mitglied weitere, nicht abgemeldete Mitglieder enthält
- **THEN** werden deren Anwesenheitswerte normal gespeichert und die Anfrage liefert HTTP 200

#### Scenario: Bereits erfasste Anwesenheit bei nachträglicher Abmeldung wird ausgeschlossen

- **WHEN** für ein Mitglied bereits eine `training_attendances`-Zeile existiert und danach eine greifende Serien-Abmeldung angelegt wird
- **THEN** wird die Session für dieses Mitglied in der Statistik dennoch ausgeschlossen (der Abmelde-Ausschluss hat Vorrang, siehe Capability `attendance-statistics`)

