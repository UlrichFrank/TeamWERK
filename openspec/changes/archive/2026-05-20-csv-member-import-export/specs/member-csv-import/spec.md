## ADDED Requirements

### Requirement: CSV-Import-Endpunkt
Das System SHALL auf `POST /api/members/import` eine CSV-Datei als `multipart/form-data` entgegennehmen und verarbeiten. Zugriff ist nur für Admins erlaubt.

#### Scenario: Erfolgreicher Import
- **WHEN** ein Admin eine gültige CSV-Datei mit korrektem Header hochlädt
- **THEN** gibt der Server HTTP 200 mit einem JSON-Importbericht zurück

#### Scenario: Ungültige Datei
- **WHEN** die hochgeladene Datei keinen gültigen CSV-Header hat (fehlende Pflichtfelder `Vorname`, `Nachname`)
- **THEN** gibt der Server HTTP 400 zurück

### Requirement: Idempotenz-Schlüssel
Das System SHALL `Vorname + Nachname` (case-insensitiv) als Primärschlüssel zur Erkennung bestehender Mitglieder verwenden. Wenn `Geburtsdatum` in der CSV-Zeile nicht leer ist, wird es als Tiebreaker verwendet.

#### Scenario: Bestehendes Mitglied gefunden
- **WHEN** ein Mitglied mit gleichem Vor- und Nachnamen (case-insensitiv) bereits in der DB existiert
- **THEN** wird es als "bestehend" erkannt und je nach Modus übersprungen oder aktualisiert

#### Scenario: Geburtsdatum als Tiebreaker
- **WHEN** zwei DB-Mitglieder denselben Namen haben und die CSV-Zeile ein Geburtsdatum enthält
- **THEN** wird nur das Mitglied mit übereinstimmendem Geburtsdatum als Match verwendet

#### Scenario: Duplikat innerhalb der CSV
- **WHEN** zwei CSV-Zeilen denselben Idempotenz-Schlüssel haben
- **THEN** werden beide Zeilen mit Status "error" im Bericht markiert (Meldung: "Mehrfach in CSV")

### Requirement: Import-Modi
Das System SHALL zwei Modi unterstützen, gesteuert über den Form-Parameter `mode`.

#### Scenario: Modus "append" — neues Mitglied
- **WHEN** `mode=append` und der Idempotenz-Schlüssel existiert noch nicht in der DB
- **THEN** wird ein neues Mitglied angelegt

#### Scenario: Modus "append" — bestehendes Mitglied
- **WHEN** `mode=append` und der Idempotenz-Schlüssel bereits in der DB existiert
- **THEN** wird die Zeile übersprungen (Status "unchanged" im Bericht)

#### Scenario: Modus "update" — neues Mitglied
- **WHEN** `mode=update` und der Idempotenz-Schlüssel existiert noch nicht in der DB
- **THEN** wird ein neues Mitglied angelegt

#### Scenario: Modus "update" — bestehendes Mitglied mit geänderten Feldern
- **WHEN** `mode=update`, das Mitglied existiert, und mindestens ein nicht-leeres CSV-Feld weicht vom DB-Wert ab
- **THEN** werden nur die abweichenden, nicht-leeren Felder aktualisiert; der Bericht listet jede Änderung als "Feldname: alt → neu"

### Requirement: Non-Destructive-Policy
Das System SHALL niemals Daten durch den Import entfernen oder leeren.

#### Scenario: Leere CSV-Zelle überschreibt nicht
- **WHEN** ein CSV-Feld leer ist und das Mitglied in der DB bereits einen Wert hat
- **THEN** bleibt der DB-Wert unverändert (kein Überschreiben mit leerem Wert)

#### Scenario: Bestehender User-Link bleibt erhalten
- **WHEN** die Spalte `Benutzer_Email` in der CSV leer ist, aber `members.user_id` in der DB gesetzt ist
- **THEN** bleibt `user_id` unverändert

#### Scenario: Bestehende family_links bleiben erhalten
- **WHEN** die Erziehungsberechtigten-Spalten leer sind, aber `family_links` in der DB existieren
- **THEN** bleiben alle `family_links` unverändert

### Requirement: Benutzer- und Erziehungsberechtigten-Verknüpfung beim Import
Das System SHALL Verknüpfungen zu bestehenden Benutzern nur dann anlegen, wenn die Email in der `users`-Tabelle gefunden wird.

#### Scenario: Email gefunden — user_id setzen
- **WHEN** `Benutzer_Email` in der CSV nicht leer ist und ein User mit dieser Email in der DB existiert
- **THEN** wird `members.user_id` auf diesen User gesetzt (nur wenn noch nicht gesetzt)

#### Scenario: Email nicht gefunden — kein Fehler
- **WHEN** `Benutzer_Email` in der CSV nicht leer ist, aber kein User mit dieser Email existiert
- **THEN** wird die Zeile trotzdem verarbeitet; der Bericht enthält einen Hinweis "Email nicht gefunden"

#### Scenario: Erziehungsberechtigter verknüpfen
- **WHEN** `Erziehungsberechtigter1_Email` oder `Erziehungsberechtigter2_Email` nicht leer ist und ein User mit dieser Email existiert
- **THEN** wird ein `family_links`-Eintrag angelegt (nur wenn noch nicht vorhanden)

### Requirement: Importbericht
Das System SHALL nach dem Import einen strukturierten JSON-Bericht zurückgeben.

#### Scenario: Bericht-Struktur
- **WHEN** der Import abgeschlossen ist
- **THEN** enthält der Response-Body: `total`, `created`, `updated`, `unchanged`, `errors` (Zählwerte) sowie `rows` (Array mit Einträgen je verarbeiteter Zeile, jeder Eintrag mit `line`, `status`, `name`, und bei `updated` ein `changes`-Array, bei `error` eine `message`)

#### Scenario: Status "created"
- **WHEN** eine CSV-Zeile ein neues Mitglied angelegt hat
- **THEN** hat der Row-Eintrag `status: "created"` mit `name` und optionalem `dob`

#### Scenario: Status "updated"
- **WHEN** eine CSV-Zeile ein bestehendes Mitglied aktualisiert hat
- **THEN** hat der Row-Eintrag `status: "updated"` mit `name` und `changes: ["Feldname: 'alt' → 'neu'"]`

#### Scenario: Status "unchanged"
- **WHEN** eine CSV-Zeile einem bestehenden Mitglied entspricht, aber keine Änderungen ergeben hat
- **THEN** hat der Row-Eintrag `status: "unchanged"` mit `name`

#### Scenario: Status "error"
- **WHEN** eine CSV-Zeile nicht verarbeitet werden konnte
- **THEN** hat der Row-Eintrag `status: "error"` mit `line`, `name` (wenn erkennbar) und `message`
