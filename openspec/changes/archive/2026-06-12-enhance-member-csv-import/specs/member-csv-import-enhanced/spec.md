## ADDED Requirements

### Requirement: Fehlende Mitgliederfelder aus CSV importieren
Das System SHALL beim CSV-Import die Spalten `Adresse`, `PLZ`, `Ort`, `Mitglied seit`, `IBAN`, `Kontoinhaber` und `SEPA Mandat` in die entsprechenden DB-Felder (`street`, `zip`, `city`, `join_date`, `iban`, `account_holder`, `sepa_mandat`) der `members`-Tabelle schreiben. Im `update`-Modus werden nichtleere CSV-Werte immer übernommen (auch wenn bereits ein DB-Wert vorhanden ist). Im `append`-Modus werden diese Felder beim Anlegen neuer Datensätze gesetzt.

#### Scenario: Neue Felder beim Anlegen gesetzt
- **WHEN** ein Mitglied neu importiert wird (nicht in DB vorhanden) und die CSV-Zeile enthält Werte für Adresse, PLZ, Ort, Mitglied seit, IBAN, Kontoinhaber und SEPA Mandat
- **THEN** legt das System den Datensatz mit allen diesen Feldern an

#### Scenario: Felder im Update-Modus überschrieben
- **WHEN** ein bestehendes Mitglied im `mode=update` importiert wird und die CSV enthält abweichende Werte für street/zip/city/join_date/iban/account_holder/sepa_mandat
- **THEN** überschreibt das System die vorhandenen DB-Werte mit den CSV-Werten und listet die Änderungen im Report

#### Scenario: Leere CSV-Felder lassen DB-Wert unverändert
- **WHEN** eine CSV-Zeile für ein Feld keinen Wert hat (leer)
- **THEN** lässt das System den bestehenden DB-Wert unverändert

#### Scenario: SEPA-Mandat-Normalisierung
- **WHEN** die CSV-Spalte `SEPA Mandat` den Wert `vorliegend` enthält
- **THEN** setzt das System `sepa_mandat = 1`
- **WHEN** die Spalte leer oder ein anderer Wert ist
- **THEN** setzt das System `sepa_mandat = 0`

### Requirement: IBAN-Validierung per MOD-97
Das System SHALL jede nichtleere IBAN aus der CSV vor dem Speichern per MOD-97-Algorithmus validieren. Eine ungültige IBAN wird nicht in die DB geschrieben. Die Zeile wird dennoch mit den übrigen Feldern verarbeitet. Der Report enthält für diese Zeile eine `IBANWarning`-Meldung mit der konkreten Fehlerbeschreibung.

#### Scenario: Gültige IBAN wird gespeichert
- **WHEN** die CSV-Spalte `IBAN` eine IBAN enthält, die MOD-97 = 1 ergibt und für DE-IBANs genau 22 Zeichen hat
- **THEN** speichert das System die IBAN und kein Warning wird erzeugt

#### Scenario: Ungültige IBAN — falsche Prüfziffer
- **WHEN** die IBAN-Prüfsumme (MOD-97) nicht 1 ergibt
- **THEN** speichert das System die IBAN nicht, setzt die übrigen Felder der Zeile normal, und fügt dem ImportRow eine `IBANWarning`-Meldung `"Prüfziffer falsch"` hinzu

#### Scenario: Ungültige IBAN — falsche Länge
- **WHEN** eine DE-IBAN nicht genau 22 Zeichen hat
- **THEN** speichert das System die IBAN nicht und fügt eine `IBANWarning`-Meldung `"DE-IBAN muss 22 Zeichen haben, hat N"` hinzu

#### Scenario: Leere IBAN wird übersprungen
- **WHEN** die CSV-Spalte `IBAN` leer ist
- **THEN** bleibt der IBAN-DB-Wert unverändert und kein Warning wird erzeugt

### Requirement: Email-Klassifizierung und automatische Verknüpfung
Das System SHALL die CSV-Spalten `Email` und `Email 2` per Heuristik klassifizieren und entsprechende Verknüpfungen anlegen. Die Klassifizierung erfolgt zweistufig: (1) Alter des Mitglieds (Geburtsdatum < 18 Jahre zum Importzeitpunkt), (2) Vorname des Mitglieds als Substring im normalisierten lokalen Teil der Email-Adresse (Kleinbuchstaben, nur a-z).

Klassifizierungsregeln:
- Alter ≥ 18 → EIGEN: System sucht `users WHERE email = ?` und setzt `members.user_id` wenn gefunden und noch nicht gesetzt
- Alter < 18 UND Vorname NICHT im lokalen Teil → ELTERN: System sucht `users WHERE email = ?` und legt `family_links(parent_user_id, member_id)` an wenn gefunden und noch nicht vorhanden
- Alter < 18 UND Vorname im lokalen Teil → KIND-EIGEN: keine automatische Verknüpfung, nur Notiz im Report

`Email` und `Email 2` werden unabhängig voneinander nach derselben Logik verarbeitet.

#### Scenario: Eigene Email eines erwachsenen Mitglieds verknüpft
- **WHEN** ein Mitglied (Alter ≥ 18) importiert wird, dessen `Email`-Spalte die Email-Adresse eines bestehenden Users enthält, und das Mitglied noch keine `user_id` hat
- **THEN** setzt das System `members.user_id` auf die ID dieses Users und meldet die Verknüpfung im Report

#### Scenario: Eltern-Email eines minderjährigen Mitglieds verknüpft
- **WHEN** ein minderjähriges Mitglied importiert wird, die `Email`-Spalte keinen Vornamen-Match hat, und ein User mit dieser Email existiert
- **THEN** legt das System einen `family_links`-Eintrag an (wenn noch nicht vorhanden) und meldet die Verknüpfung im Report

#### Scenario: Email führt zu keinem bekannten User
- **WHEN** für eine Email kein User-Account in der DB existiert
- **THEN** wird keine Verknüpfung angelegt; das Report-Feld `Changes` enthält eine Notiz `"Email: <adresse> (kein User-Account gefunden)"`

#### Scenario: Zwei Eltern-Emails für dasselbe Kind
- **WHEN** `Email` und `Email 2` eines minderjährigen Mitglieds beide als ELTERN klassifiziert werden und beide zu existierenden Users führen
- **THEN** legt das System für beide Users je einen `family_links`-Eintrag an

#### Scenario: Kind-Email ohne Verknüpfung
- **WHEN** ein minderjähriges Mitglied importiert wird und eine Email ihren Vornamen enthält (KIND-EIGEN)
- **THEN** wird keine Verknüpfung vorgenommen; das Report-Feld `Changes` enthält eine Notiz `"Email 2: <adresse> (Kind-Email, kein automatischer Link)"`

#### Scenario: Bereits verknüpfter User wird nicht überschrieben
- **WHEN** das Mitglied bereits eine `user_id` hat
- **THEN** wird keine neue user_id-Verknüpfung vorgenommen (auch wenn die CSV eine andere Email enthält)

### Requirement: Preview-Modus für CSV-Import
Das System SHALL einen `mode=preview`-Parameter für `POST /api/members/import` unterstützen. Im Preview-Modus läuft die komplette Import-Logik durch (Feldvergleiche, IBAN-Validierung, Email-Klassifizierung), ohne jedoch Schreiboperationen in die DB auszuführen. Der zurückgegebene Report ist identisch mit dem eines echten `mode=update`-Laufs.

#### Scenario: Preview zeigt geplante Änderungen ohne zu schreiben
- **WHEN** ein Admin `POST /api/members/import` mit `mode=preview` aufruft
- **THEN** gibt das System einen vollständigen `ImportReport` zurück (created/updated/unchanged/errors mit Changes-Liste)
- **THEN** ist der Zustand der DB nach dem Aufruf identisch mit dem Zustand vor dem Aufruf

#### Scenario: Preview meldet IBAN-Warnings
- **WHEN** die CSV ungültige IBANs enthält und `mode=preview` verwendet wird
- **THEN** erscheinen die IBAN-Warnings im Report, obwohl nichts geschrieben wurde

#### Scenario: Unbekannter mode-Parameter fällt auf append zurück
- **WHEN** `mode` einen unbekannten Wert hat (weder `append`, `update` noch `preview`)
- **THEN** verhält sich das System wie `mode=append`
