## ADDED Requirements

### Requirement: Import-Verhalten ist durch HTTP-Charakterisierungstests festgenagelt
Das beobachtbare Verhalten von `POST /api/members/import` SHALL durch HTTP-Tests abgedeckt sein,
die Report-JSON, Row-Fehlermeldungen und DB-Effekte für die tragenden Verzweigungen festhalten:
BOM-Handling, Delimiter-Erkennung, Spalten-Aliase, CSV-interne Dublettenerkennung, die 400-
Fehlerpfade, den not_found-Pfad und Row-Fehler bei leeren Pflichtfeldern.

#### Scenario: Delimiter-Erkennung ist ein expliziter Contract
- **WHEN** eine CSV mit `;`- bzw. `,`-Trennung importiert wird
- **THEN** je ein Test SHALL das erwartete Parsing festhalten, und ein Test SHALL fixieren, dass die Trennzeichen-Erkennung nur die erste Zeile betrachtet

#### Scenario: CSV-interne Dublette
- **WHEN** dieselbe (Vorname, Nachname, Geburtsdatum) mehrfach in der CSV steht
- **THEN** ein Test SHALL festhalten, dass beide Zeilen als Fehler markiert werden, mit den zeilennummern-referenzierenden Meldungen

#### Scenario: 400-Fehlerpfade
- **WHEN** eine Pflichtspalte fehlt, die CSV inkonsistente Feldzahlen hat, die Datei leer ist oder das Datei-Feld fehlt
- **THEN** je ein Test SHALL den 400-Status (text/plain, kein JSON) und den erwarteten Meldungstext festhalten

#### Scenario: Row-Fehler statt Abbruch
- **WHEN** eine Datenzeile leere Pflichtfelder hat
- **THEN** ein Test SHALL festhalten, dass die Zeile als Row-Fehler (Status 200 gesamt) behandelt wird und valide Zeilen weiterhin verarbeitet werden

#### Scenario: changes[]-Report ist als Contract festgehalten
- **WHEN** ein Update mehrere Felder eines Mitglieds ändert
- **THEN** ein Test SHALL die exakten `changes`-Strings (`"Feld: alt → neu"`) und ihre Reihenfolge festhalten, da dieser beobachtbare Report vollständig in der zu extrahierenden `buildMemberUpdate`-Einheit entsteht

#### Scenario: Beide Ambiguitäts-Ausgänge sind abgedeckt
- **WHEN** eine enrich-Zeile ohne Geburtsdatum auf mehrere gleichnamige Mitglieder trifft
- **THEN** ein Test SHALL den frühen Mehrdeutigkeits-Ausgang („… Treffer – Geburtsdatum in CSV fehlt") festhalten, getrennt vom Ausgang über gleichnamige Datensätze ohne Geburtsdatum

### Requirement: Import ist verhaltenserhaltend in benannte Einheiten unter der Komplexitätsschwelle zerlegt
`members.Import` SHALL in benannte Funktionen/Methoden zerlegt sein (`parseImportCSV`,
`detectCSVDuplicates`, `lookupExistingMember`, `insertNewMember`, `buildMemberUpdate`, Top-Level-
`normalize*`), sodass die Funktion die Komplexitäts-Schwellen aus `metrics/thresholds.yml`
(gocognit, gocyclo) einhält. Die Zerlegung SHALL kein beobachtbares Verhalten ändern.

#### Scenario: Charakterisierungssuite bleibt nach jedem Schritt grün
- **WHEN** ein Extract-Schritt durchgeführt wird
- **THEN** `go test ./internal/members/` SHALL ohne Änderung an den Charakterisierungstests grün bleiben

#### Scenario: Komplexität unter der Gate-Schwelle
- **WHEN** `make metrics-gate` nach abgeschlossenem Refactor läuft
- **THEN** `Import` SHALL die konfigurierten gocognit-/gocyclo-Schwellen einhalten

#### Scenario: Exakte Fehler-Contracts bleiben erhalten
- **WHEN** die Extract-Schritte durchgeführt sind
- **THEN** die englischen 400-Texte und die deutschen Row-Meldungen SHALL wörtlich unverändert sein
