## MODIFIED Requirements

### Requirement: Ein- und Ausschlussregeln

Der Vorschau-Endpoint MUST Mitglieder mit `status IN ('honorar','anwaerter','foerderkind')`
oder `beitragsfrei = 1` ausschließen, ebenso Mitglieder ohne gültiges SEPA-Mandat
(`sepa_mandat = 0`), ohne Mitgliedsnummer oder mit unvollständiger Adresse — diese
Prüfungen erfolgen serverseitig anhand **nicht-verschlüsselter** Felder. Mitglieder mit
`status = 'ausgetreten'` MUST ausgeschlossen werden, **es sei denn** ihr `exit_date` liegt
im Saisonfenster `[start_date, end_date]` — dann werden sie als unterjähriger Austritt
**einbezogen** (Kategorie aus `home_club_id` wie bei aktiven Mitgliedern, Beitrag halbiert).
Die Ausschlüsse **IBAN fehlt** (`iban_fehlt`) und **IBAN ungültig** (`iban_ungueltig`) SHALL
**clientseitig** nach Entschlüsselung der IBAN ermittelt werden. Jeder Ausschluss MUST dem
Nutzer mit Begründung angezeigt werden. Ein fehlender Stammverein (`home_club_id = NULL`)
führt NICHT zum Ausschluss, sondern regulär zur Kategorie `aktiv_ohne`.

#### Scenario: Förderkind wird ausgeschlossen
- **WHEN** ein Mitglied mit `status = 'foerderkind'` im Beitragslauf verarbeitet wird
- **THEN** wird es `included = false` mit `exclusions` enthält `status_inaktiv` (bzw. die
  bestehende Status-Ausschluss-Begründung) und erscheint nicht in den Summen

#### Scenario: Nicht-IBAN-Ausschluss kommt vom Server
- **WHEN** ein Mitglied `sepa_mandat = 0` hat
- **THEN** meldet die Server-Vorschau `included = false` mit `exclusions` enthält `kein_sepa_mandat`

#### Scenario: Früher ausgetretenes Mitglied bleibt ausgeschlossen
- **WHEN** ein Mitglied `status = 'ausgetreten'` mit `exit_date` **vor** dem Saisonstart (oder
  ohne `exit_date`) verarbeitet wird
- **THEN** wird es ausgeschlossen (`status_inaktiv`) und erscheint nicht in den Summen

#### Scenario: Unterjähriger Austritt wird einbezogen
- **WHEN** ein Mitglied `status = 'ausgetreten'` mit `exit_date` im Saisonfenster, gültigem
  SEPA-Mandat, IBAN-Envelope, Mitgliedsnummer und vollständiger Adresse verarbeitet wird
- **THEN** wird es `included = true` mit halbiertem Betrag und `half_reason = "austritt"`
