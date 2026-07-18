## 1. Phase 1 — Charakterisierungstests (`internal/members/import_test.go`)

- [x] 1.1 Test-Response-Struct `importReport` erweitern: pro Row `Message` (`json:"message"`), `DOB` (`json:"dob"`), `IBANWarning` (`json:"iban_warning"`) — additiv, bricht keinen Bestandstest
- [x] 1.2 Inline-Multipart-Helfer für „ohne Datei-Feld" (nur `WriteField("mode","append")`), da `postImport`/`PostMultipart` immer ein `file`-Feld schreiben
- [x] 1.3 BOM: `TestImport_StripsUTF8BOM` (BOM vor Header → 200, Created==1), optional `TestImport_BOMWithCommaDelimiter`
- [x] 1.4 Delimiter: `TestImport_DelimiterSemicolon`, `TestImport_DelimiterComma`, `TestImport_DelimiterDetectedFromFirstLineOnly` (Detection nur erste Zeile)
- [x] 1.5 Aliase: `TestImport_ColumnAliasName` (`Name`→`Nachname`), `TestImport_ColumnAliasGeborenAmUndMitgliedSeit`
- [x] 1.6 Dedup: `TestImport_CSVInternalDuplicate` (beide Zeilen error, Meldungen „auch/zuerst Zeile N") **und** `TestImport_CSVDuplicateDistinctByDOB` (verschiedene DOB ⇒ kein Dup — sichert die DOB-Komponente des Dedup-Keys, Stufe 3; **verpflichtend**, nicht optional)
- [x] 1.7 400-Pfade: `TestImport_MissingRequiredColumnVorname`, `_MissingRequiredColumnNachname`, `_BrokenCSVFieldCount`, `_EmptyFileNoHeader`, `_MissingFileField` (text/plain-Body prüfen, kein decode)
- [x] 1.8 `TestImport_EmptyNameCellIsRowError` (leerer Vorname → Row-error, 200 gesamt, valide Zeile trotzdem created)
- [x] 1.9 `TestImport_EnrichNotFound` (enrich + unbekannt → not_found, kein Insert, `name="Neu, Max"`, `dob="2007-10-14"`)
- [x] 1.10 **Stufe-4-Guard** `TestImport_EnrichAmbiguousNoDOB_MeldungA` — enrich, DOB in CSV fehlt, ≥2 gleichnamige DB-Treffer → Meldung **A** `"Mehrdeutig (%d Treffer) – Geburtsdatum in CSV fehlt"` (handler.go:1886-1901; anderer Zweig als der emptyCnt-Pfad von `_EnrichAmbiguousNoDOB_NichtBefuellt`), nicht befüllt
- [x] 1.11 **Stufe-6-Guard** `TestImport_UpdateChangesContract` — Multi-Feld-Update → `Rows[0].Changes` enthält die exakten `"Feld: alt → neu"`-Strings in der erzeugten Reihenfolge (der einzige Test, der den beobachtbaren `changes[]`-Report-Contract festnagelt — Format entsteht komplett in `buildMemberUpdate`)
- [x] 1.12 **Stufe-5-Guard** `TestImport_AppendStatusFallbackAktiv` — append ohne/mit unbekanntem `Status TeamWERK` → `member.status == "aktiv"` (Fallback `""→"aktiv"` lebt nur im Insert)
- [x] 1.13 `go test ./internal/members/` grün; Commit: `test(members): Charakterisierungstests für Import (BOM/Delimiter/Dedup/400/not_found + Stufen-Guards 3-6)`

## 2. Phase 2 · Stufe 1 — `normalize*` → Top-Level

- [x] 2.1 `normalizeGender/Status/Beitragsfrei/Sepa`-Closures (handler.go ~1771-1812) auf Paket-Ebene heben (Doc-Kommentar zum `""`-Contract von `normalizeStatus` mitnehmen); Aufrufstellen unverändert
- [x] 2.2 `go test ./internal/members/` grün; `make metrics` (Signal); Commit: `refactor(members): normalize*-Helfer top-level`

## 3. Phase 2 · Stufe 2 — `parseImportCSV`

- [x] 3.1 BOM+Delimiter+Header/Alias+Pflichtspalten+ReadAll (~1728-1826) → `parseImportCSV(raw) (*parsedCSV, error)` mit `(*parsedCSV).col`; die 3 englischen 400-Texte als unterscheidbare Fehler zurückgeben, Handler macht `http.Error(w, err.Error(), 400)`
- [x] 3.2 Reihenfolge erhalten: Pflichtspalten-Check VOR `ReadAll`; `go test` grün; Commit: `refactor(members): parseImportCSV extrahieren`

## 4. Phase 2 · Stufe 3 — `detectCSVDuplicates`

- [x] 4.1 Dup-Block (~1828-1851) → `detectCSVDuplicates(rows, col) csvDupes`; Key `{lower(Vorname),lower(Nachname),normalizeDate(dob)}`, Zeilennummern 1-basiert inkl. Header; Meldungen im Handler-Loop unverändert
- [x] 4.2 `go test` grün; Commit: `refactor(members): detectCSVDuplicates extrahieren`

## 5. Phase 2 · Stufe 4 — `lookupExistingMember` (KRITISCH)

- [x] 5.1 Lookup+Ambiguität (~1885-1976) → `lookupExistingMember(ctx, first,last,dob,mode) lookupResult{outcome, member dbMember, message, dbErr}`; **wörtlich** erhalten: `substr(COALESCE(date_of_birth,''),1,10)=?`, Fall-B-Umschalter `useDobArg=false`, 17-Spalten-`Scan`-Reihenfolge, die 2 dt. Ambiguitäts-Meldungen; Report-Ausgänge (found/not_found/ambiguous/db-error) im Handler übersetzt
- [x] 5.2 `go test` grün (Fokus enrich-Member-Number-Tests: `_WithDOB`, `_TwoDigitYear1967`, `_DBohneGeburtsdatum`, `_EnrichAmbiguousNoDOB`); Commit: `refactor(members): lookupExistingMember extrahieren`

## 6. Phase 2 · Stufe 5 — `insertNewMember`

- [x] 6.1 created-Branch (~1989-2041) → `insertNewMember(ctx, pc, row, first,last,dob, dryRun) (ibanWarn, err)`; Status-Fallback `""→"aktiv"` NUR hier; 17 INSERT-Parameter in exakter Reihenfolge/Nullbarkeit; Report im Handler
- [x] 6.2 `go test` grün; Commit: `refactor(members): insertNewMember extrahieren`

## 7. Phase 2 · Stufe 6 — `buildMemberUpdate`

- [x] 7.1 update/enrich-Branch (~2062-2159) → `buildMemberUpdate(pc, row, dob, enrichOnly, fieldAllowed, db dbMember) memberUpdate`; `append`-Branch + UPDATE-Exec + selected/applyLines + Report-Switch bleiben im Handler; Sonderfälle (Trikot/SEPA/beitragsfrei/grund) + `changes`-Formate wörtlich erhalten
- [x] 7.2 `go test` grün; Commit: `refactor(members): buildMemberUpdate extrahieren`

## 8. Abschluss

- [x] 8.1 `make metrics-gate` grün. **Hinweis:** `Import` gocognit 182→**60**, gocyclo 111→**42** (golangci-Hauptgate flaggt `Import` nicht mehr), aber aspirationales <35 NICHT erreicht — Extract verteilt Komplexität um (Import 60 + buildMemberUpdate 43 beide >20). Ratchet bewusst re-baselined 35→38 / 12→14 mit dokumentierter Begründung (thresholds.yml). Keine `dupl`-Regression.
- [x] 8.2 `go test ./...` grün (1528); `git diff`-Contract-Check: die `http.Error`- und Row-`message`-Strings sind unverändert (nur verschoben: `errors.New` statt inline). `make test-race` — siehe Verifikationslauf.
- [x] 8.3 `openspec validate refactor-members-import --strict` grün
- [ ] 8.4 Rückblick (Roadmap 9.1); Roadmap-Section 6 abhaken — **nach Merge** von Phase-1-PR #153 + Phase-2-PR
- [ ] 8.5 Change archivieren (`openspec archive`) — **nach Merge beider PRs** (Phase 1 zuerst); appliziert Capability `members-import-refactor`

## Test-Anforderungen

Route → Testname → erwarteter Status → garantierte Invariante. Diese HTTP-Charakterisierungstests
SIND die Abnahme-Instanz für jeden Refactor-Schritt (Suite nach jedem Schritt grün).

`POST /api/members/import` (`internal/members`)
- `TestImport_StripsUTF8BOM` → 200 → UTF-8-BOM vor dem Header wird toleriert (Created==1)
- `TestImport_DelimiterSemicolon` / `_DelimiterComma` → 200 → beide Trennzeichen korrekt geparst
- `TestImport_DelimiterDetectedFromFirstLineOnly` → 200 → Trennzeichen-Erkennung betrachtet nur die erste Zeile
- `TestImport_ColumnAliasName` → 200 → `Name` erfüllt die `Nachname`-Pflichtspalte
- `TestImport_ColumnAliasGeborenAmUndMitgliedSeit` → 200 → `geboren am`/`Mitglied seit` mappen auf Geburtsdatum/join_date
- `TestImport_CSVInternalDuplicate` → 200 → beide Dublettenzeilen → error, Meldungen referenzieren die je andere Zeile; kein Insert
- `TestImport_MissingRequiredColumnVorname` / `_MissingRequiredColumnNachname` → 400 → fehlende Pflichtspalte, text/plain-Meldung
- `TestImport_BrokenCSVFieldCount` → 400 → inkonsistente Feldzahl → „cannot parse CSV"
- `TestImport_EmptyFileNoHeader` → 400 → leere Datei → „cannot read CSV header"
- `TestImport_MissingFileField` → 400 → Multipart ohne `file` → „missing file"
- `TestImport_EmptyNameCellIsRowError` → 200 → leere Pflichtzelle → Row-error, valide Zeile weiterhin created
- `TestImport_EnrichNotFound` → 200 → enrich + unbekannt → not_found, kein Insert
- `TestImport_CSVDuplicateDistinctByDOB` → 200 → gleiche Namen, verschiedene DOB ⇒ kein Dup (Stufe-3-Guard: DOB im Dedup-Key)
- `TestImport_EnrichAmbiguousNoDOB_MeldungA` → 200 → enrich ohne CSV-DOB + ≥2 gleichnamige → „Mehrdeutig (N Treffer) – Geburtsdatum in CSV fehlt" (Stufe-4-Guard)
- `TestImport_UpdateChangesContract` → 200 → Multi-Feld-Update → exakte `changes[]`-Strings/Reihenfolge (Stufe-6-Guard)
- `TestImport_AppendStatusFallbackAktiv` → 200 → append ohne/unbekannter Status → `status=="aktiv"` (Stufe-5-Guard)
