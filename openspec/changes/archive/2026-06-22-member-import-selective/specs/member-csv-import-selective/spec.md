## ADDED Requirements

### Requirement: Feld-Auswahl beim CSV-Import (`fields`)

Das System SHALL ein optionales Formfeld `fields` für `POST /api/members/import` unterstützen, das eine komma-separierte Liste von DB-Spaltennamen enthält. Bei einem Update bestehender Mitglieder (Modi `update`, `enrich` sowie der Dry-Run `preview`) werden ausschließlich die in `fields` gelisteten Spalten verändert. Zulässige Spalten sind: `member_number`, `date_of_birth`, `gender`, `pass_number`, `position`, `status`, `home_club`, `jersey_number`, `street`, `zip`, `city`, `join_date`, `account_holder`, `sepa_mandat`, `iban`. Ist `fields` leer oder nicht gesetzt, sind alle Spalten erlaubt (rückwärtskompatibel).

Die Whitelist greift NUR auf den Update-Pfad bestehender Mitglieder. Neu angelegte Mitglieder (`created`) werden unabhängig von `fields` mit allen vorhandenen CSV-Werten gespeichert.

#### Scenario: Nur ausgewählte Spalte wird aktualisiert
- **WHEN** ein bestehendes Mitglied im Modus `update` importiert wird, die CSV abweichende Werte für `IBAN` und `Status` enthält und `fields=iban` gesendet wird
- **THEN** aktualisiert das System nur `iban` und lässt `status` unverändert
- **THEN** listet der Report für die Zeile nur die IBAN-Änderung

#### Scenario: Leeres fields aktualisiert alle Spalten
- **WHEN** ein bestehendes Mitglied im Modus `update` importiert wird und `fields` leer oder nicht gesetzt ist
- **THEN** verhält sich der Import exakt wie bisher und übernimmt alle nichtleeren, abweichenden CSV-Werte

#### Scenario: Status-Auswahl steuert abgeleitetes beitragsfrei
- **WHEN** `fields` die Spalte `status` NICHT enthält
- **THEN** wird weder `status` noch das daraus abgeleitete `beitragsfrei` verändert
- **WHEN** `fields` die Spalte `status` enthält
- **THEN** dürfen sowohl `status` als auch das abgeleitete `beitragsfrei` aktualisiert werden

#### Scenario: Neue Mitglieder ignorieren die Whitelist
- **WHEN** im Modus `update` ein in der DB nicht vorhandenes Mitglied importiert wird und `fields=iban` gesendet wird
- **THEN** legt das System den neuen Datensatz mit allen vorhandenen CSV-Feldern an (nicht nur IBAN)

### Requirement: Mitglieder-Auswahl beim CSV-Import (`apply_lines`)

Das System SHALL ein optionales Formfeld `apply_lines` für `POST /api/members/import` unterstützen, das eine komma-separierte Liste von CSV-Zeilennummern (1-basiert, inkl. Headerzeile; Datenzeilen beginnen bei 2) enthält. Außerhalb des Dry-Runs werden Updates an Bestandsmitgliedern nur für Zeilen geschrieben, deren Nummer in `apply_lines` enthalten ist. Ist `apply_lines` leer oder nicht gesetzt, werden alle Zeilen angewendet. Die Auswahl greift nicht im Dry-Run (`preview`); dort werden stets alle Zeilen ausgewertet.

#### Scenario: Nur ausgewählte Zeile wird geschrieben
- **WHEN** zwei Bestandsmitglieder (CSV-Zeilen 2 und 3) je eine Änderung hätten und `apply_lines=2` ohne Dry-Run gesendet wird
- **THEN** schreibt das System nur die Änderung von Zeile 2 in die DB
- **THEN** bleibt das Mitglied aus Zeile 3 in der DB unverändert

#### Scenario: Leeres apply_lines wendet alle Zeilen an
- **WHEN** ohne Dry-Run importiert wird und `apply_lines` leer oder nicht gesetzt ist
- **THEN** werden alle Zeilen mit Änderungen geschrieben

#### Scenario: apply_lines im Dry-Run ignoriert
- **WHEN** mit `mode=preview` (Dry-Run) und gesetztem `apply_lines` importiert wird
- **THEN** wertet das System alle Zeilen aus und schreibt nichts in die DB

### Requirement: Report-Status `skipped`

Das System SHALL den `ImportReport` um einen `skipped`-Zähler und `ImportRow` um den Status `skipped` erweitern. Eine Zeile erhält im Apply-Lauf (kein Dry-Run) den Status `skipped`, wenn sie auf ein Bestandsmitglied matcht, Änderungen hätte, aber nicht in `apply_lines` enthalten ist. Im Dry-Run werden solche Zeilen weiterhin als `updated` mit ihrer Änderungsliste gemeldet.

#### Scenario: Abgewählte Zeile wird als skipped gemeldet
- **WHEN** ein Bestandsmitglied mit Änderungen ohne Dry-Run importiert wird und seine Zeilennummer NICHT in `apply_lines` steht
- **THEN** erhält die Zeile im Report Status `skipped` und ihre Änderungen werden nicht in die DB geschrieben
- **THEN** erhöht das System den `skipped`-Zähler im Report

#### Scenario: Dry-Run zeigt abgewählte Zeilen weiterhin als updated
- **WHEN** im Dry-Run (`preview`) eine Zeile Änderungen hätte
- **THEN** wird sie als `updated` mit Änderungsliste gemeldet, unabhängig von `apply_lines`

#### Scenario: skipped-Zähler im Summary
- **WHEN** der Apply-Lauf abgeschlossen ist und N Zeilen übersprungen wurden
- **THEN** enthält der `ImportReport` `skipped: N`

### Requirement: Frontend-Auswahl im Import-Dialog

Das Frontend SHALL im Import-Dialog die Feld- und Mitglieder-Auswahl anbieten. In Schritt 1 (Datei + Modus) SHALL bei den Modi `update` und `enrich` eine Liste von Feld-Checkboxen erscheinen (standardmäßig alle ausgewählt); die ausgewählten Spalten werden als `fields` an Vorschau und Anwendung gesendet. In der Vorschau SHALL jede `updated`-Zeile eine Checkbox (standardmäßig angehakt) erhalten; beim Anwenden werden nur die angehakten Zeilennummern als `apply_lines` gesendet. Der Status `skipped` SHALL mit eigenem Icon, eigener Farbe und eigenem Summary-Badge dargestellt werden.

#### Scenario: Feld-Checkboxen nur bei update/enrich
- **WHEN** der Nutzer den Modus `update` oder `enrich` wählt
- **THEN** zeigt der Dialog die Feld-Checkboxen (alle vorausgewählt)
- **WHEN** der Nutzer den Modus `append` wählt
- **THEN** werden keine Feld-Checkboxen angezeigt

#### Scenario: Abgewählte Felder werden nicht gesendet
- **WHEN** der Nutzer einzelne Feld-Checkboxen abwählt und die Vorschau startet
- **THEN** sendet das Frontend `fields` nur mit den verbliebenen Spalten

#### Scenario: Abgewählte Zeile wird beim Anwenden ausgespart
- **WHEN** der Nutzer in der Vorschau die Checkbox einer Zeile abwählt und „Jetzt anwenden" klickt
- **THEN** sendet das Frontend `apply_lines` ohne diese Zeilennummer

#### Scenario: skipped-Darstellung im Ergebnis
- **WHEN** der Ergebnis-Report Zeilen mit Status `skipped` enthält
- **THEN** zeigt das Frontend diese mit eigenem Icon/Farbe und einen Summary-Badge „X übersprungen"
