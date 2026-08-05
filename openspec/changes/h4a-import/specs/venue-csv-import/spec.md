## ADDED Requirements

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
