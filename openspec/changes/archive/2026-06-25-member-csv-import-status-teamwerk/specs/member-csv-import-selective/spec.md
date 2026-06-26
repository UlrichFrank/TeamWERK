## MODIFIED Requirements

### Requirement: Feld-Auswahl beim CSV-Import (`fields`)

Das System SHALL ein optionales Formfeld `fields` für `POST /api/members/import` unterstützen, das eine komma-separierte Liste von DB-Spaltennamen enthält. Bei einem Update bestehender Mitglieder (Modi `update`, `enrich` sowie der Dry-Run `preview`) werden ausschließlich die in `fields` gelisteten Spalten verändert. Zulässige Spalten sind: `member_number`, `date_of_birth`, `gender`, `pass_number`, `position`, `status`, `home_club`, `jersey_number`, `street`, `zip`, `city`, `join_date`, `account_holder`, `sepa_mandat`, `iban`, `beitragsfrei`, `beitragsfrei_grund`. Ist `fields` leer oder nicht gesetzt, sind alle Spalten erlaubt (rückwärtskompatibel).

Die Whitelist greift NUR auf den Update-Pfad bestehender Mitglieder. Neu angelegte Mitglieder (`created`) werden unabhängig von `fields` mit allen vorhandenen CSV-Werten gespeichert.

`status`, `beitragsfrei` und `beitragsfrei_grund` sind drei voneinander unabhängige Whitelist-Einträge. Wer nur `beitragsfrei` whitelistet, ändert weder Status noch Grund. Die frühere Kopplung „Status-Auswahl steuert abgeleitetes beitragsfrei" entfällt.

#### Scenario: Nur ausgewählte Spalte wird aktualisiert
- **WHEN** ein bestehendes Mitglied im Modus `update` importiert wird, die CSV abweichende Werte für `IBAN` und `Status TeamWERK` enthält und `fields=iban` gesendet wird
- **THEN** aktualisiert das System nur `iban` und lässt `status` unverändert
- **THEN** listet der Report für die Zeile nur die IBAN-Änderung

#### Scenario: Leeres fields aktualisiert alle Spalten
- **WHEN** ein bestehendes Mitglied im Modus `update` importiert wird und `fields` leer oder nicht gesetzt ist
- **THEN** verhält sich der Import exakt wie ohne Whitelist und übernimmt alle nichtleeren, abweichenden CSV-Werte (inkl. `status`, `beitragsfrei`, `beitragsfrei_grund`)

#### Scenario: beitragsfrei separat whitelisten
- **WHEN** `fields=beitragsfrei` gesendet wird und die CSV abweichende Werte für `Status TeamWERK` und `beitragsfrei` enthält
- **THEN** wird nur `members.beitragsfrei` aktualisiert; `members.status` bleibt unverändert

#### Scenario: Grund ohne Flag whitelisten
- **WHEN** `fields=beitragsfrei_grund` gesendet wird und das CSV-Feld `Grund für Beitragsfreiheit` einen Wert enthält
- **THEN** wird nur `members.beitragsfrei_grund` aktualisiert (sofern Enrich-Regeln das nicht verbieten); `members.beitragsfrei` bleibt unverändert
- **AND** die Kopplungs-Invariante (`beitragsfrei=0` ⇒ `grund IS NULL`) wird respektiert: ist `beitragsfrei` in der DB bereits `0`, wird der Grund nicht gesetzt und die Zeile bleibt `unchanged`

#### Scenario: Neue Mitglieder ignorieren die Whitelist
- **WHEN** im Modus `update` ein in der DB nicht vorhandenes Mitglied importiert wird und `fields=iban` gesendet wird
- **THEN** legt das System den neuen Datensatz mit allen vorhandenen CSV-Feldern an (nicht nur IBAN)

### Requirement: Frontend-Auswahl im Import-Dialog

Das Frontend SHALL im Import-Dialog die Feld- und Mitglieder-Auswahl anbieten. In Schritt 1 (Datei + Modus) SHALL bei den Modi `update` und `enrich` eine Liste von Feld-Checkboxen erscheinen (standardmäßig alle ausgewählt); die ausgewählten Spalten werden als `fields` an Vorschau und Anwendung gesendet. In der Vorschau SHALL jede `updated`-Zeile eine Checkbox (standardmäßig angehakt) erhalten; beim Anwenden werden nur die angehakten Zeilennummern als `apply_lines` gesendet. Der Status `skipped` SHALL mit eigenem Icon, eigener Farbe und eigenem Summary-Badge dargestellt werden.

Die Feld-Checkboxen MUST `status`, `beitragsfrei` und `beitragsfrei_grund` als drei eigenständige Einträge führen. Die frühere kombinierte Checkbox „Status / Beitragsfrei" wird ersetzt.

#### Scenario: Feld-Checkboxen nur bei update/enrich
- **WHEN** der Nutzer den Modus `update` oder `enrich` wählt
- **THEN** zeigt der Dialog die Feld-Checkboxen (alle vorausgewählt), darunter `Status`, `Beitragsfrei` und `Grund für Beitragsfreiheit` als separate Einträge
- **WHEN** der Nutzer den Modus `append` wählt
- **THEN** werden keine Feld-Checkboxen angezeigt

#### Scenario: Abgewählte Felder werden nicht gesendet
- **WHEN** der Nutzer einzelne Feld-Checkboxen abwählt und die Vorschau startet
- **THEN** sendet das Frontend `fields` nur mit den verbliebenen Spalten

## REMOVED Requirements

### Requirement: Kombinierte Whitelist-Kategorie „Status / Beitragsfrei"

**Reason:** Mit dem neuen CSV-Schema sind `status`, `beitragsfrei` und `beitragsfrei_grund` drei voneinander unabhängige Quell-Spalten. Die bisherige Logik „Status-Auswahl steuert abgeleitetes beitragsfrei" ist hinfällig (siehe `members-csv-enrich-mode` REMOVED-Requirement „Ableitung von `beitragsfrei` aus der Status-Spalte").

**Migration:** Wo Frontend-Code oder API-Aufrufe `fields=status` mit der Erwartung gesendet haben, dass `beitragsfrei` mit aktualisiert wird, MUSS die Aufrufstelle künftig `fields=status,beitragsfrei[,beitragsfrei_grund]` senden. Standardmäßig setzt das Frontend alle drei Häkchen vor — dadurch ist der typische Update-Flow rückwärtskompatibel.
