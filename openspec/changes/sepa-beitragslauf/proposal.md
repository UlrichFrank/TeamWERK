## Why

Mitgliedsbeiträge werden heute außerhalb von TeamWERK manuell abgerechnet. Der Verein soll künftig pro Saison einen SEPA-XML-Beitragslauf direkt aus der Mitgliederliste erzeugen können, der bei der BW-Bank im Online-Banking eingereicht wird.

Die Beitragsordnung (Anlage 1, beschlossen 22.04.2026) definiert sechs Aktivbeitrags-Stufen, einen Passivbeitrag und einen Stichtag pro Beitragsart. Der Beitrag richtet sich nach Status, Volljährigkeit, Stammverein-Zugehörigkeit (eine von 8 Vereinen) und Ausbildungsstatus. Pro-rata-Berechnung beim Mitgliedschaftsbeginn und bei Stichtagswechseln ist erforderlich.

Das aktuelle Schema kennt zwar `iban`, `sepa_mandat`, `home_club` und `join_date`, aber weder die Gläubiger-ID des Vereins noch FRST/RCUR-Tracking, noch die Beitragsmatrix, noch ein Feld für „in Ausbildung".

## What Changes

**DB-Migration (neu):**
- `clubs` erweitern: `glaeubiger_id TEXT`, `iban TEXT`, `bic TEXT`, `kontoinhaber TEXT`
- `members` erweitern: `in_ausbildung INTEGER NOT NULL DEFAULT 0`, `last_sepa_einzug_am DATETIME`
- Neue Tabelle `beitrags_saetze` mit `(id, kategorie, betrag_eur, valid_from)`

**Backend (neu):**
- `GET /api/club` und `PUT /api/club` erweitert um SEPA-Felder (`glaeubiger_id`, `iban`, `bic`, `kontoinhaber`)
- `GET /api/beitrags-saetze` (vorstand/kassierer/admin) — alle Sätze inkl. Historie
- `POST /api/beitrags-saetze` (vorstand/kassierer/admin) — neuen Satz mit `valid_from` anlegen
- `GET /api/beitragslauf/preview?saison_id=…` (vorstand/kassierer/admin) — Vorschau-Liste pro Mitglied: Kategorie, Pro-rata-Monate, Betrag, SeqTp (FRST/RCUR), Ausschluss-/Warnungs-Begründung
- `POST /api/beitragslauf/export` (vorstand/kassierer/admin) — Body: `saison_id`, `member_ids` (vom UI angehakt). Antwort: SEPA-XML als Datei-Download
- `POST /api/beitragslauf/confirm-uploaded` (vorstand/kassierer/admin) — Body: `member_ids`. Setzt `last_sepa_einzug_am = now()` für die genannten Mitglieder
- `PUT /api/members/{id}/sepa-sequence-reset` (vorstand/kassierer/admin) — setzt `last_sepa_einzug_am = NULL` (für Bank-Reject-Fälle)

**Backend (geändert):**
- `GET /api/members/{id}` und `PUT /api/members/{id}` erweitert um `in_ausbildung`

**Frontend:**
- `AdminSettingsPage`: VereinTab erweitert um SEPA-Felder (Gläubiger-ID, IBAN, BIC, Kontoinhaber); neuer Tab „Beiträge" mit Beitragsmatrix-Pflege (alle 7 Kategorien × Betrag + valid_from, neue Zeile = neue valid_from-Version)
- Neue Seite `/admin/beitragslauf` mit zweistufigem Workflow:
  1. Saison wählen → Vorschau-Tabelle mit Checkbox pro Mitglied (Default angehakt außer ausgeschlossen), Spalten Name/Status/Kategorie/Monate/Betrag/SeqTp/Begründung, Summe
  2. „XML herunterladen" (Fälligkeit = heute + 7 Tage) → danach Button „Bei Bank hochgeladen bestätigen" zum Setzen von `last_sepa_einzug_am`
- `MemberDetailPage` erweitert um Toggle `in_ausbildung` (Vorstand/Admin)
- `MemberDetailPage` zeigt Button „SEPA-Sequenz zurücksetzen" (Vorstand/Kassierer/Admin), wenn `last_sepa_einzug_am IS NOT NULL`

**Beitragsmatrix / Pro-rata-Logik:**
- Abrechnungsjahr = Saisonjahr (01.07.YYYY – 30.06.(YYYY+1))
- Kategorien: `aktiv_volljaehrig_ohne`, `aktiv_volljaehrig_mit`, `aktiv_volljaehrig_ausb_ohne`, `aktiv_volljaehrig_ausb_mit`, `aktiv_minderj_ohne`, `aktiv_minderj_mit`, `passiv`
- Status-Mapping: `aktiv`+`verletzt` → Aktivbeitrag; `pausiert`+`passiv` → Passivbeitrag; `ausgetreten`+`honorar`+`anwaerter` → kein Einzug
- Volljährigkeit: Stichtag Saisonbeginn (01.07.); wer am 01.07. <18 ist, gilt die ganze Saison als minderjährig
- Stammverein-Zugehörigkeit: Fuzzy-Match auf 8-Vereine-Whitelist (hardcoded); unklare Treffer landen als Warnung in der Vorschau, kein automatischer Ausschluss
- Effective Start = `MAX(saisonstart, valid_from der Kategorie, join_date)`
- Monate = volle Kalendermonate ab `effective_start`-Folgemonat bis Saisonende (angefangene zählen NICHT)
- Betrag = `jahresbeitrag × monate / 12`, kaufmännisch gerundet auf 2 Nachkommastellen

**SEPA-XML:**
- Schema: `pain.008.001.08` (CORE, SeqTp pro Eintrag FRST oder RCUR)
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
- `last_sepa_einzug_am ≥ Saisonstart` → vermutlicher Doppellauf für diese Saison

## Capabilities

### New Capabilities

- `sepa-beitragslauf`: Vorschau, Export und Bestätigung von SEPA-Lastschriftläufen für die Mitgliedsbeiträge einer Saison, inkl. Pro-rata-Berechnung, FRST/RCUR-Tracking und pain.008.001.08-XML-Erzeugung
- `beitrags-saetze`: Konfigurierbare Beitragsmatrix mit Historie (`valid_from`) — pflegbar im Admin-UI
- `vereins-sepa-stammdaten`: Gläubiger-ID, IBAN, BIC, Kontoinhaber pro Verein im Admin-UI

### Modified Capabilities

- `members`: neues Feld `in_ausbildung` (Vorstand-pflegbar); neues Feld `last_sepa_einzug_am` (system-gepflegt, manuell rücksetzbar)
- `club`: SEPA-Stammdaten in GET/PUT `/api/club`

## Impact

- `internal/db/migrations/043_sepa_beitragslauf.up.sql` (+`.down.sql`)
- `internal/config/handler.go` — SEPA-Felder in `/api/club`
- `internal/members/handler.go` — `in_ausbildung`-Feld, `sepa-sequence-reset`-Endpoint
- `internal/beitragslauf/` — neues Package: Auswahl-Logik, Pro-rata-Berechnung, XML-Builder (pain.008.001.08), Preview-/Export-/Confirm-Handler
- `internal/beitragssaetze/` — neues Package: CRUD für Beitragsmatrix
- `web/src/pages/AdminSettingsPage.tsx` — VereinTab erweitert, neuer BeitraegeTab
- `web/src/pages/admin/BeitragslaufPage.tsx` — neue Seite
- `web/src/pages/MemberDetailPage.tsx` — `in_ausbildung`-Toggle, „SEPA-Sequenz zurücksetzen"-Button
- `web/src/App.tsx` + `AppShell.tsx` — Route `/admin/beitragslauf`, Nav-Eintrag (vorstand/kassierer/admin)
- Keine neuen Backend-Dependencies (XML via `encoding/xml` aus stdlib)
- Frontend: keine neuen Dependencies

## Test-Anforderungen

- Route `GET /api/beitragslauf/preview`: TestPreview_AktivVollMitStammverein, TestPreview_PassivProRata2026, TestPreview_AusschlussOhneMandat, TestPreview_AusschlussOhneIBAN, TestPreview_WarnungUnklarerStammverein, TestPreview_BeitragsfreiAusgeschlossen, TestPreview_ProRataNeumitgliedSeptember (angefangener Monat zählt nicht), TestPreview_Forbidden (nicht-berechtigte Rolle → 403)
- Route `POST /api/beitragslauf/export`: TestExport_HappyPath (gültiges pain.008.001.08-XML), TestExport_FRSTvsRCUR (Mischung), TestExport_VerwendungszweckFormat, TestExport_Forbidden (403)
- Route `POST /api/beitragslauf/confirm-uploaded`: TestConfirm_SetztLastEinzug, TestConfirm_NurAngegebeneMitglieder, TestConfirm_Forbidden (403)
- Route `PUT /api/members/{id}/sepa-sequence-reset`: TestReset_SetztAufNull, TestReset_Forbidden (403)
- Route `GET/POST /api/beitrags-saetze`: TestSaetze_HistorieErhalten, TestSaetze_NeueValidFromAnlegen, TestSaetze_Forbidden (403)
- Route `GET/PUT /api/club`: TestClub_SepaFelder_GetSet
- Invariante: Pro Mitglied + Saison wird höchstens einmal `last_sepa_einzug_am` durch Confirm gesetzt (Idempotenz)
- Invariante: Pro-rata-Betrag ist `jahresbeitrag × monate / 12`, kaufmännisch gerundet auf 2 Nachkommastellen, niemals > Jahresbeitrag
- Invariante: SeqTp = FRST ⟺ `last_sepa_einzug_am IS NULL` zum Lauf-Zeitpunkt
- Invariante: Mitglieder mit `status IN ('ausgetreten','honorar','anwaerter')` oder `beitragsfrei = 1` tauchen nie im Export-XML auf
- Invariante: Erzeugtes XML validiert gegen das pain.008.001.08-XSD (mind. ein Schema-Validator-Test)
