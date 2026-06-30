## REMOVED Requirements

### Requirement: Voller Jahresbeitrag ohne anteilige Berechnung

**Grund:** Ersetzt durch die Halbierungsregel bei unterjährigem Ein-/Austritt und im ersten
Abrechnungsjahr (siehe ADDED „Halbierung des Jahresbeitrags …"). Der volle Jahresbeitrag
bleibt der Normalfall für ganzjährige Mitglieder, ist aber nicht mehr ausnahmslos.

## ADDED Requirements

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

## MODIFIED Requirements

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

### Requirement: Beitragsberechnung pro Mitglied
Der Beitragslauf MUST für jedes Mitglied anhand von `members.status` die Beitragsgruppe und den fälligen Jahresbeitrag zum **Stichtag 01.07. der Saison** bestimmen. Status `aktiv`/`verletzt` → Gruppe `aktiv` (Kategorie `aktiv_mit` bzw. `aktiv_ohne` je nach Stammverein); Status `pausiert`/`passiv` → Kategorie `passiv`. Ein im Saisonfenster `ausgetreten`-es Mitglied (unterjähriger Austritt) MUST wie ein aktives Mitglied kategorisiert werden (Kategorie aus `home_club_id`), da keine historische Statusverfolgung existiert. Für jede einzuziehende Kategorie MUST zum Stichtag ein gültiger Beitragssatz (`valid_from <= Stichtag`) existieren; fehlt er, wird das Mitglied mit Begründung `kein_beitragssatz` ausgeschlossen.

Die Beitragsmatrix MUST für die Kategorie `passiv` ab dem frühestmöglichen Saisonstart (01.07.2026) einen gültigen Satz enthalten, damit passive Mitglieder in der laufenden Saison erkannt und nicht fälschlich ausgeschlossen werden.

#### Scenario: Passives Mitglied in Saison 2026/27 wird einbezogen
- **WHEN** der Beitragslauf für eine Saison mit Start `2026-07-01` ausgeführt wird und ein Mitglied `status='passiv'` mit gültigem SEPA-Mandat, IBAN und vollständiger Adresse hat
- **THEN** wird das Mitglied mit Kategorie `passiv` einbezogen und **nicht** mit `kein_beitragssatz` ausgeschlossen

#### Scenario: Pausiertes Mitglied zählt als passiv
- **WHEN** ein Mitglied `status='pausiert'` im Lauf für Saison 2026/27 verarbeitet wird
- **THEN** wird es der Kategorie `passiv` zugeordnet und mit dem ab `2026-07-01` gültigen Passiv-Satz berechnet

### Requirement: Beitragslauf-Vorschau
Nutzer mit Vereinsfunktion `vorstand` oder `kassierer` (sowie System-Rolle `admin`) SHALL via `GET /api/fee-run/preview?saison_id=…` eine Vorschau aller Mitglieder einer Saison abrufen können. Die Antwort MUST pro Mitglied `member_id`, `name`, `status`, `included` sowie für eingeschlossene Mitglieder `kategorie` und `betrag_cent` enthalten, plus die Halbierungs-Felder `half` (bool) und — falls halbiert — `half_reason`. Die Antwort MUST ein `summary` mit `included_count`, `excluded_count`, `warned_count` und `total_cent` enthalten. Die Antwort MUSS das Fälligkeitsdatum als `faelligkeit` (01.07. der Saison) enthalten.

#### Scenario: Aktives Mitglied mit Stammverein
- **WHEN** ein Vorstand `GET /api/fee-run/preview?saison_id=42` aufruft und Mitglied M aktiv ist, `home_club_id` gesetzt und ganzjährig Mitglied ist (nicht `is_inaugural`)
- **THEN** erhält M `kategorie = "aktiv_mit"`, `included = true`, `half = false` und den vollen Jahresbeitrag der Kategorie `aktiv_mit`

#### Scenario: Zugriff ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `GET /api/fee-run/preview` aufruft
- **THEN** antwortet der Server mit HTTP 403
