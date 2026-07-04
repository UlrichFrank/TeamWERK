## MODIFIED Requirements

### Requirement: SEPA-XML-Export (pain.008.001.08), immer RCUR — clientseitig erzeugt

Die `pain.008.001.08`-Datei SHALL **im Browser** des berechtigten Nutzers
(`vorstand`/`kassierer`/`admin` mit entsperrtem Finance-Gruppenschlüssel) erzeugt werden;
der Server SHALL keine Klartext-IBANs verarbeiten. Der Server SHALL via
`POST /api/fee-run/export-data` mit Body `{saison_id, member_ids}` die zur Erzeugung
nötigen Daten ausschließlich als **Ciphertext + Group-Wraps** ausliefern (Mitglieds-Blobs,
Vereins-SEPA-Stammdaten-Blob, Beträge, Verwendungszweck-Bausteine). Der Client SHALL die
Blobs entschlüsseln, IBANs validieren, den genau einen `PmtInf`-Block mit `SeqTp = RCUR`
und `ReqdColltnDt = 01.07.` der Saison (Wochenende → nächster Werktag) bauen und die Datei
lokal zum Download anbieten. Alle übrigen fachlichen Invarianten (RCUR, ein PmtInf-Block,
Verwendungszweck, voller Jahresbeitrag) bleiben unverändert. Sind die Vereins-SEPA-
Stammdaten unvollständig oder enthält die Auswahl ein ausgeschlossenes Mitglied, SHALL der
Export (clientseitig nach Entschlüsselung bzw. serverseitig anhand nicht-verschlüsselter
Felder) abgewiesen werden.

#### Scenario: XML wird clientseitig erzeugt
- **WHEN** ein Kassierer mit entsperrtem Gruppenschlüssel den Export auslöst
- **THEN** liefert der Server nur Ciphertext + Wraps; der Browser entschlüsselt, baut die
  pain.008.001.08-Datei und bietet sie zum Download an — der Server sieht keine Klartext-IBAN

#### Scenario: Erzeugtes XML validiert gegen das XSD
- **WHEN** der Browser die Datei aus den entschlüsselten Daten erzeugt
- **THEN** validiert sie gegen das pain.008.001.08-XSD und enthält genau einen `PmtInf`-Block mit `SeqTp = RCUR`

#### Scenario: Export ohne entsperrten Gruppenschlüssel
- **WHEN** ein berechtigter Nutzer den Export ohne entsperrten Finance-Gruppenschlüssel auslöst
- **THEN** wird zuerst die Schlüssel-/Passphrase-Eingabe verlangt; ohne sie entsteht keine Datei

#### Scenario: Export ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` die Export-Daten anfordert
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Ein- und Ausschlussregeln

Der Vorschau-Endpoint MUST Mitglieder mit `status IN ('ausgetreten','honorar','anwaerter')`
oder `beitragsfrei = 1` ausschließen, ebenso Mitglieder ohne gültiges SEPA-Mandat
(`sepa_mandat = 0`), ohne Mitgliedsnummer oder mit unvollständiger Adresse — diese
Prüfungen erfolgen serverseitig anhand **nicht-verschlüsselter** Felder. Die Ausschlüsse
**IBAN fehlt** (`iban_fehlt`) und **IBAN ungültig** (`iban_ungueltig`) SHALL hingegen
**clientseitig** nach Entschlüsselung der IBAN ermittelt werden, da der Server die IBAN
nicht mehr im Klartext kennt. Jeder Ausschluss MUSS dem Nutzer mit Begründung angezeigt
werden (server-gemeldete + clientseitig ergänzte).

#### Scenario: Nicht-IBAN-Ausschluss kommt vom Server
- **WHEN** ein Mitglied `sepa_mandat = 0` hat
- **THEN** meldet die Server-Vorschau `included = false` mit `exclusions` enthält `kein_sepa_mandat`

#### Scenario: IBAN-Ausschluss wird clientseitig ergänzt
- **WHEN** der Browser die IBAN eines sonst eingeschlossenen Mitglieds entschlüsselt und sie
  fehlt oder die Prüfziffer ungültig ist
- **THEN** markiert der Client das Mitglied clientseitig als ausgeschlossen
  (`iban_fehlt`/`iban_ungueltig`) und nimmt es nicht in die erzeugte XML-Datei auf
