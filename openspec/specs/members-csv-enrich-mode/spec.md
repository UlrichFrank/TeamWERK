# members-csv-enrich-mode Specification

## Purpose

Diese Spezifikation beschreibt die Capability `members-csv-enrich-mode`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Status-Spalte „Status TeamWERK"

Das System SHALL beim CSV-Import den Lebenszyklus-Status eines Mitglieds ausschließlich aus der CSV-Spalte `Status TeamWERK` ableiten. Erlaubte Eingabewerte sind die zulässigen Werte des CHECK-Constraint auf `members.status` (`aktiv`, `verletzt`, `pausiert`, `ausgetreten`, `passiv`, `honorar`, `anwaerter`) sowie der Alias `gekündigt` → `ausgetreten`. Unbekannte Werte werden beim Anlegen auf `aktiv` gemappt; beim Update wird `members.status` in diesem Fall nicht verändert.

#### Scenario: Status TeamWERK = passiv legt passives Mitglied an

- **GIVEN** eine CSV-Zeile mit `Status TeamWERK=passiv` und kein bestehendes Mitglied
- **WHEN** der Import im Modus `update` (Create-Pfad bei `not_found`) oder `append` läuft
- **THEN** wird das Mitglied mit `members.status='passiv'` angelegt

#### Scenario: Status TeamWERK = gekündigt aliasiert ausgetreten

- **GIVEN** eine CSV-Zeile mit `Status TeamWERK=gekündigt`
- **WHEN** der Import einen Bestandsmatch findet
- **THEN** wird `members.status='ausgetreten'` gesetzt

### Requirement: Direkte Beitragsfrei-Spalte

Das System SHALL den `beitragsfrei`-Flag ausschließlich aus der CSV-Spalte `beitragsfrei` ableiten: Wert `ja` (case-insensitive, getrimmt) → `1`, sonst `0`. Im Enrich-Modus darf das Flag NUR von `0` auf `1` ergänzt werden; ein bestehendes `1` wird nicht auf `0` zurückgesetzt.

#### Scenario: Beitragsfrei = ja setzt Flag bei Anlage

- **WHEN** eine CSV-Zeile mit `beitragsfrei=ja` ein neues Mitglied erzeugt
- **THEN** wird `members.beitragsfrei=1` gespeichert

#### Scenario: Enrich respektiert bestehendes Flag

- **GIVEN** ein Bestandsmitglied mit `members.beitragsfrei=1`
- **WHEN** der Enrich-Import die Zeile mit `beitragsfrei=` (leer) verarbeitet
- **THEN** bleibt `members.beitragsfrei=1` unverändert

### Requirement: Spalte „Grund für Beitragsfreiheit"

Das System SHALL die CSV-Spalte `Grund für Beitragsfreiheit` auf das Feld `members.beitragsfrei_grund` mappen. Im Enrich-Modus wird ein bereits gefülltes DB-Feld nicht überschrieben; ein leeres DB-Feld wird mit dem CSV-Wert befüllt.

#### Scenario: Grund wird beim Anlegen übernommen

- **WHEN** eine CSV-Zeile mit `beitragsfrei=ja` und `Grund für Beitragsfreiheit=kein aktiver Sportler mehr` ein neues Mitglied erzeugt
- **THEN** wird `members.beitragsfrei_grund='kein aktiver Sportler mehr'` gespeichert

#### Scenario: Enrich überschreibt belegten Grund nicht

- **GIVEN** ein Bestandsmitglied mit `members.beitragsfrei_grund='Zweitspielrecht'`
- **WHEN** der Enrich-Import die Zeile mit `Grund für Beitragsfreiheit=kein aktiver Sportler mehr` verarbeitet
- **THEN** bleibt `members.beitragsfrei_grund='Zweitspielrecht'`

### Requirement: Enrich-Modus im CSV-Import
Das System MUST einen dritten Import-Modus `enrich` unterstützen. In diesem Modus werden ausschließlich leere Felder bestehender Mitglieder befüllt. Es werden keine neuen Mitglieder angelegt und keine belegten Felder überschrieben.

#### Scenario: Leeres Feld wird ergänzt
- **WHEN** eine CSV-Zeile auf ein bestehendes Mitglied matcht und ein Feld in der DB leer (NULL oder leer) ist, das in der CSV einen Wert enthält
- **THEN** wird das DB-Feld mit dem CSV-Wert befüllt und die Zeile erhält Status `updated`

#### Scenario: Belegtes Feld wird nicht überschrieben
- **WHEN** eine CSV-Zeile auf ein bestehendes Mitglied matcht und ein Feld in der DB bereits einen Wert enthält
- **THEN** bleibt der DB-Wert unverändert, auch wenn die CSV einen abweichenden Wert enthält

#### Scenario: Kein Match — Zeile wird übersprungen
- **WHEN** eine CSV-Zeile kein bestehendes Mitglied findet (weder via Name+DOB noch via Name allein)
- **THEN** wird keine neue Zeile angelegt; die CSV-Zeile erhält Status `not_found` im Report

#### Scenario: Unverändertes Mitglied
- **WHEN** eine CSV-Zeile auf ein bestehendes Mitglied matcht, aber alle CSV-Felder entweder leer sind oder zu belegten DB-Feldern gehören
- **THEN** erhält die Zeile Status `unchanged`

### Requirement: Matching-Strategie im Enrich-Modus
Das System MUST Mitglieder zuerst via Vorname+Nachname+Geburtsdatum suchen. Fehlt das Geburtsdatum in der CSV, MUST auf Vorname+Nachname-Matching zurückgefallen werden.

#### Scenario: Eindeutiges Match ohne Geburtsdatum
- **WHEN** die CSV kein Geburtsdatum enthält und genau ein Mitglied mit übereinstimmendem Vor- und Nachname existiert
- **THEN** wird dieses Mitglied als Match verwendet

#### Scenario: Mehrdeutiges Match ohne Geburtsdatum
- **WHEN** die CSV kein Geburtsdatum enthält und zwei oder mehr Mitglieder mit übereinstimmendem Vor- und Nachname existieren
- **THEN** erhält die Zeile Status `error` mit Meldung „mehrdeutig (N Treffer)"

### Requirement: IBAN-Behandlung im Enrich-Modus
IBAN und Kontoinhaber MUST im Enrich-Modus denselben „nur leer befüllen"-Regeln folgen wie alle anderen Felder. Die MOD-97-Validierung MUST weiterhin ausgeführt werden.

#### Scenario: IBAN ergänzen wenn leer
- **WHEN** das IBAN-Feld in der DB leer ist und die CSV eine gültige IBAN enthält
- **THEN** wird die IBAN gespeichert

#### Scenario: IBAN-Validierungswarnung bei ungültiger IBAN
- **WHEN** die CSV eine IBAN enthält, die die MOD-97-Prüfung nicht besteht
- **THEN** wird eine `iban_warning` im ImportRow gesetzt; das Feld wird nicht geschrieben

#### Scenario: Bestehende IBAN bleibt erhalten
- **WHEN** das IBAN-Feld in der DB bereits einen Wert enthält
- **THEN** wird die CSV-IBAN ignoriert, unabhängig davon ob sie identisch oder verschieden ist

### Requirement: ImportReport-Erweiterung
Das System MUST den `ImportReport` um einen `not_found`-Zähler erweitern und `ImportRow` MUST den neuen Status `not_found` unterstützen.

#### Scenario: not_found-Zähler im Summary
- **WHEN** der Import abgeschlossen ist und N Zeilen keinen Match hatten
- **THEN** enthält der `ImportReport` `not_found: N`

#### Scenario: not_found-Status in der Zeilendetailansicht
- **WHEN** eine Zeile den Status `not_found` hat
- **THEN** enthält `ImportRow` `status: "not_found"`, `name` aus der CSV und optional `dob`

### Requirement: Frontend-Modus-Auswahl
Das Frontend MUST einen dritten Radio-Button „Nur leere Felder ergänzen" im Import-Dialog anzeigen, der den `enrich`-Modus auswählt.

#### Scenario: Modus auswählen
- **WHEN** der Nutzer „Nur leere Felder ergänzen" auswählt und die Vorschau startet
- **THEN** wird `mode=enrich` an das Backend gesendet

#### Scenario: not_found im Preview-Report
- **WHEN** der Preview-Report Zeilen mit Status `not_found` enthält
- **THEN** werden diese grau dargestellt mit dem Icon `—` und dem Text „nicht gefunden"

#### Scenario: not_found-Badge im Summary
- **WHEN** `not_found > 0` im Report
- **THEN** zeigt das Summary einen Badge „X nicht gefunden"
