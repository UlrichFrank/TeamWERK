## ADDED Requirements

### Requirement: Unterstützte SEPA-XML-Schema-Version

Der Beitragslauf-Export MUST das SEPA-Nachrichtenformat **`pain.008.001.08`**
(SEPA-Basislastschrift, ISO 20022) erzeugen — Namespace
`urn:iso:std:iso:20022:tech:xsd:pain.008.001.08`. Diese ISO-Version ist laut
DFÜ-Abkommen Anlage 3 die verpflichtende SEPA-Lastschrift-Formatversion für
alle deutschen Banken; ältere ISO-Versionen (`pain.008.001.02`) laufen zum
30.11.2026 aus dem DK-LifeCycle, `pain.008.003.02` (ISO-2019-Vorabversion)
und `pain.008.001.10` (RTP-adjacent) sind für die klassische
SEPA-Basislastschrift nicht zulässig.

Innerhalb dieser ISO-Version MUST die erzeugte Datei gegen das aktuell
gültige DK-TVS (Technical Validation Subset) validieren:

- **Aktuell verpflichtend (ab 05.10.2025):** DK-TVS **GBIC_5** — XSD
  `pain.008.001.08_GBIC_5.xsd` aus Anlage 3 V3.9 / V26.11.
- **Übergangsweise weiter akzeptiert (End of LifeCycle 11/2026):** DK-TVS
  GBIC_4 (XSD `pain.008.001.08_GBIC_4.xsd`) aus Anlage 3 V3.7/V3.8. Der
  Generator MUSS **nicht** parallel GBIC_4 unterstützen; die GBIC_5-Ausgabe
  ist rückwärtskompatibel zu Banken, die noch GBIC_4 lesen.

Bei Erscheinen einer neuen DK-TVS-Version (GBIC_6+) SHALL die eingecheckte
XSD-Datei unter `web/src/lib/__schemas__/` gegen die neue Version ersetzt,
der Generator angepasst und diese Anforderung aktualisiert werden.

#### Scenario: Namespace der erzeugten Datei
- **WHEN** der Generator eine Beispiel-Datei aus repräsentativen Eingaben erzeugt
- **THEN** ist das Wurzelelement `<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pain.008.001.08">`

#### Scenario: Prüfung gegen aktuelle DK-TVS-Version
- **WHEN** die erzeugte Datei per `xmllint --noout --schema web/src/lib/__schemas__/pain.008.001.08_GBIC_5.xsd` validiert wird
- **THEN** endet der Aufruf mit Exit-Code 0

#### Scenario: Wechsel auf zukünftige DK-TVS-Version
- **WHEN** die DK eine neue TVS-Version (z.B. GBIC_6) veröffentlicht und diese die alte ersetzt
- **THEN** wird die alte XSD-Datei unter `web/src/lib/__schemas__/` durch die neue ersetzt und `sepaXml.xsd.test.ts` sowie diese Anforderung entsprechend aktualisiert

## MODIFIED Requirements

### Requirement: SEPA-XML-Export (pain.008.001.08), immer RCUR — clientseitig erzeugt

Die `pain.008.001.08`-Datei SHALL **im Browser** des berechtigten Nutzers
(`vorstand`/`kassierer`/`admin` mit entsperrtem Finance-Gruppenschlüssel)
erzeugt werden; der Server SHALL keine Klartext-IBANs verarbeiten. Der Server
SHALL via `POST /api/fee-run/export-data` mit Body `{saison_id, member_ids}`
die zur Erzeugung nötigen Daten ausschließlich als **Ciphertext + Group-Wraps**
ausliefern. Der Client SHALL die Blobs entschlüsseln, IBANs validieren, genau
einen `PmtInf`-Block mit `SeqTp = RCUR` und `ReqdColltnDt = 01.07.` der Saison
(Wochenende → nächster Werktag) bauen und die Datei lokal zum Download
anbieten. Alle übrigen fachlichen Invarianten (RCUR, ein PmtInf-Block,
Verwendungszweck, voller Jahresbeitrag) bleiben unverändert. Sind die
Vereins-SEPA-Stammdaten unvollständig oder enthält die Auswahl ein
ausgeschlossenes Mitglied, SHALL der Export abgewiesen werden.

Die erzeugte Datei MUST gegen das **DK-TVS `pain.008.001.08_GBIC_5.xsd`**
(Anlage 3 V26.11, gültig ab 05.10.2025) valide sein — nicht nur gegen das
lockerere ISO-Basis-XSD. Das XSD ist im Repo eingecheckt
(`web/src/lib/__schemas__/pain.008.001.08_GBIC_5.xsd`); ein CI-Test-Gate
(`sepaXml.xsd.test.ts`) validiert jede vom Generator erzeugte Beispieldatei
per `xmllint` gegen dieses XSD. Der Gate MUST in CI verpflichtend laufen
(Debian-Paket `libxml2-utils` in `ci.yml` installiert); lokal darf er
skippen, wenn `xmllint` fehlt (mit sichtbarem Warnhinweis).

Zusätzlich zur XSD-Konformität MUST der Generator vier DK-TVS-Härtungen
einhalten, die das XSD zwar prüft, deren Verletzung aber typischerweise erst
bei Bestandsdaten mit Kanten aufschlägt:

- Alle `<Nm>`-Elemente (Debtor, Creditor, Initiating Party) MUST auf 70
  Zeichen (nach ASCII-Normalisierung) begrenzt werden — DK-`Max140Text_SDD`
  ist auf 70 eingeschränkt.
- Der Verwendungszweck `<Ustrd>` MUST auf 140 Zeichen begrenzt werden.
- `<PstlAdr>` MUST all-or-nothing sein: entweder komplett mit `<TwnNm>` und
  `<Ctry>` (und optional Straße/Hausnummer/PLZ), oder komplett weggelassen.
  Ein `<PstlAdr>` ohne `<TwnNm>` ist unter GBIC_5 XSD-invalid.
- `<DtOfSgntr>` MUST bei fehlendem Mandatsdatum auf `2026-06-01`
  zurückfallen — ein weggelassenes Element ist unter GBIC_5 XSD-invalid,
  weil DtOfSgntr Pflichtelement in `MndtRltdInf` ist. Neu erfasste Mandate
  tragen weiter ihr echtes Signatur-Datum.

#### Scenario: XML wird clientseitig erzeugt
- **WHEN** ein Kassierer mit entsperrtem Gruppenschlüssel den Export auslöst
- **THEN** liefert der Server nur Ciphertext + Wraps; der Browser entschlüsselt, baut die pain.008.001.08-Datei und bietet sie zum Download an — der Server sieht keine Klartext-IBAN

#### Scenario: Erzeugtes XML validiert gegen das DK-TVS-XSD
- **WHEN** der Generator eine Beispiel-Datei aus repräsentativen Eingaben (Standard, fehlendes Mandatsdatum, fehlende Stadt, Multi-Transaktion) erzeugt
- **THEN** validiert `xmllint --noout --schema pain.008.001.08_GBIC_5.xsd <datei>` mit Exit-Code 0 gegen das eingecheckte DK-TVS

#### Scenario: CI-Gate blockiert schema-brüchige Änderungen
- **WHEN** ein PR den Generator so ändert, dass die erzeugte Datei das DK-TVS-XSD verletzt
- **THEN** schlägt der CI-Job `gate` beim `pnpm -C web test`-Schritt fehl, der `sepaXml.xsd.test.ts` läuft (nicht skipped), und der Merge ist blockiert

#### Scenario: Lokaler Entwickler ohne libxml2
- **WHEN** ein Entwickler ohne installiertes `xmllint` `pnpm test` lokal startet
- **THEN** wird der XSD-Gate-Test übersprungen und eine sichtbare `console.warn`-Meldung ausgegeben — die restliche Test-Suite läuft normal weiter

#### Scenario: Debitor-Name über 70 Zeichen
- **WHEN** ein Mitglied einen (nach ASCII-Normalisierung) über 70 Zeichen langen Namen hat
- **THEN** kürzt der Generator den `<Nm>`-Wert auf 70 Zeichen und die erzeugte Datei validiert gegen das GBIC_5-XSD

#### Scenario: Debitor ohne erfasste Stadt
- **WHEN** ein Mitglied kein `city`-Feld gepflegt hat
- **THEN** enthält `<Dbtr>` nur `<Nm>`, aber keinen `<PstlAdr>`-Block, und die Datei bleibt XSD-valide

#### Scenario: Debitor ohne erfasstes Mandatsdatum
- **WHEN** ein Mitglied kein `mandatDatum` gepflegt hat (Altbestand)
- **THEN** schreibt der Generator `<DtOfSgntr>2026-06-01</DtOfSgntr>` als Fallback, und die Datei bleibt XSD-valide

#### Scenario: Export ohne entsperrten Gruppenschlüssel
- **WHEN** ein berechtigter Nutzer den Export ohne entsperrten Finance-Gruppenschlüssel auslöst
- **THEN** wird zuerst die Schlüssel-/Passphrase-Eingabe verlangt; ohne sie entsteht keine Datei

#### Scenario: Export ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` die Export-Daten anfordert
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: CreDtTm mit UTC-Zeitzone
- **WHEN** der Browser die Datei erzeugt
- **THEN** enthält `<CreDtTm>` einen ISO-Timestamp mit angehängtem `Z` (z. B. `2026-07-15T17:18:03Z`)
