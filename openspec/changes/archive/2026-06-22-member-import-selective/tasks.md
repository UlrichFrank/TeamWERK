## 1. Backend — Feld-Whitelist & Zeilen-Auswahl

- [x] 1.1 `fields`-Formfeld parsen → `fieldWhitelist`-Map + `fieldAllowed(column)`-Closure (leer/abwesend = alle erlaubt)
- [x] 1.2 `apply_lines`-Formfeld parsen → `applyLines`-Map (leer/abwesend = alle)
- [x] 1.3 `fieldAllowed`-Check in `addChange`/`addNullableChange` einbauen
- [x] 1.4 `fieldAllowed`-Check in den Inline-Blöcken (jersey_number, sepa_mandat, beitragsfrei↔status, iban)
- [x] 1.5 `selected`-Gate (`applyLines == nil || applyLines[lineNum]`) vor dem `UPDATE` und im Reporting

## 2. Backend — Status `skipped`

- [x] 2.1 `ImportReport` um `Skipped int json:"skipped"` erweitern
- [x] 2.2 Reporting-Block: abgewählte Zeile mit Änderungen (kein Dry-Run, nicht `selected`) → Status `skipped` statt `unchanged`, `report.Skipped++`
- [x] 2.3 `report.Total`-Summe um `Skipped` ergänzen

## 3. Backend — Tests (`internal/members/import_test.go`)

- [x] 3.1 `postImportOpts`-Helper mit Zusatz-Formfeldern (erledigt)
- [x] 3.2 `TestImport_FieldsWhitelist_NurIBAN` — nur `iban` geschrieben, `status` unverändert
- [x] 3.3 `TestImport_FieldsWhitelist_StatusSteuertBeitragsfrei` — `beitragsfrei` ohne `status` in `fields` unverändert
- [x] 3.4 `TestImport_LeeresFields_AlleFelder` — Regression: ohne `fields` alle Felder
- [x] 3.5 `TestImport_FieldsWhitelist_NeuesMitgliedAllFelder` — neues Mitglied trotz `fields=iban` voll angelegt
- [x] 3.6 `TestImport_ApplyLines_NurAusgewaehlteZeile` — nur Zeile 2 geschrieben
- [x] 3.7 `TestImport_ApplyLines_AbgewaehltIstSkipped` — Status `skipped` + Zähler + DB unverändert
- [x] 3.8 `TestImport_ApplyLines_DryRunIgnoriert` — Dry-Run meldet alle als `updated`, schreibt nichts

## 4. Frontend — Feld-Auswahl (`web/src/pages/MembersPage.tsx`)

- [x] 4.1 Konstante `IMPORT_FIELDS` (Label ↔ DB-Spalte) zentral definieren
- [x] 4.2 State `selectedFields: Set<string>` (default alle), Reset in `resetImport`
- [x] 4.3 Feld-Checkbox-Block in Schritt 1, nur sichtbar bei `mode=update`/`enrich` (alle/keine-Umschalter optional)
- [x] 4.4 `fields` in `handlePreview` und `handleImport` an FormData anhängen (nur bei update/enrich)

## 5. Frontend — Mitglieder-Auswahl & skipped-Darstellung

- [x] 5.1 State `selectedLines: Set<number>` aus `updated`-Zeilen der Vorschau initialisieren (default alle angehakt)
- [x] 5.2 Checkbox pro `updated`-Zeile in der Vorschau; „alle/keine"-Umschalter im Vorschau-Header
- [x] 5.3 `apply_lines` in `handleImport` aus `selectedLines` senden
- [x] 5.4 `ImportReport`-Typ um `skipped` erweitern; `skipped`-Status in `rowStatusIcon`/`rowStatusColor`
- [x] 5.5 Summary-Badge „X übersprungen" in Vorschau- und Ergebnis-Ansicht

## 6. Verifikation

- [x] 6.1 `go test ./internal/members/...` grün
- [x] 6.2 `pnpm -C web build` + `pnpm -C web lint` grün
- [x] 6.3 `openspec validate member-import-selective --strict`
- [ ] 6.4 `/verify-change` (Build/Test/Lint + Invarianten: Route→Tests, brand-Tokens, lucide-Icons)
