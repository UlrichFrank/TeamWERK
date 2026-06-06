# venue-csv-import Specification

## Purpose
TBD - created by archiving change venues-csv-import. Update Purpose after archive.
## Requirements
### Requirement: Admin kann CSV-Datei mit Veranstaltungsorten importieren
Das System SHALL einen Endpoint `POST /api/admin/venues/import` bereitstellen, der eine CSV-Datei (multipart/form-data, Feld `file`) akzeptiert und die enthaltenen Hallen per Upsert in die `venues`-Tabelle schreibt. Nur Nutzer mit Rolle `admin` dürfen diesen Endpoint aufrufen.

#### Scenario: Erfolgreicher Import
- **WHEN** ein Admin eine gültige BWHV-CSV-Datei hochlädt
- **THEN** gibt der Endpoint HTTP 200 zurück mit `{ imported, updated, skipped, errors }` und alle Hallen sind in der DB

#### Scenario: Neue Halle wird angelegt
- **WHEN** eine Zeile einen Namen enthält, der noch nicht in `venues` existiert
- **THEN** wird eine neue Zeile mit name, street, postal_code, city, note, country="DE", is_home_venue=false eingefügt

#### Scenario: Bestehende Halle wird aktualisiert
- **WHEN** eine Zeile einen Namen enthält, der bereits in `venues` existiert
- **THEN** werden street, postal_code, city, note aktualisiert; `is_home_venue` bleibt unverändert

#### Scenario: Zeile ohne Namen wird übersprungen
- **WHEN** eine Datenzeile einen leeren Namen hat
- **THEN** wird diese Zeile zum `errors`-Array hinzugefügt und übersprungen; der Rest des Imports läuft weiter

#### Scenario: CSV mit BOM-Präfix wird korrekt verarbeitet
- **WHEN** die Datei mit einem UTF-8 BOM beginnt
- **THEN** wird der BOM ignoriert und der Import läuft korrekt

#### Scenario: Hallennamen mit eingebetteten Kommata werden korrekt geparst
- **WHEN** ein Hallenname in der CSV in Anführungszeichen steht und ein Komma enthält (z.B. `"St.-Jakobs-Halle, Feld 1"`)
- **THEN** wird der Name vollständig und korrekt eingelesen

#### Scenario: Preamble-Zeilen werden übersprungen
- **WHEN** die CSV die BWHV-Standardpreamble enthält (Titel-Zeile, Leerzeile, dann Header-Zeile mit "Name")
- **THEN** werden die ersten Zeilen bis zur Header-Zeile (inkl.) übersprungen und nur Datenzeilen importiert

### Requirement: Import-UI als Split-Button auf der Venues-Seite
Das System SHALL den "+ Neuer Ort"-Button durch einen Split-Button ersetzen. Die linke Hälfte öffnet das bestehende Neu-Modal, die rechte Hälfte öffnet ein Dropdown mit dem Eintrag "Import CSV".

#### Scenario: Split-Button zeigt beide Aktionen
- **WHEN** ein Admin die Seite `/admin/veranstaltungsorte` öffnet
- **THEN** sieht er einen zweigeteilten Button: links "+ Neuer Ort", rechts ein ChevronDown

#### Scenario: Dropdown öffnet sich per Klick auf ChevronDown
- **WHEN** der Admin auf den ChevronDown-Teil klickt
- **THEN** öffnet sich ein Dropdown mit dem Eintrag "Import CSV"

#### Scenario: Import-Modal zeigt Ergebnis nach erfolgreichem Import
- **WHEN** der Admin eine Datei auswählt und "Importieren" klickt und der Import erfolgreich ist
- **THEN** zeigt das Modal die Anzahl importierter, aktualisierter und übersprungener Einträge sowie eventuelle Fehler

#### Scenario: Schließen des Dropdowns bei Klick außerhalb
- **WHEN** das Dropdown offen ist und der Nutzer außerhalb klickt
- **THEN** schließt sich das Dropdown

