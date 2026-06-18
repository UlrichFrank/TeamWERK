## Why

Mitgliedsbeiträge werden heute außerhalb von TeamWERK manuell abgerechnet. Der Verein soll künftig pro Saison einen SEPA-XML-Beitragslauf direkt aus der Mitgliederliste erzeugen können, der bei der BW-Bank im Online-Banking eingereicht wird.

Die Abrechnung ist bewusst einfach gehalten: Es gibt **keine anteilige Berechnung** — jedes einzuziehende Mitglied zahlt den vollen **Jahresbeitrag**, fällig **immer zum 01.07.** Spieler werden grundsätzlich als Kinder eingestuft; Volljährigkeit, Ausbildung, Beruf o. Ä. spielen für den Beitrag **keine Rolle**. Es wird **nicht** zwischen Erst- und Folgelastschrift unterschieden — alle Einzüge sind **RCUR**. Die einzige Beitragsabstufung ergibt sich aus Aktiv/Passiv-Status und der Stammverein-Zugehörigkeit (eine von 8 Vereinen).

Der **Kassierer** ist die Hauptrolle für den Beitragslauf: Er bekommt Lese-Zugriff auf die Mitglieder und darf deren **Bankdaten** (IBAN, SEPA-Mandat, Adresse) korrigieren, die SEPA-XML erzeugen und nach erfolgreicher Bank-Einreichung das Ergebnis **bestätigen**. Die Bestätigung schreibt ein **append-only Saison-Protokoll** (Textdatei pro Saisonjahr), das festhält, bei welchen Mitgliedern erfolgreich bzw. nicht erfolgreich eingezogen wurde, inkl. Betrag.

Das aktuelle Schema kennt zwar `iban`, `sepa_mandat`, `home_club` und `join_date`, aber weder die Gläubiger-ID des Vereins noch die Beitragsmatrix.

## What Changes

**DB-Migration (neu):**
- `clubs` erweitern: `glaeubiger_id TEXT`, `iban TEXT`, `bic TEXT`, `kontoinhaber TEXT`
- Neue Tabelle `beitrags_saetze` mit `(id, kategorie, betrag_eur, valid_from)` — 3 Kategorien
- Keine neuen `members`-Spalten (kein FRST/RCUR-Tracking nötig)

**Backend (neu):**
- `GET /api/club` und `PUT /api/club` erweitert um SEPA-Felder (`glaeubiger_id`, `iban`, `bic`, `kontoinhaber`)
- `GET /api/beitrags-saetze` (vorstand/kassierer/admin) — alle Sätze inkl. Historie
- `POST /api/beitrags-saetze` (vorstand/kassierer/admin) — neuen Satz mit `valid_from` anlegen
- `GET /api/beitragslauf/preview?saison_id=…` (vorstand/kassierer/admin) — Vorschau-Liste pro Mitglied: Kategorie, Jahresbeitrag, Ausschluss-/Warnungs-Begründung
- `POST /api/beitragslauf/export` (vorstand/kassierer/admin) — Body: `saison_id`, `member_ids` (vom UI angehakt). Antwort: SEPA-XML als Datei-Download
- `POST /api/beitragslauf/confirm` (vorstand/kassierer/admin) — Body: `saison_id`, `results: [{member_id, betrag_cent, success}]`. Hängt einen Eintrag an das Saison-Protokoll an (kein Überschreiben). Keine sonstige DB-Mutation.
- `GET /api/beitragslauf/protocol?saison_id=…` (vorstand/kassierer/admin) — gibt den Inhalt des Saison-Protokolls (Textdatei) zurück; für Anzeige/Download im UI

**Backend (geändert) — Kassierer-Zugriff auf Mitglieder:**
- Mitglieder-Lesen für `kassierer` freigeben: `GET /api/members`, `GET /api/members/{id}`, `GET /api/members/{id}/parents`, `GET /api/members/export` (von der Vorstand-only-Gruppe in eine `vorstand`+`kassierer`-Gruppe verschieben)
- Neuer Endpoint `PUT /api/members/{id}/bankdaten` (vorstand/kassierer) — aktualisiert **ausschließlich** die bankrelevanten Felder (`iban`, `sepa_mandat`, `sepa_mandat_date`, `account_holder`, `street`, `zip`, `city`). Keine Änderung an Name/Status/Rollen
- SEPA-Mandat-Datei: `POST /api/upload/sepa-mandat/{id}` und `DELETE /api/members/{id}/sepa-mandat` zusätzlich für `kassierer` freigeben
- Mitglieder anlegen/löschen, Status, Import, Rollen-/Family-Verwaltung bleiben `vorstand`-only

**Frontend:**
- `AdminSettingsPage`: VereinTab erweitert um SEPA-Felder (Gläubiger-ID, IBAN, BIC, Kontoinhaber); neuer Tab „Beiträge" mit Beitragsmatrix-Pflege (3 Kategorien × Betrag + valid_from, neue Zeile = neue valid_from-Version)
- Neue Seite `/admin/beitragslauf`:
  1. Saison wählen → Vorschau-Tabelle mit Checkbox pro Mitglied (Default angehakt außer ausgeschlossen), Spalten Name/Status/Kategorie/Betrag/Begründung, Summe
  2. „XML herunterladen" (Fälligkeit = 01.07. der Saison)
  3. Nach erfolgreicher Bank-Einreichung „Lauf bestätigen": Kassierer markiert ggf. einzelne Mitglieder als „nicht eingezogen" (Default alle erfolgreich) → `POST /confirm` schreibt das Saison-Protokoll fort. „Protokoll ansehen" zeigt die Textdatei
- Mitglieder-Bereich (`MembersPage`, `MemberDetailPage`): für `kassierer` sichtbar/erreichbar; Bankdaten-Bearbeitung (z. B. im `MemberDatenschutzTab`) für `kassierer` freigeschaltet, übrige Member-Felder nur lesbar

**Beitragsmatrix / Beitragslogik:**
- Abrechnungsjahr = Saisonjahr (01.07.YYYY – 30.06.(YYYY+1)); Fälligkeit immer 01.07.YYYY
- **Keine Pro-rata-Berechnung** — jedes einzuziehende Mitglied zahlt den vollen Jahresbeitrag, unabhängig vom Eintrittsdatum
- Kategorien: `aktiv_ohne`, `aktiv_mit`, `passiv`
- Status-Mapping: `aktiv`+`verletzt` → Aktivbeitrag; `pausiert`+`passiv` → Passivbeitrag; `ausgetreten`+`honorar`+`anwaerter` → kein Einzug
- **Keine Volljährigkeits-, Ausbildungs- oder Berufsprüfung** — Aktivbeitrag = Kinder-Satz für alle
- Stammverein-Zugehörigkeit: Fuzzy-Match auf 8-Vereine-Whitelist (hardcoded); unklare Treffer landen als Warnung in der Vorschau, kein automatischer Ausschluss → entscheidet `aktiv_mit` vs. `aktiv_ohne`
- Betrag = voller Jahresbeitrag laut Beitragssatz, der zum Saisonstart (01.07.) gilt

**SEPA-XML:**
- Schema: `pain.008.001.08` (CORE)
- **SeqTp immer `RCUR`** — keine Erst-/Folge-Unterscheidung, genau ein `PmtInf`-Block
- Mandatsreferenz = `members.member_number`
- Mandatsdatum = `members.sepa_mandat_date`
- Gläubiger-ID = `clubs.glaeubiger_id`
- Verwendungszweck = `"Jahresbeitrag Saison {saison_kurz} – Mitgliedsnr. {member_number}"`
- Kontoinhaber-Name im `<DbtrAcct>` = `members.account_holder` falls gesetzt, sonst `first_name + last_name`
- Adresse strukturiert (Pflicht in .08) aus `street`, `zip`, `city`

**Auswahl-/Validierungsregeln im Vorschau-Endpoint:**

Eingeschlossen (Default angehakt):
- Status in der jeweiligen Gruppe (siehe oben)
- `beitragsfrei = 0`
- `sepa_mandat = 1 AND sepa_mandat_path IS NOT NULL AND iban IS NOT NULL`
- `member_number IS NOT NULL`
- Adresse vollständig (`street`, `zip`, `city` alle NOT NULL/NOT BLANK)
- IBAN-Format gültig (Längen + Prüfsumme)

Ausgeschlossen (nicht angehakt, mit Begründung):
- Eine der obigen Bedingungen verletzt
- Status `ausgetreten`, `honorar`, `anwaerter`
- `beitragsfrei = 1`

Warnung (angehakt, aber Hinweis im UI):
- `home_club` setzt zwar einen Wert, der jedoch nicht eindeutig einem der 8 Mitgliedsvereine zuzuordnen ist → Vorstand entscheidet manuell, ob mit/ohne Stammverein

## Capabilities

### New Capabilities

- `sepa-beitragslauf`: Vorschau, Export und Bestätigung von SEPA-Lastschriftläufen für die Mitgliedsbeiträge einer Saison, inkl. pain.008.001.08-XML-Erzeugung und append-only Saison-Protokoll. Voller Jahresbeitrag pro Mitglied, keine anteilige Berechnung, immer RCUR.
- `beitrags-saetze`: Konfigurierbare Beitragsmatrix (3 Kategorien) mit Historie (`valid_from`) — pflegbar im Admin-UI
- `vereins-sepa-stammdaten`: Gläubiger-ID, IBAN, BIC, Kontoinhaber pro Verein im Admin-UI
- `kassierer-member-zugriff`: Kassierer erhält Lese-Zugriff auf Mitglieder und darf deren Bankdaten (IBAN, SEPA-Mandat, Adresse) korrigieren

### Modified Capabilities

- `club`: SEPA-Stammdaten in GET/PUT `/api/club`

## Impact

- `internal/db/migrations/043_sepa_beitragslauf.up.sql` (+`.down.sql`)
- `internal/config/config.go` — neue Konfig `BeitragslaufDir` (default `./storage/beitragslauf-protokolle`)
- `internal/config/handler.go` — SEPA-Felder in `/api/club`
- `internal/beitragslauf/` — neues Package: Auswahl-Logik, Kategorie-Bestimmung, XML-Builder (pain.008.001.08), Preview-/Export-/Confirm-/Protocol-Handler, Protokoll-Writer (append-only Textdatei)
- `internal/beitragssaetze/` — neues Package: CRUD für Beitragsmatrix
- `internal/members/handler.go` — neuer Endpoint `UpdateBankdaten` (Feld-Whitelist)
- `internal/app/router.go` — Member-Lese-Routen in `vorstand`+`kassierer`-Gruppe verschoben; `bankdaten`-, SEPA-Mandat-Routen für `kassierer`; Beitragslauf-Routen registriert
- `web/src/pages/AdminSettingsPage.tsx` — VereinTab erweitert, neuer BeitraegeTab
- `web/src/pages/admin/BeitragslaufPage.tsx` — neue Seite (inkl. Bestätigen + Protokoll-Anzeige)
- `web/src/pages/MembersPage.tsx` + `MemberDetailPage.tsx` + `components/admin/MemberDatenschutzTab.tsx` — Sichtbarkeit/Bearbeitbarkeit für `kassierer`
- `web/src/App.tsx` + `AppShell.tsx` — Route `/admin/beitragslauf`, Nav-Einträge (vorstand/kassierer/admin); Mitglieder-Nav auch für `kassierer`
- Keine neuen Backend-Dependencies (XML via `encoding/xml` aus stdlib, Protokoll via `os`)
- Frontend: keine neuen Dependencies

## Test-Anforderungen

- Route `GET /api/beitragslauf/preview`: TestPreview_AktivMitStammverein, TestPreview_AktivOhneStammverein, TestPreview_PassivVollerBeitrag, TestPreview_AusschlussOhneMandat, TestPreview_AusschlussOhneIBAN, TestPreview_WarnungUnklarerStammverein, TestPreview_BeitragsfreiAusgeschlossen, TestPreview_NeumitgliedZahltVollenBeitrag (kein Pro-rata), TestPreview_Forbidden (nicht-berechtigte Rolle → 403)
- Route `POST /api/beitragslauf/export`: TestExport_HappyPath (gültiges pain.008.001.08-XML), TestExport_EinPmtInfBlockRCUR, TestExport_VerwendungszweckFormat, TestExport_FehlendeStammdaten400, TestExport_ExcludedMember400, TestExport_Forbidden (403); TestExport_KassiererErlaubt (kassierer → 200)
- Route `POST /api/beitragslauf/confirm`: TestConfirm_HaengtProtokollAn (zweiter Aufruf ergänzt, überschreibt nicht), TestConfirm_ErfolgUndFehlerGetrennt (success/failed je gelistet), TestConfirm_Forbidden (403)
- Route `GET /api/beitragslauf/protocol`: TestProtocol_LiefertInhalt, TestProtocol_LeerWennKeinLauf (leer/404 sauber), TestProtocol_Forbidden (403)
- Route `GET/POST /api/beitrags-saetze`: TestSaetze_HistorieErhalten, TestSaetze_NeueValidFromAnlegen, TestSaetze_InvalidKategorie, TestSaetze_Forbidden (403)
- Route `GET /api/members` & `GET /api/members/{id}`: TestMembers_KassiererDarfLesen (200), TestMembers_SpielerVerboten (403)
- Route `PUT /api/members/{id}/bankdaten`: TestBankdaten_KassiererUpdatetNurBankfelder (Name/Status unverändert), TestBankdaten_Forbidden (403)
- Route `GET/PUT /api/club`: TestClub_SepaFelder_GetSet
- Invariante: Jedes eingeschlossene Mitglied wird mit dem vollen Jahresbeitrag (`betrag_cent` = `beitrags_saetze.betrag_eur` der gültigen Kategorie) abgerechnet — niemals anteilig
- Invariante: Jede Lastschrift im Export trägt `SeqTp = RCUR`; das XML enthält genau einen `PmtInf`-Block
- Invariante: Mitglieder mit `status IN ('ausgetreten','honorar','anwaerter')` oder `beitragsfrei = 1` tauchen nie im Export-XML auf
- Invariante: Erzeugtes XML validiert gegen das pain.008.001.08-XSD (mind. ein Schema-Validator-Test)
- Invariante: `POST /confirm` hängt nur an das Saison-Protokoll an — bestehende Einträge werden nie verändert oder gelöscht
- Invariante: `PUT /api/members/{id}/bankdaten` verändert ausschließlich die Bankfelder-Whitelist; alle übrigen Member-Felder bleiben unverändert
