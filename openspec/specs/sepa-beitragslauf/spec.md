# sepa-beitragslauf Specification

## Purpose
SEPA-Beitragslauf für den jährlichen Einzug der Mitgliedsbeiträge: Vorschau, deterministische Berechnung pro Mitglied, XML-Export (pain.008.001.08, immer RCUR) und append-only Saison-Protokoll.

## Requirements

### Requirement: Beitragslauf-Vorschau
Nutzer mit Vereinsfunktion `vorstand` oder `kassierer` (sowie System-Rolle `admin`) SHALL via `GET /api/fee-run/preview?saison_id=…` eine Vorschau aller Mitglieder einer Saison abrufen können. Die Antwort MUST pro Mitglied `member_id`, `name`, `status`, `included` sowie für eingeschlossene Mitglieder `kategorie` und `betrag_cent` enthalten, plus ein `summary` mit `included_count`, `excluded_count`, `warned_count` und `total_cent`. Die Antwort MUSS das Fälligkeitsdatum als `faelligkeit` (01.07. der Saison) enthalten.

#### Scenario: Aktives Mitglied mit Stammverein
- **WHEN** ein Vorstand `GET /api/fee-run/preview?saison_id=42` aufruft und Mitglied M aktiv ist und `home_club_id` gesetzt ist
- **THEN** erhält M `kategorie = "aktiv_mit"`, `included = true` und den vollen Jahresbeitrag der Kategorie `aktiv_mit`

#### Scenario: Aktives Mitglied ohne Stammverein
- **WHEN** ein aktives Mitglied `home_club_id = NULL` hat
- **THEN** erhält es `kategorie = "aktiv_ohne"` und den vollen Jahresbeitrag der Kategorie `aktiv_ohne`

#### Scenario: Zugriff ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `GET /api/fee-run/preview` aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Voller Jahresbeitrag ohne anteilige Berechnung
Der Beitragslauf MUST jedes eingeschlossene Mitglied mit dem vollen Jahresbeitrag des zum Saisonstart (01.07.) gültigen Beitragssatzes abrechnen. Es DARF KEINE anteilige (Pro-rata-)Berechnung anhand des Eintrittsdatums erfolgen. Volljährigkeit, Ausbildung und Beruf DÜRFEN für die Beitragshöhe NICHT berücksichtigt werden; aktive Spieler werden grundsätzlich mit dem Aktiv-(Kinder-)Satz abgerechnet.

#### Scenario: Neumitglied mitten in der Saison
- **WHEN** ein Mitglied mit `join_date = 2026-09-15` in der Saison mit Start `2026-07-01` einbezogen wird
- **THEN** ist `betrag_cent` der volle Jahresbeitrag der Kategorie, ohne anteiligen Abzug

#### Scenario: Beitrag gleich Beitragssatz
- **WHEN** ein eingeschlossenes Mitglied der Kategorie K zugeordnet ist
- **THEN** entspricht `betrag_cent` exakt `beitrags_saetze.betrag_eur` des für K zum Saisonstart gültigen Satzes

### Requirement: Ein- und Ausschlussregeln
Der Vorschau-Endpoint MUST Mitglieder mit `status IN ('ausgetreten','honorar','anwaerter')` oder `beitragsfrei = 1` ausschließen. Ebenso ausgeschlossen werden Mitglieder ohne gültiges SEPA-Mandat, ohne gültige IBAN, ohne Mitgliedsnummer oder mit unvollständiger Adresse. Jeder Ausschluss MUSS in `exclusions` mit Begründung gemeldet werden. Ein fehlender Stammverein (`home_club_id = NULL`) führt NICHT zum Ausschluss, sondern regulär zur Kategorie `aktiv_ohne`.

#### Scenario: Mitglied ohne SEPA-Mandat
- **WHEN** ein Mitglied `sepa_mandat = 0` hat
- **THEN** ist `included = false` und `exclusions` enthält `kein_sepa_mandat`

### Requirement: SEPA-XML-Export (pain.008.001.08), immer RCUR
Berechtigte Nutzer SOLLEN via `POST /api/fee-run/export` mit Body `{saison_id, member_ids}` eine SEPA-XML-Datei im Schema `pain.008.001.08` herunterladen können. Alle Lastschriften MUST `SeqTp = RCUR` verwenden; es wird NICHT zwischen Erst- und Folgelastschrift unterschieden. Das XML besteht daher aus genau einem `PmtInf`-Block, setzt das Fälligkeitsdatum (`ReqdColltnDt`) auf den 01.07. der Saison (bei Wochenende auf den nächsten Werktag) und führt den Verwendungszweck `Jahresbeitrag Saison {saison_kurz} – Mitgliedsnr. {member_number}`. Sind die Vereins-SEPA-Stammdaten unvollständig, MUSS der Server mit HTTP 400 antworten. Enthält `member_ids` ein ausgeschlossenes Mitglied, MUSS der Server mit HTTP 400 antworten.

#### Scenario: Gültiges XML
- **WHEN** ein Vorstand `POST /api/fee-run/export` mit gültiger Saison und eingeschlossenen `member_ids` aufruft
- **THEN** liefert der Server `application/xml`, das gegen das pain.008.001.08-XSD validiert

#### Scenario: Genau ein PmtInf-Block mit RCUR
- **WHEN** das XML erzeugt wird
- **THEN** enthält es genau einen `PmtInf`-Block mit `SeqTp = RCUR`, unabhängig davon, ob ein Mitglied zum ersten Mal eingezogen wird

#### Scenario: Fehlende Stammdaten
- **WHEN** die Gläubiger-ID des Vereins nicht gesetzt ist
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Export ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `POST /api/fee-run/export` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Kassierer darf exportieren
- **WHEN** ein Nutzer mit `club_functions: ["kassierer"]` `POST /api/fee-run/export` mit gültiger Auswahl aufruft
- **THEN** liefert der Server die XML-Datei (HTTP 200)

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
Der Beitragslauf MUST für jedes Mitglied anhand von `members.status` die Beitragsgruppe und den fälligen Jahresbeitrag zum **Stichtag 01.07. der Saison** bestimmen. Status `aktiv`/`verletzt` → Gruppe `aktiv` (Kategorie `aktiv_mit` bzw. `aktiv_ohne` je nach Stammverein); Status `pausiert`/`passiv` → Kategorie `passiv`. Für jede einzuziehende Kategorie MUST zum Stichtag ein gültiger Beitragssatz (`valid_from <= Stichtag`) existieren; fehlt er, wird das Mitglied mit Begründung `kein_beitragssatz` ausgeschlossen.

Die Beitragsmatrix MUST für die Kategorie `passiv` ab dem frühestmöglichen Saisonstart (01.07.2026) einen gültigen Satz enthalten, damit passive Mitglieder in der laufenden Saison erkannt und nicht fälschlich ausgeschlossen werden.

#### Scenario: Passives Mitglied in Saison 2026/27 wird einbezogen
- **WHEN** der Beitragslauf für eine Saison mit Start `2026-07-01` ausgeführt wird und ein Mitglied `status='passiv'` mit gültigem SEPA-Mandat, IBAN und vollständiger Adresse hat
- **THEN** wird das Mitglied mit Kategorie `passiv` und Betrag 6000 ct (60 €) einbezogen und **nicht** mit `kein_beitragssatz` ausgeschlossen

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
