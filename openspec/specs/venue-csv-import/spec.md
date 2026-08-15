# venue-csv-import Specification

## Purpose
TBD - created by archiving change venues-csv-import. Update Purpose after archive.
## Requirements
### Requirement: Admin kann CSV-Datei mit Veranstaltungsorten importieren
Das System SHALL einen Endpoint `POST /api/venues/import` bereitstellen, der eine CSV-Datei (multipart/form-data, Feld `file`) akzeptiert und die enthaltenen Hallen per Upsert in die `venues`-Tabelle schreibt. Nur Nutzer mit Rolle `admin` dürfen diesen Endpoint aufrufen.

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
- **WHEN** ein Admin die Seite `/veranstaltungsorte` öffnet
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

### Requirement: Hallennummer wird importiert und an Bestands-Venues rück-verknüpft

Das System SHALL beim Import der BWHV-Hallenliste die Spalte `Nummer` (Hallennummer) lesen
und in `venues.hall_number` speichern. Für bereits existierende Venues SHALL der Import die
Hallennummer per Match über `(Name, Ort, Straße)` nachtragen (Backfill). Ist die Zuordnung
mehrdeutig (mehrere Hallennummern für dieselbe Adresse) oder findet sich kein Venue, SHALL
`hall_number` `NULL` bleiben und der Fall im Ergebnis-Report (`ambiguous` bzw. `unmatched`)
ausgewiesen werden.

Das System SHALL `hall_number` als nullable Feld mit einem Partial-Unique-Index
`WHERE hall_number IS NOT NULL` führen, sodass Venues ohne eindeutige Hallennummer
(manuelle Nicht-BWHV-Orte, mehrdeutige Adressen) koexistieren können.

#### Scenario: Neue Halle wird mit Hallennummer angelegt
- **WHEN** eine Hallenlisten-Zeile mit `Nummer` importiert wird, deren Adresse noch kein Venue hat
- **THEN** wird ein Venue mit gesetztem `hall_number` angelegt

#### Scenario: Bestehendes Venue erhält seine Hallennummer per Backfill
- **WHEN** eine Hallenlisten-Zeile auf genau ein bestehendes Venue mit gleichem `(Name, Ort, Straße)` passt und dieses Venue noch keine `hall_number` hat
- **THEN** wird `venues.hall_number` auf die Nummer der Zeile gesetzt

#### Scenario: Mehrdeutige Adresse bleibt ohne Hallennummer
- **WHEN** zwei Hallenlisten-Zeilen mit unterschiedlichen Nummern dieselbe Adresse `(Name, Ort, Straße)` tragen (BWHV-Datenfehler)
- **THEN** bleibt das betroffene Venue bei `hall_number = NULL` und der Fall erscheint im Report unter `ambiguous`

#### Scenario: Manuelles Nicht-BWHV-Venue behält NULL
- **WHEN** ein bestehendes Venue keiner Hallenlisten-Zeile entspricht (z. B. Vereinsgaststätte, Jugendraum)
- **THEN** bleibt `hall_number = NULL` und das Venue wird nicht verändert

#### Scenario: Report weist Backfill-Ergebnis aus
- **WHEN** ein Hallenlisten-Import abgeschlossen ist
- **THEN** enthält die Antwort neben `imported`/`updated`/`skipped` auch die Zahl der eindeutig zugeordneten, mehrdeutigen und nicht zugeordneten Hallennummern

