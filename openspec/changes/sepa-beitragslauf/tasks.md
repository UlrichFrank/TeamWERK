# Tasks: SEPA-Beitragslauf

## 1. Datenbank-Migration

- [ ] 1.1 `internal/db/migrations/043_sepa_beitragslauf.up.sql` anlegen:
  - `ALTER TABLE clubs ADD COLUMN glaeubiger_id TEXT;`
  - `ALTER TABLE clubs ADD COLUMN iban TEXT;`
  - `ALTER TABLE clubs ADD COLUMN bic TEXT;`
  - `ALTER TABLE clubs ADD COLUMN kontoinhaber TEXT;`
  - `ALTER TABLE members ADD COLUMN in_ausbildung INTEGER NOT NULL DEFAULT 0;`
  - `ALTER TABLE members ADD COLUMN last_sepa_einzug_am DATETIME;`
  - `CREATE TABLE beitrags_saetze (…)` gemäß design.md §1.3 inkl. CHECK-Constraint, Index
  - Seed-INSERTs für die 7 Kategorien mit `INSERT OR IGNORE` und `valid_from` aus Gebührenordnung
  - Commit: `chore(db): Migration 043 — SEPA-Stammdaten, Beitragsmatrix, FRST/RCUR-Tracking`
- [ ] 1.2 Korrespondierende `.down.sql` mit `DROP TABLE beitrags_saetze;` und Spalten-Drop via Tabellen-Recreate (SQLite hat kein `DROP COLUMN` vor 3.35; lieber Tabellen-Recreate-Block für `members` und `clubs` analog zu 002, 018, 039). Commit: Teil von 1.1.
- [ ] 1.3 `make migrate-up && make migrate-down && make migrate-up` lokal verifizieren — Roundtrip muss sauber laufen. Commit: kein eigener Commit, manuelle Verifikation.

## 2. Backend — Stammdaten

### 2.1 Club-Handler erweitern

- [ ] 2.1.1 `internal/config/handler.go`: Struct `Club` um `GlaeubigerID`, `IBAN`, `BIC`, `Kontoinhaber` (alle `*string`, nullable) erweitern.
- [ ] 2.1.2 `GetClub` SELECT um neue Felder erweitern, JSON-Marshalling testen.
- [ ] 2.1.3 `UpdateClub` UPDATE um neue Felder erweitern, Validierung gemäß design.md §1.1:
  - `glaeubigerIDRegex = regexp.MustCompile(`^DE\d{2}[A-Z0-9]{3}\d{11}$`)`
  - IBAN via `iban.Validate()` aus neuer interner Util-Funktion (keine externe Lib, manueller Mod-97-Check)
  - BIC: 8 oder 11 Zeichen, `^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$`
- [ ] 2.1.4 Test `TestClub_SepaFelder_GetSet` in `internal/config/handler_test.go`: setzt alle 4 Felder via PUT, liest sie via GET zurück, prüft Validierungs-Fehler bei ungültiger Gläubiger-ID und ungültiger IBAN.
- [ ] 2.1.5 Commit: `feat(config): Club-API um SEPA-Stammdaten erweitert`

### 2.2 IBAN-Util

- [ ] 2.2.1 `internal/sepa/iban.go` neu anlegen mit:
  - `func NormalizeIBAN(s string) string` — Leerzeichen raus, uppercase
  - `func IsValidIBAN(iban string) bool` — Längen-Check pro Ländercode + Mod-97-Prüfsumme
- [ ] 2.2.2 Tabellen-Test mit gültigen und ungültigen IBANs (DE, AT, CH, Müll, leerer String).
- [ ] 2.2.3 Commit: `feat(sepa): IBAN-Validierung (Mod-97, länderspezifische Länge)`

## 3. Backend — Beitragssätze

- [ ] 3.1 Package `internal/beitragssaetze/` mit `handler.go`:
  - Struct `Satz { ID, Kategorie, BetragCent, ValidFrom, CreatedAt }`
  - `List(w, r)` → `GET /api/beitrags-saetze`
  - `Create(w, r)` → `POST /api/beitrags-saetze` mit Validation gemäß design.md §7.1
  - Constructor `NewHandler(db, hub)` mit Broadcast `beitragssatz-changed`
- [ ] 3.2 Routen in `cmd/teamwerk/main.go` registrieren unter `auth.RequireClubFunction("vorstand", "kassierer")` mit Admin-Override (siehe bestehender Pattern).
- [ ] 3.3 Tests in `internal/beitragssaetze/handler_test.go`:
  - `TestSaetze_HistorieErhalten` — anlegen mit zwei `valid_from`, GET liefert beide sortiert
  - `TestSaetze_NeueValidFromAnlegen` — POST mit identischer `kategorie + valid_from` ist erlaubt, kein 409
  - `TestSaetze_InvalidKategorie` — CHECK-Verletzung → 400
  - `TestSaetze_Forbidden` — Persona `spieler` → 403 (`testutil.Token` mit nur `club_functions: ["spieler"]`)
- [ ] 3.4 Commit: `feat(beitragssaetze): CRUD-Handler + Tests`

## 4. Backend — Beitragslauf-Kern (Berechnung)

### 4.1 Package-Skelett

- [ ] 4.1.1 `internal/beitragslauf/` neu anlegen mit Dateien:
  - `compute.go` — reine Funktionen, keine DB-Abhängigkeit
  - `query.go` — DB-Reads (members, clubs, sätze)
  - `xml.go` — XML-Generator
  - `handler.go` — HTTP-Handler
  - `*_test.go` jeweils

### 4.2 Pure-Compute-Funktionen

- [ ] 4.2.1 `compute.go`:
  - `func IstVolljaehrigAmSaisonstart(dob, saisonStart time.Time) bool`
  - `func BeitragsGruppe(status string) string` — gibt `"aktiv"`, `"passiv"` oder `""`
  - `func AktivKategorie(volljaehrig, inAusb, mitStammverein bool) string`
  - `func ProRataMonate(effectiveStart, saisonEnde time.Time) int`
  - `func ProRataBetragCent(jahresbeitragCent, monate int) int` (mit `math.Round`)
  - `func SequenceType(lastEinzug *time.Time) string`
- [ ] 4.2.2 `compute_test.go`:
  - `TestAktivKategorie_AlleAchtKombinationen`
  - `TestProRataMonate_*` — Tabelle (vollSaison=12, septemberJoin=9, junior=0 für joins nach Saisonende, etc.)
  - `TestProRataBetrag_KaufmaennischeRundung` — 14000×7/12=8166.67 → 8167
  - `TestIstVolljaehrig_Stichtag` — exakt am Saisonstart 18-jährig → true; einen Tag jünger → false
- [ ] 4.2.3 Commit: `feat(beitragslauf): Pure Compute-Funktionen + Tests`

### 4.3 Stammverein-Matching

- [ ] 4.3.1 `compute.go` ergänzen:
  - `var Mitgliedsvereine = []string{...}` (8 Vereine aus Gebührenordnung)
  - `func NormalizeClubName(s string) string` — Lowercase, Whitespace-collapse, Punkte/Bindestriche raus, „/" raus
  - `func levenshtein(a, b string) int` — Standard-DP, max 50 Zeichen
  - `func MatchHomeClub(homeClub string) ClubMatch` — Result-Struct `{Matched bool, Canonical string, Warning string}`
- [ ] 4.3.2 `compute_test.go`:
  - `TestMatchHomeClub_ExakterMatch` — `"TV Cannstatt 1846"` → Matched=true, kein Warning
  - `TestMatchHomeClub_LowerCase` — `"tv cannstatt 1846"` → Matched=true
  - `TestMatchHomeClub_Fuzzy` — `"TV Cannstadt 1846"` (Tippfehler) → Matched=true, Warning gesetzt
  - `TestMatchHomeClub_Leer` — `""` → Matched=false, Warning=""
  - `TestMatchHomeClub_Unbekannt` — `"FC Bayern"` → Matched=false, Warning gesetzt
- [ ] 4.3.3 Commit: `feat(beitragslauf): Stammverein-Fuzzy-Match`

### 4.4 Sätze-Lookup

- [ ] 4.4.1 `query.go`:
  - `func LoadSaetzeMap(db *sql.DB) (map[string][]Satz, error)` — sortiert pro Kategorie nach `valid_from` DESC
  - `func LookupBetragCent(saetze map[string][]Satz, kategorie string, effectiveStart time.Time) (int, error)` — erstes Element mit `validFrom <= effectiveStart`
- [ ] 4.4.2 Test `TestLookupBetragCent_VorValidFrom` — wenn `effectiveStart < kleinster validFrom`, dann Error (kein Beitragssatz hinterlegt)
- [ ] 4.4.3 Commit: `feat(beitragslauf): Beitragssatz-Lookup nach Effective Start`

## 5. Backend — Vorschau-Endpoint

- [ ] 5.1 `query.go`: `func LoadMembersForLauf(db *sql.DB) ([]MemberRow, error)` — alle Member (kein Status-Filter, der passiert im Compute), mit JOIN auf `clubs` (einzeilig — wir hängen die SEPA-Stammdaten an).
- [ ] 5.2 `handler.go`: `func (h *Handler) Preview(w, r)` → `GET /api/beitragslauf/preview?saison_id=…`
  - Saison aus DB laden (`saisons.start_date`, `end_date`, `name` für `saison_label`)
  - Pro Member: alle Filter-Bedingungen prüfen, exclusions-Array befüllen
  - Falls included: Kategorie bestimmen, Effective Start, Monate, Betrag, SeqTp, Warnings
  - Response gemäß design.md §3.2
- [ ] 5.3 Route in `cmd/teamwerk/main.go` unter `auth.RequireClubFunction("vorstand", "kassierer")` mit Admin-Override.
- [ ] 5.4 Tests in `internal/beitragslauf/handler_test.go`:
  - `TestPreview_AktivVollMitStammverein` — Member mit `home_club="TV Cannstatt 1846"`, volljährig, aktiv → Kategorie `aktiv_volljaehrig_mit`, Betrag 14000
  - `TestPreview_PassivProRata2026` — Member mit Status `passiv`, joinDate Mitte Saison → Pro-rata korrekt
  - `TestPreview_AusschlussOhneMandat` — `sepa_mandat=0` → included=false, exclusions=[`kein_sepa_mandat`]
  - `TestPreview_AusschlussOhneIBAN` — IBAN=NULL → exclusions=[`iban_fehlt`]
  - `TestPreview_AusschlussUngueltigeIBAN` — IBAN-Mod-97 falsch → exclusions=[`iban_ungueltig`]
  - `TestPreview_WarnungUnklarerStammverein` — `home_club="FC Bayern"` → included=true (ohne Stammverein), warnings=[`home_club_unklar`]
  - `TestPreview_BeitragsfreiAusgeschlossen` — `beitragsfrei=1` → exclusions=[`beitragsfrei`]
  - `TestPreview_ProRataNeumitgliedSeptember` — `join_date=2026-09-15`, saisonStart=2026-07-01 → monate=9 (Okt-Jun), Betrag = jahres × 9/12
  - `TestPreview_Forbidden` — Persona `spieler` → 403
- [ ] 5.5 Commit: `feat(beitragslauf): Vorschau-Endpoint mit Filter, Pro-rata, Warnungen`

## 6. Backend — XML-Generator

- [ ] 6.1 `xml.go`: Structs `Document`, `CstmrDrctDbtInitn`, `GrpHdr`, `PmtInf`, `DrctDbtTxInf`, `PstlAdr` mit `encoding/xml`-Tags entsprechend pain.008.001.08.
- [ ] 6.2 `func BuildXML(input BuildInput) ([]byte, error)`:
  - Input: Saison-Label, Vereinsdaten, Fälligkeitsdatum, Liste `[]ExportItem{MemberID, Name, Adresse, IBAN, BetragCent, SeqTp, MandatRef, MandatDatum, MemberNumber}`
  - Output: serialisiertes XML inklusive `<?xml version="1.0" encoding="UTF-8"?>`-Header
  - Items nach SeqTp gruppieren → zwei `PmtInf`-Blöcke (FRST und/oder RCUR, je nach Inhalt)
  - `MsgId` und `PmtInfId` gemäß design.md §4.3, ≤ 35 Zeichen, ASCII (Umlaute strippen über `golang.org/x/text/unicode/norm` + Filter — keine externe Dependency)
- [ ] 6.3 `func parseStreet(street string) (strtNm, bldgNb string)` mit Regex aus design.md §4.5.
- [ ] 6.4 `func nextBusinessDay(t time.Time) time.Time` — wenn Sa/So, auf nächsten Werktag verschieben. (DE-Feiertage out of scope; Bank akzeptiert auch in der Praxis Feiertage als ReqdColltnDt-Eingabe und führt am nächsten Werktag aus.)
- [ ] 6.5 Tests `xml_test.go`:
  - `TestBuildXML_SnapshotFRSTUndRCUR` — golden-XML-File `testdata/beitragslauf_sample.xml`, byte-equal-Compare nach `xml.Encoder` mit `Indent("", "  ")`
  - `TestBuildXML_NurFRST` — alle Items FRST → genau ein `PmtInf`-Block
  - `TestBuildXML_StraßenParsing` — „Hauptstr. 12" → StrtNm="Hauptstr.", BldgNb="12"; „Postfach 100" → StrtNm="Postfach 100", BldgNb=""
  - `TestBuildXML_UmlautInName` — „Müller" → in `<Nm>` als `Müller` (UTF-8, kein Strip), in `MsgId` als `Mueller` (ASCII)
  - `TestBuildXML_VerwendungszweckFormat` — exakt `Jahresbeitrag Saison 2026/27 – Mitgliedsnr. 1042`
- [ ] 6.6 Commit: `feat(beitragslauf): XML-Generator pain.008.001.08`

### 6.7 XSD-Validierung

- [ ] 6.7.1 XSD `pain.008.001.08.xsd` von www.iso20022.org oder Deutsche Kreditwirtschaft herunterladen, in `internal/beitragslauf/testdata/pain.008.001.08.xsd` ablegen.
- [ ] 6.7.2 Test `TestExport_XSDValid` mit `xmllint --schema` als externem Aufruf (Skip wenn `xmllint` nicht im PATH, mit `t.Skip("xmllint not installed")`).
- [ ] 6.7.3 README-Hinweis in `internal/beitragslauf/testdata/README.md`: `brew install libxml2` lokal nötig für XSD-Validierung.
- [ ] 6.7.4 Commit: `test(beitragslauf): XSD-Schema-Validierung via xmllint`

## 7. Backend — Export- & Confirm-Endpoints

- [ ] 7.1 `handler.go`:
  - `func (h *Handler) Export(w, r)` → `POST /api/beitragslauf/export`
    - Body: `{saison_id, member_ids}`
    - Verifiziert Vereins-SEPA-Stammdaten gesetzt (sonst 400)
    - Lädt Preview-Ergebnisse, filtert auf `member_ids`
    - Wirft 400 wenn einer der `member_ids` excluded ist
    - Ruft `BuildXML` auf, Response als `application/xml` mit Content-Disposition
  - `func (h *Handler) ConfirmUploaded(w, r)` → `POST /api/beitragslauf/confirm-uploaded`
    - Body: `{member_ids}`
    - `UPDATE members SET last_sepa_einzug_am=CURRENT_TIMESTAMP WHERE id IN (…)`
    - `h.hub.Broadcast("members-changed")`
    - Response: `{updated_count}`
- [ ] 7.2 Tests:
  - `TestExport_HappyPath` — vollständiges XML, validiert gegen XSD
  - `TestExport_FRSTvsRCUR` — Mischung von Members mit/ohne `last_sepa_einzug_am` → zwei `PmtInf`-Blöcke
  - `TestExport_VerwendungszweckFormat` — Ustrd-Inhalt exakt prüfen
  - `TestExport_FehlendeStammdaten400` — Club ohne Gläubiger-ID → 400
  - `TestExport_ExcludedMember400` — `member_ids` enthält Member ohne Mandat → 400
  - `TestExport_Forbidden` — Persona `spieler` → 403
  - `TestConfirm_SetztLastEinzug` — vorher NULL, nachher Timestamp gesetzt
  - `TestConfirm_NurAngegebeneMitglieder` — Member außerhalb der Liste bleiben unverändert
  - `TestConfirm_Forbidden` — 403
- [ ] 7.3 Commit: `feat(beitragslauf): Export- und Confirm-Endpoints`

## 8. Backend — Sequence-Reset & in_ausbildung

- [ ] 8.1 `internal/members/handler.go`:
  - `Get`/`Update`-SELECT/UPDATE um `in_ausbildung` erweitern
  - Neuer Handler `SepaSequenceReset(w, r)` → `PUT /api/members/{id}/sepa-sequence-reset`
    - `UPDATE members SET last_sepa_einzug_am = NULL WHERE id = ?`
    - `h.hub.Broadcast("members-changed")`
- [ ] 8.2 Route registrieren unter `auth.RequireClubFunction("vorstand", "kassierer")` mit Admin-Override.
- [ ] 8.3 Tests:
  - `TestMember_InAusbildung_GetSet` — Toggle via PUT funktioniert
  - `TestReset_SetztAufNull` — vorher Timestamp, danach NULL
  - `TestReset_Forbidden` — Persona `spieler` → 403
- [ ] 8.4 Commit: `feat(members): in_ausbildung-Feld + SEPA-Sequence-Reset`

## 9. Frontend — AdminSettingsPage VereinTab

- [ ] 9.1 `web/src/pages/AdminSettingsPage.tsx` (VereinTab): vier neue Input-Felder Gläubiger-ID, IBAN, BIC, Kontoinhaber. Layout: zweispaltig auf Desktop, einspaltig auf Mobile.
- [ ] 9.2 Validierung clientseitig:
  - Gläubiger-ID: gleicher Regex wie Backend
  - IBAN: einfache Form-Anzeige (Leerzeichen-Formattierung beim Tippen), Mod-97 Backend-Job
- [ ] 9.3 PUT `/api/club` mit allen Feldern, Erfolgs-Toast.
- [ ] 9.4 Commit: `feat(admin-settings): SEPA-Stammdaten im VereinTab`

## 10. Frontend — BeitraegeTab in AdminSettingsPage

- [ ] 10.1 Neuer Tab „Beiträge" in `AdminSettingsPage.tsx` (zwischen „Kader" und „Dienste").
- [ ] 10.2 Sichtbarkeit: nur für `vorstand`, `kassierer`, `admin` (siehe bestehende `useHasFunction`-Pattern).
- [ ] 10.3 Komponente `BeitraegeTab.tsx`:
  - GET `/api/beitrags-saetze`
  - Pro Kategorie eine Tabelle (Datum + Betrag €), sortiert nach `valid_from` DESC
  - Unter jeder Tabelle Inline-Form: `[Datum] [Betrag in €] [Hinzufügen]`-Button
  - POST `/api/beitrags-saetze`, Reload nach Erfolg
  - Live-Update via `useLiveUpdates('beitragssatz-changed')`
- [ ] 10.4 Kategorie-Labels in `web/src/lib/beitragsKategorien.ts` (Mapping `aktiv_volljaehrig_mit` → „Aktiv volljährig (mit Stammverein)" etc.).
- [ ] 10.5 Commit: `feat(admin-settings): BeitraegeTab — Beitragsmatrix-Pflege mit Historie`

## 11. Frontend — BeitragslaufPage

- [ ] 11.1 `web/src/pages/admin/BeitragslaufPage.tsx` neu anlegen.
- [ ] 11.2 Saison-Dropdown (default = aktive Saison) → triggert `GET /api/beitragslauf/preview?saison_id=…`.
- [ ] 11.3 Summary-Header: angehakt/Warnungen/Ausgeschlossen + Gesamtsumme. Bei Wechsel der Auswahl neu berechnen client-side (`useMemo` über `items`).
- [ ] 11.4 Tabelle (Desktop) + MobileCard (`< 640px`):
  - Spalten: Checkbox, Name, Status, Kategorie, Monate, Betrag, SeqTp, Hinweise (Icon + Tooltip)
  - Default-Haken aus `item.included`; ausgeschlossene Zeilen Checkbox disabled, grauer Hintergrund
- [ ] 11.5 Buttons:
  - „XML herunterladen" → `POST /api/beitragslauf/export` mit `selected_ids`, Blob-Download via `URL.createObjectURL`
  - Nach Download: zweiter Button erscheint „Bei Bank hochgeladen bestätigen" → `POST /api/beitragslauf/confirm-uploaded` mit gleichen `selected_ids`, Success-Toast, Reload
- [ ] 11.6 Route in `web/src/App.tsx`: `<Route path="/admin/beitragslauf" element={<BeitragslaufPage />} />`
- [ ] 11.7 Nav-Eintrag in `AppShell.tsx` unter Verwaltung-Modul, Sichtbarkeit: `vorstand`, `kassierer`, `admin`.
- [ ] 11.8 Commit: `feat(admin): /admin/beitragslauf — SEPA-Vorschau, Export, Confirm`

## 12. Frontend — MemberDetailPage Anpassungen

- [ ] 12.1 `web/src/pages/MemberDetailPage.tsx`: Toggle `in_ausbildung` im Stammdaten-Abschnitt (Checkbox, sichtbar für `vorstand`, `admin`).
- [ ] 12.2 Im Datenschutz-/SEPA-Abschnitt: Button „SEPA-Sequenz zurücksetzen" sichtbar für `vorstand`, `kassierer`, `admin`, nur wenn `last_sepa_einzug_am != null`.
  - Confirm-Dialog: „Damit gilt der nächste Einzug wieder als Erstlastschrift (FRST). Fortfahren?"
  - PUT `/api/members/{id}/sepa-sequence-reset` → Success-Toast, Reload.
- [ ] 12.3 Tests in `web/src/pages/__tests__/MemberDetailPage.permissions.test.tsx` analog zu bestehender Permission-Test-Konvention (sofern Tests aus `permissions-baseline-tests` schon laufen).
- [ ] 12.4 Commit: `feat(members): in_ausbildung-Toggle und SEPA-Sequenz-Reset auf Detail-Seite`

## 13. Frontend — Live-Updates & API-Lib

- [ ] 13.1 `web/src/lib/api.ts`: keine Änderungen nötig (alle Endpoints unter `/api/`).
- [ ] 13.2 `web/src/lib/sepa.ts` neu: Hilfsfunktionen `formatBetrag(cent)`, `formatIBAN(iban)` (Vierergruppen mit Leerzeichen).
- [ ] 13.3 `useLiveUpdates` auf `BeitragslaufPage` und `MemberDetailPage`:
  - `members-changed` → Reload Vorschau bzw. Member-Detail
  - `beitragssatz-changed` → Reload BeitraegeTab
- [ ] 13.4 Commit: `feat(web): SEPA-Helper und Live-Update-Bindings`

## 14. CLAUDE.md & OpenSpec

- [ ] 14.1 `CLAUDE.md` Abschnitt „API-Routen" um die neuen Endpoints erweitern (Vorstand/Kassierer-Block).
- [ ] 14.2 `CLAUDE.md` „Datenbankschema" — neue Tabelle `beitrags_saetze`, neue Spalten in `clubs` und `members`.
- [ ] 14.3 `CLAUDE.md` „Bekannte Gotchas" — Hinweis zur Pflicht-Konfiguration der SEPA-Stammdaten vor erstem Lauf.
- [ ] 14.4 Commit: `docs(claude-md): SEPA-Beitragslauf dokumentiert`

## 15. Manuelle Verifikation

- [ ] 15.1 Lokales Setup: clubs.iban + glaeubiger_id + bic + kontoinhaber via UI gepflegt.
- [ ] 15.2 Drei Test-Member angelegt:
  - Volljährig, mit Stammverein, Erstlauf → FRST, voller Beitrag 140€
  - Minderjährig, ohne Stammverein, joinDate Mitte Saison → Pro-rata
  - Passiv, Mandat fehlt → Ausgeschlossen
- [ ] 15.3 Vorschau aufrufen, Auswahl bestätigen, XML herunterladen.
- [ ] 15.4 XML mit `xmllint --schema testdata/pain.008.001.08.xsd downloaded.xml` lokal validieren.
- [ ] 15.5 Bei BW-Bank im Test-Modus / Test-Mandant einreichen, Erfolgsmeldung dokumentieren.
- [ ] 15.6 „Bei Bank hochgeladen bestätigen" klicken, Detail-Seite eines Members öffnen, `last_sepa_einzug_am` sichtbar, „SEPA-Sequenz zurücksetzen"-Button erscheint, klicken → wieder NULL.

## 16. Abschluss

- [ ] 16.1 Alle Tests grün: `make test` (Backend + Frontend).
- [ ] 16.2 `make coverage` — `internal/beitragslauf` ≥ 80% (Berechnungs- und XML-Logik komplett abgedeckt).
- [ ] 16.3 PR-Beschreibung mit Screenshots BeitragslaufPage (Desktop + Mobile), beispielhaftem XML-Snippet (anonymisiert).
- [ ] 16.4 Nach Merge: Proposal archivieren via `/openspec-archive-change`.
- [ ] 16.5 Follow-up-Issue: „Audit-Log für SEPA-Sequenz-Resets" (out of scope dieses Proposals).

## Abhängigkeiten

- 1.1 (Migration) blockiert 2.x, 3.x, 4.x, 5.x, 7.x, 8.x
- 2.2 (IBAN-Util) blockiert 5.x (Vorschau-Filter), 7.x (Export-Validierung)
- 4.x (Compute) blockiert 5.x, 7.x
- 5.x (Preview) blockiert 7.1 (Export greift auf Preview-Logik zu) und 11.x (Frontend)
- 6.x (XML-Generator) blockiert 7.1
- 9.x + 10.x (Settings) können parallel zu Backend laufen, sind aber für 15.x (manuelle Verifikation) Pflicht
- 11.x (BeitragslaufPage) blockiert von 5.x und 7.x

## Aufwand-Schätzung

| Phase | Tasks | Aufwand |
|---|---|---|
| 1: Migration | 1.1–1.3 | 0,5 Tag |
| 2: Stammdaten Backend | 2.1–2.2 | 1 Tag |
| 3: Beitragssätze Backend | 3.1–3.4 | 0,5 Tag |
| 4: Compute | 4.1–4.4 | 1,5 Tage |
| 5: Preview-Endpoint | 5.1–5.5 | 1,5 Tage |
| 6: XML-Generator | 6.1–6.7 | 2 Tage |
| 7: Export & Confirm | 7.1–7.3 | 1 Tag |
| 8: Sequence-Reset | 8.1–8.4 | 0,5 Tag |
| 9: VereinTab UI | 9.1–9.4 | 0,5 Tag |
| 10: BeitraegeTab UI | 10.1–10.5 | 1 Tag |
| 11: BeitragslaufPage | 11.1–11.8 | 2 Tage |
| 12: MemberDetailPage | 12.1–12.4 | 0,5 Tag |
| 13: Frontend-Glue | 13.1–13.4 | 0,5 Tag |
| 14: Docs | 14.1–14.4 | 0,5 Tag |
| 15: Manuelle Verifikation | 15.1–15.6 | 1 Tag |
| 16: Abschluss | 16.1–16.5 | 0,5 Tag |
| **Summe** | | **≈ 14,5 Tage** |
