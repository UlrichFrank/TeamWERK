## 1. Phase 1 — Charakterisierungstests (`internal/members/import_test.go`)

- [ ] 1.1 Test-Response-Struct `importReport` erweitern: pro Row `Message` (`json:"message"`), `DOB` (`json:"dob"`), `IBANWarning` (`json:"iban_warning"`) — additiv, bricht keinen Bestandstest
- [ ] 1.2 Inline-Multipart-Helfer für „ohne Datei-Feld" (nur `WriteField("mode","append")`), da `postImport`/`PostMultipart` immer ein `file`-Feld schreiben
- [ ] 1.3 BOM: `TestImport_StripsUTF8BOM` (BOM vor Header → 200, Created==1), optional `TestImport_BOMWithCommaDelimiter`
- [ ] 1.4 Delimiter: `TestImport_DelimiterSemicolon`, `TestImport_DelimiterComma`, `TestImport_DelimiterDetectedFromFirstLineOnly` (Detection nur erste Zeile)
- [ ] 1.5 Aliase: `TestImport_ColumnAliasName` (`Name`→`Nachname`), `TestImport_ColumnAliasGeborenAmUndMitgliedSeit`
- [ ] 1.6 Dedup: `TestImport_CSVInternalDuplicate` (beide Zeilen error, Meldungen „auch/zuerst Zeile N"), optional `TestImport_CSVDuplicateDistinctByDOB`
- [ ] 1.7 400-Pfade: `TestImport_MissingRequiredColumnVorname`, `_MissingRequiredColumnNachname`, `_BrokenCSVFieldCount`, `_EmptyFileNoHeader`, `_MissingFileField` (text/plain-Body prüfen, kein decode)
- [ ] 1.8 `TestImport_EmptyNameCellIsRowError` (leerer Vorname → Row-error, 200 gesamt, valide Zeile trotzdem created)
- [ ] 1.9 `TestImport_EnrichNotFound` (enrich + unbekannt → not_found, kein Insert, `name="Neu, Max"`, `dob="2007-10-14"`)
- [ ] 1.10 `go test ./internal/members/` grün; Commit: `test(members): Charakterisierungstests für Import (BOM/Delimiter/Dedup/400/not_found)`

## 2. Phase 2 · Stufe 1 — `normalize*` → Top-Level

- [ ] 2.1 `normalizeGender/Status/Beitragsfrei/Sepa`-Closures (handler.go ~1771-1812) auf Paket-Ebene heben (Doc-Kommentar zum `""`-Contract von `normalizeStatus` mitnehmen); Aufrufstellen unverändert
- [ ] 2.2 `go test ./internal/members/` grün; `make metrics` (Signal); Commit: `refactor(members): normalize*-Helfer top-level`

## 3. Phase 2 · Stufe 2 — `parseImportCSV`

- [ ] 3.1 BOM+Delimiter+Header/Alias+Pflichtspalten+ReadAll (~1728-1826) → `parseImportCSV(raw) (*parsedCSV, error)` mit `(*parsedCSV).col`; die 3 englischen 400-Texte als unterscheidbare Fehler zurückgeben, Handler macht `http.Error(w, err.Error(), 400)`
- [ ] 3.2 Reihenfolge erhalten: Pflichtspalten-Check VOR `ReadAll`; `go test` grün; Commit: `refactor(members): parseImportCSV extrahieren`

## 4. Phase 2 · Stufe 3 — `detectCSVDuplicates`

- [ ] 4.1 Dup-Block (~1828-1851) → `detectCSVDuplicates(rows, col) csvDupes`; Key `{lower(Vorname),lower(Nachname),normalizeDate(dob)}`, Zeilennummern 1-basiert inkl. Header; Meldungen im Handler-Loop unverändert
- [ ] 4.2 `go test` grün; Commit: `refactor(members): detectCSVDuplicates extrahieren`

## 5. Phase 2 · Stufe 4 — `lookupExistingMember` (KRITISCH)

- [ ] 5.1 Lookup+Ambiguität (~1885-1976) → `lookupExistingMember(ctx, first,last,dob,mode) lookupResult{outcome, member dbMember, message, dbErr}`; **wörtlich** erhalten: `substr(COALESCE(date_of_birth,''),1,10)=?`, Fall-B-Umschalter `useDobArg=false`, 17-Spalten-`Scan`-Reihenfolge, die 2 dt. Ambiguitäts-Meldungen; Report-Ausgänge (found/not_found/ambiguous/db-error) im Handler übersetzt
- [ ] 5.2 `go test` grün (Fokus enrich-Member-Number-Tests: `_WithDOB`, `_TwoDigitYear1967`, `_DBohneGeburtsdatum`, `_EnrichAmbiguousNoDOB`); Commit: `refactor(members): lookupExistingMember extrahieren`

## 6. Phase 2 · Stufe 5 — `insertNewMember`

- [ ] 6.1 created-Branch (~1989-2041) → `insertNewMember(ctx, pc, row, first,last,dob, dryRun) (ibanWarn, err)`; Status-Fallback `""→"aktiv"` NUR hier; 17 INSERT-Parameter in exakter Reihenfolge/Nullbarkeit; Report im Handler
- [ ] 6.2 `go test` grün; Commit: `refactor(members): insertNewMember extrahieren`

## 7. Phase 2 · Stufe 6 — `buildMemberUpdate`

- [ ] 7.1 update/enrich-Branch (~2062-2159) → `buildMemberUpdate(pc, row, dob, enrichOnly, fieldAllowed, db dbMember) memberUpdate`; `append`-Branch + UPDATE-Exec + selected/applyLines + Report-Switch bleiben im Handler; Sonderfälle (Trikot/SEPA/beitragsfrei/grund) + `changes`-Formate wörtlich erhalten
- [ ] 7.2 `go test` grün; Commit: `refactor(members): buildMemberUpdate extrahieren`

## 8. Abschluss

- [ ] 8.1 `make metrics-gate`: `Import` unter gocognit 35 / gocyclo 12; keine neue `dupl`-Regression
- [ ] 8.2 `make test-race` + `go test ./...` grün; `git diff` der `http.Error`- und Row-`message`-Strings gegen Original prüfen (unverändert)
- [ ] 8.3 `openspec validate refactor-members-import --strict` grün
- [ ] 8.4 Rückblick (Roadmap 9.1); Roadmap-Section 6 abhaken
- [ ] 8.5 Change archivieren (`openspec archive`) — appliziert Capability `members-import-refactor`

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
