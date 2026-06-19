# Tasks: SEPA-Beitragslauf

## 1. Datenbank-Migration

- [x] 1.1 `internal/db/migrations/043_sepa_beitragslauf.up.sql` anlegen:
  - `ALTER TABLE clubs ADD COLUMN glaeubiger_id TEXT;`
  - `ALTER TABLE clubs ADD COLUMN iban TEXT;`
  - `ALTER TABLE clubs ADD COLUMN bic TEXT;`
  - `ALTER TABLE clubs ADD COLUMN kontoinhaber TEXT;`
  - `CREATE TABLE beitrags_saetze (…)` gemäß design.md §1.3 inkl. CHECK-Constraint (`aktiv_ohne`, `aktiv_mit`, `passiv`), Index
  - Seed-INSERTs für die 3 Kategorien mit `INSERT OR IGNORE` und `valid_from` aus Gebührenordnung
  - (Keine `members`-Spalten — kein FRST/RCUR-Tracking.)
  - Commit: `chore(db): Migration 043 — SEPA-Stammdaten und Beitragsmatrix`
- [x] 1.2 Korrespondierende `.down.sql` mit `DROP TABLE beitrags_saetze;` und Spalten-Drop für `clubs` via Tabellen-Recreate (SQLite hat kein `DROP COLUMN` vor 3.35; Tabellen-Recreate-Block analog zu 002, 018, 039). Commit: Teil von 1.1.
- [x] 1.3 Up-Migration auf Temp-DB verifiziert (clubs-Spalten + 3 Seed-Sätze). Hinweis: das `migrate`-CLI führt nur `up` aus (`down` ist nicht verdrahtet); die `.down.sql` nutzt `DROP COLUMN` (wie in Migration 035 bewährt).

## 2. Backend — Stammdaten

### 2.1 Club-Handler erweitern

- [x] 2.1.1 `internal/config/handler.go`: Struct `Club` um `GlaeubigerID`, `IBAN`, `BIC`, `Kontoinhaber` (alle `*string`, nullable) erweitern.
- [x] 2.1.2 `GetClub` SELECT um neue Felder erweitern, JSON-Marshalling testen.
- [x] 2.1.3 `UpdateClub` UPDATE um neue Felder erweitern, Validierung gemäß design.md §1.1:
  - `glaeubigerIDRegex = regexp.MustCompile(`^DE\d{2}[A-Z0-9]{3}\d{11}$`)`
  - IBAN via `iban.Validate()` aus neuer interner Util-Funktion (keine externe Lib, manueller Mod-97-Check)
  - BIC: 8 oder 11 Zeichen, `^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$`
- [x] 2.1.4 Test `TestClub_SepaFelder_GetSet` in `internal/config/handler_test.go`: setzt alle 4 Felder via PUT, liest sie via GET zurück, prüft Validierungs-Fehler bei ungültiger Gläubiger-ID und ungültiger IBAN.
- [x] 2.1.5 Commit: `feat(config): Club-API um SEPA-Stammdaten erweitert`

### 2.2 IBAN-Util

- [x] 2.2.1 `internal/sepa/iban.go` neu anlegen mit:
  - `func NormalizeIBAN(s string) string` — Leerzeichen raus, uppercase
  - `func IsValidIBAN(iban string) bool` — Längen-Check pro Ländercode + Mod-97-Prüfsumme
- [x] 2.2.2 Tabellen-Test mit gültigen und ungültigen IBANs (DE, AT, CH, Müll, leerer String).
- [x] 2.2.3 Commit: `feat(sepa): IBAN-Validierung (Mod-97, länderspezifische Länge)`

## 3. Backend — Beitragssätze

- [x] 3.1 Package `internal/beitragssaetze/` mit `handler.go`:
  - Struct `Satz { ID, Kategorie, BetragCent, ValidFrom, CreatedAt }`
  - `List(w, r)` → `GET /api/fee-rates`
  - `Create(w, r)` → `POST /api/fee-rates` mit Validation gemäß design.md §7.1 (Kategorie ∈ {`aktiv_ohne`, `aktiv_mit`, `passiv`})
  - Constructor `NewHandler(db, hub)` mit Broadcast `beitragssatz-changed`
- [x] 3.2 Routen in `cmd/teamwerk/main.go` / `internal/app/router.go` registrieren unter `auth.RequireClubFunction("vorstand", "kassierer")` mit Admin-Override.
- [x] 3.3 Tests in `internal/beitragssaetze/handler_test.go`:
  - `TestSaetze_HistorieErhalten` — anlegen mit zwei `valid_from`, GET liefert beide sortiert
  - `TestSaetze_NeueValidFromAnlegen` — POST mit identischer `kategorie + valid_from` ist erlaubt, kein 409
  - `TestSaetze_InvalidKategorie` — CHECK-Verletzung → 400
  - `TestSaetze_Forbidden` — Persona `spieler` → 403 (`testutil.Token` mit nur `club_functions: ["spieler"]`)
- [x] 3.4 Commit: `feat(beitragssaetze): CRUD-Handler + Tests`

## 4. Backend — Beitragslauf-Kern (Kategorisierung)

### 4.1 Package-Skelett

- [x] 4.1.1 `internal/beitragslauf/` neu anlegen mit Dateien:
  - `compute.go` — reine Funktionen, keine DB-Abhängigkeit
  - `query.go` — DB-Reads (members, clubs, sätze)
  - `xml.go` — XML-Generator
  - `protokoll.go` — append-only Protokoll-Writer
  - `handler.go` — HTTP-Handler
  - `*_test.go` jeweils

### 4.2 Pure-Compute-Funktionen

- [x] 4.2.1 `compute.go`:
  - `func BeitragsGruppe(status string) string` — gibt `"aktiv"`, `"passiv"` oder `""`
  - `func AktivKategorie(mitStammverein bool) string` — `"aktiv_mit"` oder `"aktiv_ohne"`
  - (Keine Pro-rata-/Volljährigkeits-/Ausbildungs-/SeqTp-Funktionen — bewusst entfallen.)
- [x] 4.2.2 `compute_test.go`:
  - `TestAktivKategorie_MitOhneStammverein` — 2 Fälle
  - `TestBeitragsGruppe_AlleStatus` — aktiv/verletzt → aktiv, pausiert/passiv → passiv, sonst → ""
- [x] 4.2.3 Commit: `feat(beitragslauf): Kategorisierungs-Funktionen + Tests`

### 4.3 Stammverein-Matching

- [x] 4.3.1 `compute.go` ergänzen:
  - `var Mitgliedsvereine = []string{...}` (8 Vereine aus Gebührenordnung)
  - `func NormalizeClubName(s string) string` — Lowercase, Whitespace-collapse, Punkte/Bindestriche raus, „/" raus
  - `func levenshtein(a, b string) int` — Standard-DP, max 50 Zeichen
  - `func MatchHomeClub(homeClub string) ClubMatch` — Result-Struct `{Matched bool, Canonical string, Warning string}`
- [x] 4.3.2 `compute_test.go`:
  - `TestMatchHomeClub_ExakterMatch` — `"TV Cannstatt 1846"` → Matched=true, kein Warning
  - `TestMatchHomeClub_LowerCase` — `"tv cannstatt 1846"` → Matched=true
  - `TestMatchHomeClub_Fuzzy` — `"TV Cannstadt 1846"` (Tippfehler) → Matched=true, Warning gesetzt
  - `TestMatchHomeClub_Leer` — `""` → Matched=false, Warning=""
  - `TestMatchHomeClub_Unbekannt` — `"FC Bayern"` → Matched=false, Warning gesetzt
- [x] 4.3.3 Commit: `feat(beitragslauf): Stammverein-Fuzzy-Match`

### 4.4 Sätze-Lookup

- [x] 4.4.1 `query.go`:
  - `func LoadSaetzeMap(db *sql.DB) (map[string][]Satz, error)` — sortiert pro Kategorie nach `valid_from` DESC
  - `func LookupBetragCent(saetze map[string][]Satz, kategorie string, saisonStart time.Time) (int, error)` — erstes Element mit `validFrom <= saisonStart`; Error wenn kein Satz vor Saisonstart hinterlegt
- [x] 4.4.2 Test `TestLookupBetragCent_VorValidFrom` — wenn `saisonStart < kleinster validFrom`, dann Error (kein Beitragssatz hinterlegt)
- [x] 4.4.3 Commit: `feat(beitragslauf): Beitragssatz-Lookup nach Saisonstart`

## 5. Backend — Vorschau-Endpoint

- [x] 5.1 `query.go`: `func LoadMembersForLauf(db *sql.DB) ([]MemberRow, error)` — alle Member (kein Status-Filter, der passiert im Compute), mit JOIN auf `clubs` (einzeilig — wir hängen die SEPA-Stammdaten an).
- [x] 5.2 `handler.go`: `func (h *Handler) Preview(w, r)` → `GET /api/fee-run/preview?saison_id=…`
  - Saison aus DB laden (`saisons.start_date`, `end_date`, `name` für `saison_label`); Fälligkeit = 01.07. der Saison
  - Pro Member: alle Filter-Bedingungen prüfen, exclusions-Array befüllen
  - Falls included: Beitragsgruppe → Kategorie (Stammverein), voller Jahresbeitrag via `LookupBetragCent(…, saisonStart)`, Warnings
  - Response gemäß design.md §3.2 (kein `monate`-, kein `seq_tp`-Feld)
- [x] 5.3 Route in `internal/app/router.go` unter `auth.RequireClubFunction("vorstand", "kassierer")` mit Admin-Override.
- [x] 5.4 Tests in `internal/beitragslauf/handler_test.go`:
  - `TestPreview_AktivMitStammverein` — Member mit `home_club="TV Cannstatt 1846"`, aktiv → Kategorie `aktiv_mit`, Betrag 9600
  - `TestPreview_AktivOhneStammverein` — Member ohne `home_club`, aktiv → Kategorie `aktiv_ohne`, Betrag 22600
  - `TestPreview_PassivVollerBeitrag` — Member mit Status `passiv` → Kategorie `passiv`, voller Beitrag
  - `TestPreview_AusschlussOhneMandat` — `sepa_mandat=0` → included=false, exclusions=[`kein_sepa_mandat`]
  - `TestPreview_AusschlussOhneIBAN` — IBAN=NULL → exclusions=[`iban_fehlt`]
  - `TestPreview_AusschlussUngueltigeIBAN` — IBAN-Mod-97 falsch → exclusions=[`iban_ungueltig`]
  - `TestPreview_WarnungUnklarerStammverein` — `home_club="FC Bayern"` → included=true (ohne Stammverein), warnings=[`home_club_unklar`]
  - `TestPreview_BeitragsfreiAusgeschlossen` — `beitragsfrei=1` → exclusions=[`beitragsfrei`]
  - `TestPreview_NeumitgliedZahltVollenBeitrag` — `join_date=2026-09-15`, saisonStart=2026-07-01 → voller Jahresbeitrag (kein Pro-rata-Abzug)
  - `TestPreview_KassiererErlaubt` — Persona `kassierer` → 200
  - `TestPreview_Forbidden` — Persona `spieler` → 403
- [x] 5.5 Commit: `feat(beitragslauf): Vorschau-Endpoint mit Filter, Kategorie, Warnungen`

## 6. Backend — XML-Generator

- [x] 6.1 `xml.go`: Structs `Document`, `CstmrDrctDbtInitn`, `GrpHdr`, `PmtInf`, `DrctDbtTxInf`, `PstlAdr` mit `encoding/xml`-Tags entsprechend pain.008.001.08.
- [x] 6.2 `func BuildXML(input BuildInput) ([]byte, error)`:
  - Input: Saison-Label, Vereinsdaten, Fälligkeitsdatum (01.07.), Liste `[]ExportItem{MemberID, Name, Adresse, IBAN, BetragCent, MandatRef, MandatDatum, MemberNumber}`
  - Output: serialisiertes XML inklusive `<?xml version="1.0" encoding="UTF-8"?>`-Header
  - Genau ein `PmtInf`-Block mit `SeqTp = RCUR`
  - `MsgId` und `PmtInfId` gemäß design.md §4.3, ≤ 35 Zeichen, ASCII (Umlaute strippen über `golang.org/x/text/unicode/norm` + Filter — keine externe Dependency)
- [x] 6.3 `func parseStreet(street string) (strtNm, bldgNb string)` mit Regex aus design.md §4.5.
- [x] 6.4 `func nextBusinessDay(t time.Time) time.Time` — wenn Sa/So, auf nächsten Werktag verschieben. Angewandt auf die 01.07.-Fälligkeit. (DE-Feiertage out of scope.)
- [x] 6.5 Tests `xml_test.go`:
  - `TestBuildXML_SnapshotRCUR` — golden-XML-File `testdata/beitragslauf_sample.xml`, byte-equal-Compare nach `xml.Encoder` mit `Indent("", "  ")`
  - `TestBuildXML_EinPmtInfBlockRCUR` — Ausgabe enthält genau einen `PmtInf`-Block mit `SeqTp=RCUR`
  - `TestBuildXML_StraßenParsing` — „Hauptstr. 12" → StrtNm="Hauptstr.", BldgNb="12"; „Postfach 100" → StrtNm="Postfach 100", BldgNb=""
  - `TestBuildXML_UmlautInName` — „Müller" → in `<Nm>` als `Müller` (UTF-8, kein Strip), in `MsgId` als `Mueller` (ASCII)
  - `TestBuildXML_VerwendungszweckFormat` — exakt `Jahresbeitrag Saison 2026/27 – Mitgliedsnr. 1042`
- [x] 6.6 Commit: `feat(beitragslauf): XML-Generator pain.008.001.08`

### 6.7 XSD-Validierung

- [ ] 6.7.1 XSD `pain.008.001.08.xsd` von www.iso20022.org oder Deutsche Kreditwirtschaft herunterladen, in `internal/beitragslauf/testdata/pain.008.001.08.xsd` ablegen.
- [ ] 6.7.2 Test `TestExport_XSDValid` mit `xmllint --schema` als externem Aufruf (Skip wenn `xmllint` nicht im PATH, mit `t.Skip("xmllint not installed")`).
- [ ] 6.7.3 README-Hinweis in `internal/beitragslauf/testdata/README.md`: `brew install libxml2` lokal nötig für XSD-Validierung.
- [ ] 6.7.4 Commit: `test(beitragslauf): XSD-Schema-Validierung via xmllint`

## 7. Backend — Export-Endpoint

- [x] 7.1 `handler.go`:
  - `func (h *Handler) Export(w, r)` → `POST /api/fee-run/export`
    - Body: `{saison_id, member_ids}`
    - Verifiziert Vereins-SEPA-Stammdaten gesetzt (sonst 400)
    - Lädt Preview-Ergebnisse, filtert auf `member_ids`
    - Wirft 400 wenn einer der `member_ids` excluded ist
    - Ruft `BuildXML` auf, Response als `application/xml` mit Content-Disposition
    - Keine DB-Mutation, kein Protokoll-Schreiben
- [x] 7.2 Route unter `auth.RequireClubFunction("vorstand", "kassierer")` mit Admin-Override.
- [x] 7.3 Tests:
  - `TestExport_HappyPath` — vollständiges XML, validiert gegen XSD
  - `TestExport_EinPmtInfBlockRCUR` — Ausgabe enthält genau einen `PmtInf`-Block mit `SeqTp=RCUR`
  - `TestExport_VerwendungszweckFormat` — Ustrd-Inhalt exakt prüfen
  - `TestExport_FehlendeStammdaten400` — Club ohne Gläubiger-ID → 400
  - `TestExport_ExcludedMember400` — `member_ids` enthält Member ohne Mandat → 400
  - `TestExport_KassiererErlaubt` — Persona `kassierer` → 200
  - `TestExport_Forbidden` — Persona `spieler` → 403
- [x] 7.4 Commit: `feat(beitragslauf): Export-Endpoint`

## 8. Backend — Confirm & Saison-Protokoll

- [x] 8.1 `internal/config/config.go`: neues Feld `BeitragslaufDir` mit `getEnv("BEITRAGSLAUF_DIR", "./storage/beitragslauf-protokolle")`; beim App-Start via `os.MkdirAll(cfg.BeitragslaufDir, 0755)` anlegen.
- [x] 8.2 `protokoll.go`:
  - `func protokollPfad(dir, saisonKurz string) string` — `beitragslauf_{saison_kurz mit /→-}.txt`
  - `func AppendProtokoll(dir, saisonKurz string, entry ProtokollEntry) error` — `os.OpenFile(..., O_APPEND|O_CREATE|O_WRONLY, 0644)`, formatierter Block gemäß design.md §5.2
  - `func ReadProtokoll(dir, saisonKurz string) ([]byte, error)` — Datei-Inhalt; `os.IsNotExist` → leerer Inhalt
- [x] 8.3 `handler.go`:
  - `func (h *Handler) Confirm(w, r)` → `POST /api/fee-run/confirm`
    - Body: `{saison_id, results: [{member_id, betrag_cent, success}]}`
    - Saison-Label laden; Name + Mitgliedsnummer pro `member_id` aus DB nachladen
    - Zeitstempel aus `time.Now()`, Nutzer aus `auth.ClaimsFromCtx`
    - `AppendProtokoll(...)`, Response `{saison_label, erfolgreich, nicht_erfolgreich, summe_erfolgreich_cent}`
    - Keine Member-Mutation
  - `func (h *Handler) Protocol(w, r)` → `GET /api/fee-run/protocol?saison_id=…`
    - `ReadProtokoll(...)`, `Content-Type: text/plain; charset=utf-8`
- [x] 8.4 Routen unter `auth.RequireClubFunction("vorstand", "kassierer")` mit Admin-Override.
- [x] 8.5 Tests (Protokoll-Dir = `t.TempDir()`):
  - `TestConfirm_HaengtProtokollAn` — zwei Confirm-Aufrufe → Datei enthält zwei Blöcke, erster Block unverändert
  - `TestConfirm_ErfolgUndFehlerGetrennt` — `success`/`failed` korrekt gelistet, Summe nur über erfolgreiche
  - `TestConfirm_Forbidden` — Persona `spieler` → 403
  - `TestProtocol_LiefertInhalt` — nach Confirm liefert GET den Text
  - `TestProtocol_LeerWennKeinLauf` — ohne Datei sauberer leerer/404-Response
  - `TestProtocol_Forbidden` — Persona `spieler` → 403
- [x] 8.6 Commit: `feat(beitragslauf): Confirm-Endpoint + append-only Saison-Protokoll`

## 9. Backend — Kassierer-Zugriff auf Mitglieder

- [x] 9.1 `internal/app/router.go`: Member-**Lese**-Routen aus der Vorstand-only-Gruppe in eine neue Gruppe `auth.RequireClubFunction("vorstand", "kassierer")` verschieben: `GET /api/members`, `GET /api/members/{id}`, `GET /api/members/{id}/parents`, `GET /api/members/export`. SEPA-Mandat-Datei-Routen (`POST /api/upload/sepa-mandat/{id}`, `DELETE /api/members/{id}/sepa-mandat`) ebenfalls für `kassierer` freigeben.
- [x] 9.2 `internal/members/handler.go`: neuer Handler `UpdateBankdaten(w, r)` → `PUT /api/members/{id}/bank-details` mit Feld-Whitelist (`iban`, `sepa_mandat`, `sepa_mandat_date`, `account_holder`, `street`, `zip`, `city`); IBAN-Validierung (Mod-97); `h.hub.Broadcast("members-changed")`. Route in der `vorstand`+`kassierer`-Gruppe.
- [x] 9.3 Tests in `internal/members/handler_test.go`:
  - `TestMembers_KassiererDarfLesen` — Persona `kassierer` → `GET /api/members` 200
  - `TestMembers_SpielerVerboten` — Persona `spieler` → 403
  - `TestBankdaten_KassiererUpdatetNurBankfelder` — setzt IBAN/Adresse; Name, Status, `beitragsfrei` bleiben unverändert
  - `TestBankdaten_UngueltigeIBAN400`
  - `TestBankdaten_Forbidden` — Persona `spieler` → 403
- [x] 9.4 Commit: `feat(members): Kassierer-Lesezugriff + Bankdaten-Endpoint`

## 10. Frontend — AdminSettingsPage VereinTab

- [x] 10.1 `web/src/pages/AdminSettingsPage.tsx` (VereinTab): vier neue Input-Felder Gläubiger-ID, IBAN, BIC, Kontoinhaber. Layout: zweispaltig auf Desktop, einspaltig auf Mobile.
- [x] 10.2 Validierung clientseitig: Gläubiger-ID-Regex wie Backend; IBAN-Formattierung beim Tippen, Mod-97 serverseitig.
- [x] 10.3 PUT `/api/club` mit allen Feldern, Erfolgs-Toast.
- [x] 10.4 Commit: `feat(admin-settings): SEPA-Stammdaten im VereinTab`

## 11. Frontend — BeitraegeTab in AdminSettingsPage

- [x] 11.1 Neuer Tab „Beiträge" in `AdminSettingsPage.tsx` (zwischen „Kader" und „Dienste").
- [x] 11.2 Sichtbarkeit: nur für `vorstand`, `kassierer`, `admin` (siehe bestehende `useHasFunction`-Pattern).
- [x] 11.3 Komponente `BeitraegeTab.tsx`:
  - GET `/api/fee-rates`
  - Pro Kategorie (3 Stück) eine Tabelle (Datum + Betrag €), sortiert nach `valid_from` DESC
  - Inline-Form: `[Datum] [Betrag in €] [Hinzufügen]`-Button → POST `/api/fee-rates`, Reload
  - Live-Update via `useLiveUpdates('beitragssatz-changed')`
- [x] 11.4 Kategorie-Labels in `web/src/lib/beitragsKategorien.ts` (`aktiv_mit` → „Aktiv (mit Stammverein)", `aktiv_ohne` → „Aktiv (ohne Stammverein)", `passiv` → „Passiv").
- [x] 11.5 Commit: `feat(admin-settings): BeitraegeTab — Beitragsmatrix-Pflege mit Historie`

## 12. Frontend — BeitragslaufPage

- [x] 12.1 `web/src/pages/admin/BeitragslaufPage.tsx` neu anlegen.
- [x] 12.2 Saison-Dropdown (default = aktive Saison) → triggert `GET /api/fee-run/preview?saison_id=…`.
- [x] 12.3 Summary-Header: angehakt/Warnungen/Ausgeschlossen + Gesamtsumme. Client-seitig via `useMemo` über `items`.
- [x] 12.4 Tabelle (Desktop) + MobileCard (`< 640px`):
  - Spalten: Checkbox, Name, Status, Kategorie, Betrag, Hinweise (Icon + Tooltip) — keine Monate-/SeqTp-Spalte
  - Default-Haken aus `item.included`; ausgeschlossene Zeilen Checkbox disabled, grauer Hintergrund
- [x] 12.5 Button „XML herunterladen" → `POST /api/fee-run/export` mit `selected_ids`, Blob-Download via `URL.createObjectURL`.
- [x] 12.6 Button „Lauf bestätigen" (erst nach Export aktiv) → Dialog mit den exportierten Mitgliedern, Default „erfolgreich", einzelne als „nicht eingezogen" abhakbar → `POST /api/fee-run/confirm` mit `{saison_id, results}`, Success-Toast.
- [x] 12.7 Button „Protokoll ansehen" → `GET /api/fee-run/protocol?saison_id=…`, Text in Modal mit Download-Möglichkeit.
- [x] 12.8 Route in `web/src/App.tsx`: `<Route path="/admin/beitragslauf" element={<BeitragslaufPage />} />`
- [x] 12.9 Nav-Eintrag in `AppShell.tsx` unter Verwaltung-Modul, Sichtbarkeit: `vorstand`, `kassierer`, `admin`.
- [x] 12.10 Commit: `feat(admin): /admin/beitragslauf — Vorschau, Export, Bestätigen, Protokoll`

## 13. Frontend — Mitglieder-Bereich für Kassierer

- [x] 13.1 `AppShell.tsx`: Mitglieder-Nav-Eintrag zusätzlich für `kassierer` sichtbar.
- [x] 13.2 `web/src/pages/MembersPage.tsx` / `MemberDetailPage.tsx`: für `kassierer` erreichbar; Nicht-Bankfelder schreibgeschützt anzeigen.
- [x] 13.3 `web/src/components/admin/MemberDatenschutzTab.tsx`: Bankdaten-Formular (IBAN, SEPA-Mandat, Kontoinhaber, Adresse) für `kassierer` editierbar → `PUT /api/members/{id}/bank-details`; SEPA-Mandat-Upload/Delete für `kassierer` freigeschaltet. (Backend freigegeben; Mandat-PDF-Widget bleibt vorerst im Admin-Datenschutz-Tab.)
- [x] 13.4 Tests in `web/src/pages/__tests__/MemberDetailPage.permissions.test.tsx`: Kassierer sieht/bearbeitet Bankdaten, kann übrige Felder nicht ändern (sofern Permission-Tests aus `permissions-baseline-tests` laufen).
- [x] 13.5 Commit: `feat(members): Kassierer-Zugriff im Mitglieder-Bereich`

## 14. Frontend — Helper & Live-Updates

- [x] 14.1 `web/src/lib/sepa.ts` neu: `formatBetrag(cent)`, `formatIBAN(iban)` (Vierergruppen mit Leerzeichen).
- [x] 14.2 `useLiveUpdates('beitragssatz-changed')` auf BeitraegeTab; `useLiveUpdates('members-changed')` auf BeitragslaufPage/MemberDetailPage → Reload.
- [x] 14.3 Commit: `feat(web): SEPA-Helper und Live-Update-Bindings`

## 15. CLAUDE.md & OpenSpec

- [x] 15.1 `CLAUDE.md` „API-Routen" um die neuen Endpoints erweitern; Kassierer-Block (Member-Lesen + `bankdaten`, Beitragslauf) dokumentieren.
- [x] 15.2 `CLAUDE.md` „Datenbankschema" — neue Tabelle `beitrags_saetze`, neue Spalten in `clubs`.
- [x] 15.3 `CLAUDE.md` „Bekannte Gotchas" — SEPA-Stammdaten Pflicht vor erstem Lauf; voller Jahresbeitrag ohne Pro-rata, Fälligkeit 01.07., alle Einzüge RCUR; `BEITRAGSLAUF_DIR` ins Backup aufnehmen.
- [x] 15.4 `.env.example` + Deploy-Doku um `BEITRAGSLAUF_DIR` ergänzen.
- [ ] 15.5 Commit: `docs(claude-md): SEPA-Beitragslauf dokumentiert`

## 16. Manuelle Verifikation

- [ ] 16.1 Lokales Setup: clubs.iban + glaeubiger_id + bic + kontoinhaber via UI gepflegt; `BEITRAGSLAUF_DIR` gesetzt.
- [ ] 16.2 Drei Test-Member angelegt:
  - Aktiv, mit Stammverein → voller Beitrag 96€
  - Aktiv, ohne Stammverein, joinDate Mitte Saison → voller Beitrag 226€ (kein Pro-rata)
  - Passiv, Mandat fehlt → Ausgeschlossen
- [ ] 16.3 Als **Kassierer** einloggen: Mitglied öffnen, Bankdaten korrigieren; Vorschau aufrufen, Auswahl bestätigen, XML herunterladen.
- [ ] 16.4 XML mit `xmllint --schema testdata/pain.008.001.08.xsd downloaded.xml` validieren; prüfen, dass genau ein `PmtInf`-Block mit `SeqTp=RCUR` vorliegt.
- [ ] 16.5 „Lauf bestätigen" (eines Mitglied als „nicht eingezogen" markieren) → Protokoll ansehen: zwei-Gruppen-Block erscheint; zweite Bestätigung hängt weiteren Block an, erster bleibt erhalten.
- [ ] 16.6 Bei BW-Bank im Test-Modus / Test-Mandant einreichen, Erfolgsmeldung dokumentieren.

## 17. Abschluss

- [x] 17.1 Alle Tests grün: `make test` (Backend + Frontend).
- [ ] 17.2 `make coverage` — `internal/beitragslauf` ≥ 80% (Kategorisierungs-, XML- und Protokoll-Logik abgedeckt).
- [ ] 17.3 PR-Beschreibung mit Screenshots BeitragslaufPage (Desktop + Mobile), beispielhaftem XML- und Protokoll-Snippet (anonymisiert).
- [ ] 17.4 Nach Merge: Proposal archivieren via `/openspec-archive-change`.

## Abhängigkeiten

- 1.1 (Migration) blockiert 2.x, 3.x, 4.x, 5.x, 7.x, 9.x
- 2.2 (IBAN-Util) blockiert 5.x, 7.x, 9.2 (Bankdaten-Validierung)
- 4.x (Compute) blockiert 5.x, 7.x
- 5.x (Preview) blockiert 7.1 und 12.x
- 6.x (XML-Generator) blockiert 7.1
- 8.1 (Config `BeitragslaufDir`) blockiert 8.2–8.5
- 9.x (Kassierer-Zugriff) blockiert 13.x
- 10.x + 11.x (Settings) parallel zu Backend, für 16.x Pflicht
- 12.x (BeitragslaufPage) blockiert von 5.x, 7.x, 8.x

## Aufwand-Schätzung

| Phase | Tasks | Aufwand |
|---|---|---|
| 1: Migration | 1.1–1.3 | 0,5 Tag |
| 2: Stammdaten Backend | 2.1–2.2 | 1 Tag |
| 3: Beitragssätze Backend | 3.1–3.4 | 0,5 Tag |
| 4: Kategorisierung | 4.1–4.4 | 1 Tag |
| 5: Preview-Endpoint | 5.1–5.5 | 1 Tag |
| 6: XML-Generator | 6.1–6.7 | 2 Tage |
| 7: Export | 7.1–7.4 | 0,5 Tag |
| 8: Confirm & Protokoll | 8.1–8.6 | 1 Tag |
| 9: Kassierer-Member-Zugriff | 9.1–9.4 | 1 Tag |
| 10: VereinTab UI | 10.1–10.4 | 0,5 Tag |
| 11: BeitraegeTab UI | 11.1–11.5 | 0,5 Tag |
| 12: BeitragslaufPage | 12.1–12.10 | 2 Tage |
| 13: Mitglieder-Bereich Kassierer | 13.1–13.5 | 1 Tag |
| 14: Frontend-Glue | 14.1–14.3 | 0,5 Tag |
| 15: Docs | 15.1–15.5 | 0,5 Tag |
| 16: Manuelle Verifikation | 16.1–16.6 | 0,5 Tag |
| 17: Abschluss | 17.1–17.4 | 0,5 Tag |
| **Summe** | | **≈ 15 Tage** |
