## ADDED Requirements

### Requirement: Vollständiger Mitglieder-CSV-Export
Das System SHALL auf `GET /api/members/export` eine CSV-Datei mit allen Mitgliedern und allen 12 Feldern zurückgeben. Zugriff ist nur für Admins erlaubt.

#### Scenario: Export enthält alle Felder
- **WHEN** ein Admin `GET /api/members/export` aufruft
- **THEN** gibt der Server eine CSV-Datei zurück mit genau diesen Spalten in dieser Reihenfolge: `Mitgliedsnummer, Vorname, Nachname, Geburtsdatum, Geschlecht, Passnummer, Trikotnummer, Position, Status, Benutzer_Email, Erziehungsberechtigter1_Email, Erziehungsberechtigter2_Email`

#### Scenario: Leere Felder als leere Spalten
- **WHEN** ein Mitglied keinen Wert für ein optionales Feld hat (z.B. keine Trikotnummer)
- **THEN** wird die Spalte im CSV als leerer String ausgegeben (kein "-", kein "null")

#### Scenario: Alle Mitglieder inkl. ausgetreten
- **WHEN** der Export ausgeführt wird
- **THEN** sind alle Mitglieder enthalten, unabhängig vom Status (aktiv, verletzt, pausiert, passiv, ausgetreten)

### Requirement: Benutzer- und Erziehungsberechtigten-Emails im Export
Das System SHALL die Email-Adressen verknüpfter Benutzer und Erziehungsberechtigter in den Export aufnehmen.

#### Scenario: Verknüpfter Benutzer
- **WHEN** ein Mitglied über `members.user_id` mit einem User verknüpft ist
- **THEN** erscheint die Email dieses Users in der Spalte `Benutzer_Email`

#### Scenario: Erziehungsberechtigte via family_links
- **WHEN** ein Mitglied über `family_links` mit 1 oder 2 Elternteilen verknüpft ist
- **THEN** erscheinen deren Emails in `Erziehungsberechtigter1_Email` bzw. `Erziehungsberechtigter2_Email`

#### Scenario: Kein Erziehungsberechtigter vorhanden
- **WHEN** ein Mitglied keine family_links hat
- **THEN** sind beide Erziehungsberechtigten-Spalten leer

### Requirement: CSV-Format und Encoding
Das System SHALL die CSV-Datei in UTF-8 mit Semikolon als Trennzeichen ausgeben.

#### Scenario: Korrekte HTTP-Response
- **WHEN** der Export ausgeführt wird
- **THEN** hat die Response `Content-Type: text/csv; charset=utf-8` und `Content-Disposition: attachment; filename="mitglieder.csv"`
