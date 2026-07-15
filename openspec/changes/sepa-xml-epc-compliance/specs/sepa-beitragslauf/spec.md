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

Zusätzlich zur reinen XSD-Validität MUST die erzeugte Datei folgende EPC-SEPA-Rulebook-
und bankseitigen Konventionen erfüllen, damit deutsche Banken (u. a. LBBW/BW-Bank) den
Online-Upload akzeptieren:

- `<CreDtTm>` MUST einen expliziten UTC-Zonenmarker tragen (Suffix `Z`).
- `<Cdtr>` MUST zusätzlich zu `<Nm>` ein `<PstlAdr>` mit mindestens `<Ctry>DE</Ctry>`
  enthalten.
- Die Gläubiger-ID unter `<InitgPty>/<Id>/<OrgId>/<Othr>` MUST wie unter `<CdtrSchmeId>`
  mit `<SchmeNm><Prtry>SEPA</Prtry></SchmeNm>` annotiert sein.

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

#### Scenario: CreDtTm mit UTC-Zeitzone
- **WHEN** der Browser die Datei erzeugt
- **THEN** enthält `<CreDtTm>` einen ISO-Timestamp mit angehängtem `Z` (z. B. `2026-07-15T17:18:03Z`)

#### Scenario: Creditor-Postal-Adresse mit Country
- **WHEN** der Browser den `<PmtInf>`-Block schreibt
- **THEN** enthält `<Cdtr>` nach `<Nm>` ein `<PstlAdr><Ctry>DE</Ctry></PstlAdr>`

#### Scenario: InitgPty-Gläubiger-ID mit SEPA-SchmeNm
- **WHEN** der Browser den `<GrpHdr>`-Block schreibt
- **THEN** enthält `<InitgPty>/<Id>/<OrgId>/<Othr>` sowohl `<Id>` (Gläubiger-ID) als auch
  `<SchmeNm><Prtry>SEPA</Prtry></SchmeNm>`
