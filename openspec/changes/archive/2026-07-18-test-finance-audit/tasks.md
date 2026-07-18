## 1. fee-run/confirm + protocol (`internal/beitragslauf/handler_test.go`)

- [x] 1.1 Imports ergänzen: `os`, `path/filepath`, `strings`, `io` (Datei-/Body-Lesen, Substring-Asserts)
- [x] 1.2 `TestConfirm_HappyPath_SchreibtProtokoll` — 200 + JSON (`saison_label`, `erfolgreich=1`, `summe_erfolgreich_cent`), Datei `beitragslauf_<label>.txt` enthält `Mitgl.-Nr`, Name, Euro-Betrag (`dir` aus `setupSrv`)
- [x] 1.3 `TestConfirm_ProtokollOhneIBAN` — **Struktur-Guard** (nicht Security-Kern; der Body trägt keine IBAN): Dateiinhalt enthält **nicht** `validIBAN`/`DE89`, **wohl aber** Mitgliedsnummer **und** Euro-Betrag (positive Inhaltsprüfung nagelt das Format mit)
- [x] 1.4 `TestConfirm_MixedSuccessFailure` — ein Erfolg + ein Fehlschlag in einem Batch → JSON `erfolgreich=1`, `nicht_erfolgreich=1`; Protokoll enthält **beide** Blöcke (`"Erfolgreich (1)"` + `"Nicht erfolgreich (1)"`, `protokoll.go:55-60`) — geldnaher als die Halbierungs-Zelle
- [x] 1.5 `TestConfirm_AppendOnly_ZweiLaeufe` — zwei Confirm-POSTs → `strings.Count(..., "=== Lauf bestätigt")==2`, erster Block bleibt
- [x] 1.6 `TestConfirm_UnbekannteSaison404` — Saison-ID 99999 → 404, keine Datei
- [x] 1.7 `TestConfirm_UngueltigerBody400` — typ-fehlpassendes/kaputtes JSON → 400, Body `"ungültiger Body"`
- [x] 1.8 `TestProtocol_RueckleseNachConfirm` — nach Confirm 200, `text/plain`, Body enthält Lauf-Block, **keine** IBAN
- [x] 1.9 `TestProtocol_UnbekannteSaison404` — 404
- [x] 1.10 `TestProtocol_OhneDateiLeer200` — gültige Saison ohne Lauf → 200 + leerer Body (nicht 404)
- [x] 1.11 Commit: `test(beitragslauf): fee-run confirm/protocol — Protokoll, Append-Only, keine IBAN`

## 2. fee-run/export-data 400-Fälle (`internal/beitragslauf/encryption_export_test.go`)

- [x] 2.1 `TestExportData_MitgliedOhneMandat400` — `defaultMember()` mit `sepaMandat=0` (IBAN bleibt) → 400, Body `"ausgeschlossen oder unbekannt"`
- [x] 2.2 `TestExportData_MitgliedOhneBankdaten400` — `defaultMember()` mit `iban=""` (kein Envelope) → 400
- [x] 2.3 `TestExportData_UnbekannteMemberID400` — `member_ids:[999999]` → 400, Body `"Mitglied 999999"`
- [x] 2.4 `TestExportData_UngueltigerBody400` — `saison_id:"keine-zahl"` (Decode-Fehler) → 400, Body `"ungültiger Body"`
- [x] 2.5 Commit: `test(beitragslauf): export-data lehnt Mitglied ohne Mandat/Bankdaten/unbekannt ab`

## 3. Preview — Halbierung-Restfall + Summen (`internal/beitragslauf/handler_test.go`)

- [x] 3.1 `TestPreview_UnterjaehrigerAustrittMitStammvereinAktivMit` — `status=ausgetreten`, `exit_date` im Fenster, `home_club_id` gesetzt, `join_date` vor Saison → `kategorie=aktiv_mit`, `half=true`, `half_reason=austritt`, `betrag_cent=4800`
- [x] 3.2 `TestPreview_SummaryTotals` — Preview-Aggregation (`handler.go:217-229`): definiertes Set aus einbezogenen + ausgeschlossenen Mitgliedern → `included_count`, `total_cent` (= Summe der einbezogenen Beträge), `excluded_cent`, `gesamtsumme_cent` korrekt. Der Kassierer-lesbare Einzugsbetrag ist bisher ungetestet (geldnahe Aggregation).
- [x] 3.3 Commit: `test(beitragslauf): Preview Halbierung aktiv_mit + Summen-Aggregation`

## 4. Abschluss

- [x] 4.1 `go test ./internal/beitragslauf/...` grün, dann `go test ./...` grün; `openspec validate test-finance-audit --strict` grün
- [x] 4.2 Rückblick (Roadmap 9.1): Risiko-/Churn-Bild nach Welle 2 neu bewerten; Roadmap-Section 5 abhaken
- [x] 4.3 Change archivieren (`openspec archive`) — appliziert Capability `fee-run-audit`

## Test-Anforderungen

Route → Testname → erwarteter Status → garantierte Invariante.

**fee-run/confirm** (`internal/beitragslauf`)
- `POST /api/fee-run/confirm` → `TestConfirm_HappyPath_SchreibtProtokoll` → 200 → Lauf schreibt Protokoll (Nr/Betrag/Erfolg)
- `POST …/confirm` → `TestConfirm_ProtokollOhneIBAN` → 200 → Protokoll enthält keine IBAN/Bankdaten (Struktur-Guard)
- `POST …/confirm` → `TestConfirm_MixedSuccessFailure` → 200 → Erfolg- und Fehlschlag-Block + Counts korrekt
- `POST …/confirm` → `TestConfirm_AppendOnly_ZweiLaeufe` → 200 → append-only, kein Überschreiben
- `POST …/confirm` → `TestConfirm_UnbekannteSaison404` → 404 → keine Datei bei unbekannter Saison
- `POST …/confirm` → `TestConfirm_UngueltigerBody400` → 400 → ungültiger Body

**fee-run/protocol** (`internal/beitragslauf`)
- `GET /api/fee-run/protocol` → `TestProtocol_RueckleseNachConfirm` → 200 → Rücklesen liefert Lauf-Block ohne IBAN
- `GET …/protocol` → `TestProtocol_OhneDateiLeer200` → 200 (leer) → gültige Saison ohne Lauf ≠ 404
- `GET …/protocol` → `TestProtocol_UnbekannteSaison404` → 404 → unbekannte Saison

**fee-run/export-data** (`internal/beitragslauf`)
- `POST /api/fee-run/export-data` → `TestExportData_MitgliedOhneMandat400` → 400 → kein Export ohne SEPA-Mandat
- `POST …/export-data` → `TestExportData_MitgliedOhneBankdaten400` → 400 → kein Export ohne Bankdaten-Envelope
- `POST …/export-data` → `TestExportData_UnbekannteMemberID400` → 400 → unbekannte Member-ID abgelehnt
- `POST …/export-data` → `TestExportData_UngueltigerBody400` → 400 → ungültiger Body

**Preview** (`internal/beitragslauf`)
- `GET /api/fee-run/preview` → `TestPreview_UnterjaehrigerAustrittMitStammvereinAktivMit` → 200 → Austritt + home_club → `aktiv_mit`, halbiert (4800), `half_reason=austritt`
- `GET /api/fee-run/preview` → `TestPreview_SummaryTotals` → 200 → Summen-Aggregation (`total_cent`/`excluded_cent`/`gesamtsumme_cent`/`included_count`) korrekt
