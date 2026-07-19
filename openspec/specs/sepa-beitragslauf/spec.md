# sepa-beitragslauf Specification

## Purpose
SEPA-Beitragslauf für den jährlichen Einzug der Mitgliedsbeiträge: Vorschau, deterministische Berechnung pro Mitglied, XML-Export (pain.008.001.08, immer RCUR) und append-only Saison-Protokoll.
## Requirements
### Requirement: Beitragslauf-Vorschau
Nutzer mit Vereinsfunktion `vorstand` oder `kassierer` (sowie System-Rolle `admin`) SHALL via `GET /api/fee-run/preview?saison_id=…` eine Vorschau aller Mitglieder einer Saison abrufen können. Die Antwort MUST pro Mitglied `member_id`, `name`, `status`, `included` sowie für eingeschlossene Mitglieder `kategorie` und `betrag_cent` enthalten, plus die Halbierungs-Felder `half` (bool) und — falls halbiert — `half_reason`. Die Antwort MUST ein `summary` mit `included_count`, `excluded_count`, `warned_count` und `total_cent` enthalten. Die Antwort MUSS das Fälligkeitsdatum als `faelligkeit` (01.07. der Saison) enthalten.

#### Scenario: Aktives Mitglied mit Stammverein
- **WHEN** ein Vorstand `GET /api/fee-run/preview?saison_id=42` aufruft und Mitglied M aktiv ist, `home_club_id` gesetzt und ganzjährig Mitglied ist (nicht `is_inaugural`)
- **THEN** erhält M `kategorie = "aktiv_mit"`, `included = true`, `half = false` und den vollen Jahresbeitrag der Kategorie `aktiv_mit`

#### Scenario: Zugriff ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `GET /api/fee-run/preview` aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Ein- und Ausschlussregeln

Der Vorschau-Endpoint MUST Mitglieder mit `status IN ('honorar','anwaerter')`
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

### Requirement: Lauf bestätigen und Saison-Protokoll
Berechtigte Nutzer (`vorstand`, `kassierer`, `admin`) SHALL nach erfolgreicher Bank-Einreichung via `POST /api/fee-run/confirm` mit Body `{saison_id, results: [{member_id, betrag_cent, success}]}` das Ergebnis bestätigen. Die Bestätigung MUST einen Block an eine append-only Textdatei pro Saisonjahr anhängen, die für jeden Lauf festhält, bei welchen Mitgliedern erfolgreich bzw. nicht erfolgreich eingezogen wurde (inkl. Betrag), Zeitpunkt und bestätigendem Nutzer. Bestehende Blöcke DÜRFEN NICHT verändert oder gelöscht werden. Der Confirm DARF KEINE Mitgliederdaten ändern. Via `GET /api/fee-run/protocol?saison_id=…` SHALL der Protokoll-Inhalt als `text/plain` abrufbar sein.

#### Scenario: Bestätigung hängt an, überschreibt nicht
- **WHEN** ein Kassierer zweimal `POST /api/fee-run/confirm` für dieselbe Saison aufruft
- **THEN** enthält die Protokolldatei zwei Blöcke und der erste Block bleibt unverändert

#### Scenario: Erfolgreich und nicht erfolgreich getrennt
- **WHEN** ein Confirm `results` mit `success: true` und `success: false` enthält
- **THEN** listet der angehängte Block beide Gruppen getrennt, und die ausgewiesene Summe zählt nur die erfolgreichen Einzüge

#### Scenario: Protokoll abrufen
- **WHEN** nach mindestens einer Bestätigung `GET /api/fee-run/protocol?saison_id=…` aufgerufen wird
- **THEN** liefert der Server den Textinhalt der Saison-Protokolldatei

#### Scenario: Confirm ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `POST /api/fee-run/confirm` aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Beitragsberechnung pro Mitglied
Der Beitragslauf MUST für jedes Mitglied anhand von `members.status` die Beitragsgruppe und den fälligen Jahresbeitrag zum **Stichtag 01.07. der Saison** bestimmen. Status `aktiv`/`verletzt` → Gruppe `aktiv` (Kategorie `aktiv_mit` bzw. `aktiv_ohne` je nach Stammverein); Status `pausiert`/`passiv` → Kategorie `passiv`. Ein im Saisonfenster `ausgetreten`-es Mitglied (unterjähriger Austritt) MUST wie ein aktives Mitglied kategorisiert werden (Kategorie aus `home_club_id`), da keine historische Statusverfolgung existiert. Für jede einzuziehende Kategorie MUST zum Stichtag ein gültiger Beitragssatz (`valid_from <= Stichtag`) existieren; fehlt er, wird das Mitglied mit Begründung `kein_beitragssatz` ausgeschlossen.

Die Beitragsmatrix MUST für die Kategorie `passiv` ab dem frühestmöglichen Saisonstart (01.07.2026) einen gültigen Satz enthalten, damit passive Mitglieder in der laufenden Saison erkannt und nicht fälschlich ausgeschlossen werden.

#### Scenario: Passives Mitglied in Saison 2026/27 wird einbezogen
- **WHEN** der Beitragslauf für eine Saison mit Start `2026-07-01` ausgeführt wird und ein Mitglied `status='passiv'` mit gültigem SEPA-Mandat, IBAN und vollständiger Adresse hat
- **THEN** wird das Mitglied mit Kategorie `passiv` einbezogen und **nicht** mit `kein_beitragssatz` ausgeschlossen

#### Scenario: Pausiertes Mitglied zählt als passiv
- **WHEN** ein Mitglied `status='pausiert'` im Lauf für Saison 2026/27 verarbeitet wird
- **THEN** wird es der Kategorie `passiv` zugeordnet und mit dem ab `2026-07-01` gültigen Passiv-Satz berechnet

### Requirement: Aktiv-Kategorie aus Stammverein-Zuordnung
Der Beitragslauf MUST die Aktiv-Kategorie eines Mitglieds **deterministisch** aus `members.home_club_id` ableiten: ist ein Stammverein zugeordnet (`home_club_id IS NOT NULL`) → Kategorie `aktiv_mit`; sonst → `aktiv_ohne`. Es MUST **kein** Fuzzy-/Freitext-Abgleich (`MatchHomeClub`) mehr stattfinden, und es MUST keine `home_club_unklar`-Warnung mehr erzeugt werden. „Kein Stammverein" (`NULL`) ist ein gültiger Zustand und führt regulär zu `aktiv_ohne`.

#### Scenario: Mitglied mit zugeordnetem Stammverein
- **WHEN** ein aktives Mitglied mit gesetztem `home_club_id` im Lauf verarbeitet wird
- **THEN** wird es der Kategorie `aktiv_mit` zugeordnet (96 €) — unabhängig von Schreibweise, da keine Textzuordnung mehr erfolgt

#### Scenario: Mitglied ohne Stammverein
- **WHEN** ein aktives Mitglied mit `home_club_id = NULL` im Lauf verarbeitet wird
- **THEN** wird es der Kategorie `aktiv_ohne` zugeordnet (226 €), ohne Warnung

#### Scenario: Keine Fuzzy-Warnung mehr
- **WHEN** der Lauf-Preview für aktive Mitglieder erzeugt wird
- **THEN** enthält kein Mitglied die Warnung `home_club_unklar`

### Requirement: Halbierung des Jahresbeitrags bei unterjährigem Ein-/Austritt und im ersten Abrechnungsjahr

Der Beitragslauf MUST den Jahresbeitrag eines eingeschlossenen Mitglieds **exakt halbieren**
(`betrag_cent / 2`, ganzzahlig), wenn **mindestens eine** der folgenden Bedingungen zutrifft;
sonst MUST der volle Jahresbeitrag berechnet werden. Die Ermäßigungen DÜRFEN NICHT stapeln —
es wird höchstens halbiert, niemals geviertelt. Es DARF KEINE monatsgenaue (Pro-rata-)
Berechnung erfolgen.

- **Eintritt:** `members.join_date` liegt im Saisonfenster `[start_date, end_date]` (inklusive).
- **Austritt:** `members.status = 'ausgetreten'` und `members.exit_date` liegt im Saisonfenster.
- **Erstes Abrechnungsjahr:** Die Saison ist mit `seasons.is_inaugural = 1` markiert (einmalige
  Startkonzession) → **alle** eingeschlossenen Mitglieder zahlen halb.

Die Vorschau MUST pro Mitglied ausweisen, ob halbiert wurde (`half`) und aus welchem Grund
(`half_reason ∈ {"erstjahr","eintritt","austritt"}`, Priorität in dieser Reihenfolge).

#### Scenario: Eintritt mitten in der Saison zahlt halb
- **WHEN** ein Mitglied mit `join_date = 2027-09-15` in der Saison mit Start `2027-07-01` und
  Ende `2028-06-30` (nicht `is_inaugural`) einbezogen wird
- **THEN** ist `betrag_cent` exakt der halbe Jahresbeitrag der Kategorie, `half = true`,
  `half_reason = "eintritt"`

#### Scenario: Ganzjähriges Mitglied zahlt voll
- **WHEN** ein Mitglied mit `join_date` vor `start_date`, ohne Austritt, in einer nicht-
  `is_inaugural`-Saison einbezogen wird
- **THEN** ist `betrag_cent` der volle Jahresbeitrag und `half = false`

#### Scenario: Erstes Abrechnungsjahr halbiert alle
- **WHEN** der Lauf für eine Saison mit `is_inaugural = 1` erzeugt wird
- **THEN** zahlt jedes eingeschlossene Mitglied den halben Beitrag mit `half_reason = "erstjahr"`,
  unabhängig von `join_date`/`exit_date`

#### Scenario: Ein- und Austritt im selben Jahr halbiert nur einmal
- **WHEN** ein Mitglied sowohl `join_date` als auch `exit_date` im selben Saisonfenster hat
- **THEN** wird der Beitrag genau einmal halbiert (nicht geviertelt)

### Requirement: Eintrittsdatum verpflichtend, Austrittsdatum bei Austritt verpflichtend

Beim Anlegen oder Bearbeiten eines Mitglieds MUST `join_date` (Eintrittsdatum) angegeben
sein; fehlt es, MUST der Server mit HTTP 400 antworten. Wird der Status auf `ausgetreten`
gesetzt, MUST `exit_date` (Austrittsdatum) angegeben sein; fehlt es, MUST der Server mit
HTTP 400 antworten. Bestandsmitglieder ohne `join_date` MUST per Migration ein implizites
Eintrittsdatum **vor** dem ersten regulären Saisonstart erhalten, damit die Eintritts-
Halbierung für sie nicht greift. Die DB-Spalten DÜRFEN nullbar bleiben (Pflicht nur in der
App-Validierung).

#### Scenario: Anlegen ohne Eintrittsdatum
- **WHEN** ein Vorstand ein Mitglied ohne `join_date` anlegt
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Austritt ohne Austrittsdatum
- **WHEN** ein Vorstand den Status eines Mitglieds auf `ausgetreten` setzt, ohne `exit_date`
  anzugeben
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Bestandsmitglied bleibt voller Beitrag
- **WHEN** ein per Backfill mit implizitem `join_date` versehenes Bestandsmitglied in einer
  nicht-`is_inaugural`-Folgesaison verarbeitet wird
- **THEN** zahlt es den vollen Jahresbeitrag (`half = false`)

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

