## ADDED Requirements

### Requirement: Beitragslauf-Vorschau
Nutzer mit Vereinsfunktion `vorstand` oder `kassierer` (sowie System-Rolle `admin`) SHALL via `GET /api/fee-run/preview?saison_id=…` eine Vorschau aller Mitglieder einer Saison abrufen können. Die Antwort MUST pro Mitglied `member_id`, `name`, `status`, `included` sowie für eingeschlossene Mitglieder `kategorie` und `betrag_cent` enthalten, plus ein `summary` mit `included_count`, `excluded_count`, `warned_count` und `total_cent`. Die Antwort MUSS das Fälligkeitsdatum als `faelligkeit` (01.07. der Saison) enthalten.

#### Scenario: Aktives Mitglied mit Stammverein
- **WHEN** ein Vorstand `GET /api/fee-run/preview?saison_id=42` aufruft und Mitglied M aktiv ist und `home_club` eindeutig einem der 8 Mitgliedsvereine zugeordnet wird
- **THEN** erhält M `kategorie = "aktiv_mit"`, `included = true` und den vollen Jahresbeitrag der Kategorie `aktiv_mit`

#### Scenario: Aktives Mitglied ohne Stammverein
- **WHEN** ein aktives Mitglied keinen oder keinen zuordenbaren `home_club` hat
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
Der Vorschau-Endpoint MUST Mitglieder mit `status IN ('ausgetreten','honorar','anwaerter')` oder `beitragsfrei = 1` ausschließen. Ebenso ausgeschlossen werden Mitglieder ohne gültiges SEPA-Mandat, ohne gültige IBAN, ohne Mitgliedsnummer oder mit unvollständiger Adresse. Jeder Ausschluss MUSS in `exclusions` mit Begründung gemeldet werden. Ein nicht eindeutig zuordenbarer `home_club` führt NICHT zum Ausschluss, sondern zu einer Warnung in `warnings`.

#### Scenario: Mitglied ohne SEPA-Mandat
- **WHEN** ein Mitglied `sepa_mandat = 0` hat
- **THEN** ist `included = false` und `exclusions` enthält `kein_sepa_mandat`

#### Scenario: Unklarer Stammverein
- **WHEN** ein Mitglied `home_club = "FC Bayern"` hat, der keinem Mitgliedsverein zuzuordnen ist
- **THEN** ist `included = true` mit `kategorie = "aktiv_ohne"` und `warnings` enthält `home_club_unklar`

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
