## 1. fee-run/confirm + protocol (`internal/beitragslauf/handler_test.go`)

- [ ] 1.1 Imports ergänzen: `os`, `path/filepath`, `strings`, `io` (Datei-/Body-Lesen, Substring-Asserts)
- [ ] 1.2 `TestConfirm_HappyPath_SchreibtProtokoll` — 200 + JSON (`saison_label`, `erfolgreich=1`, `summe_erfolgreich_cent`), Datei `beitragslauf_<label>.txt` enthält `Mitgl.-Nr`, Name, Euro-Betrag (`dir` aus `setupSrv`)
- [ ] 1.3 `TestConfirm_ProtokollOhneIBAN` — Dateiinhalt enthält **nicht** `validIBAN`/`DE89`, wohl aber die Mitgliedsnummer (Guard gegen versehentliche Bankdaten im Protokoll)
- [ ] 1.4 `TestConfirm_AppendOnly_ZweiLaeufe` — zwei Confirm-POSTs → `strings.Count(..., "=== Lauf bestätigt")==2`, erster Block bleibt
- [ ] 1.5 `TestConfirm_UnbekannteSaison404` — Saison-ID 99999 → 404, keine Datei
- [ ] 1.6 `TestConfirm_UngueltigerBody400` — typ-fehlpassendes/kaputtes JSON → 400, Body `"ungültiger Body"`
- [ ] 1.7 `TestProtocol_RueckleseNachConfirm` — nach Confirm 200, `text/plain`, Body enthält Lauf-Block, **keine** IBAN
- [ ] 1.8 `TestProtocol_UnbekannteSaison404` — 404
- [ ] 1.9 `TestProtocol_OhneDateiLeer200` — gültige Saison ohne Lauf → 200 + leerer Body (nicht 404)
- [ ] 1.10 Commit: `test(beitragslauf): fee-run confirm/protocol — Protokoll, Append-Only, keine IBAN`

## 2. fee-run/export-data 400-Fälle (`internal/beitragslauf/encryption_export_test.go`)

- [ ] 2.1 `TestExportData_MitgliedOhneMandat400` — `defaultMember()` mit `sepaMandat=0` (IBAN bleibt) → 400, Body `"ausgeschlossen oder unbekannt"`
- [ ] 2.2 `TestExportData_MitgliedOhneBankdaten400` — `defaultMember()` mit `iban=""` (kein Envelope) → 400
- [ ] 2.3 `TestExportData_UnbekannteMemberID400` — `member_ids:[999999]` → 400, Body `"Mitglied 999999"`
- [ ] 2.4 `TestExportData_UngueltigerBody400` — `saison_id:"keine-zahl"` (Decode-Fehler) → 400, Body `"ungültiger Body"`
- [ ] 2.5 Commit: `test(beitragslauf): export-data lehnt Mitglied ohne Mandat/Bankdaten/unbekannt ab`

## 3. Halbierungsmatrix-Restfall (`internal/beitragslauf/handler_test.go`)

- [ ] 3.1 `TestPreview_UnterjaehrigerAustrittMitStammvereinAktivMit` — `status=ausgetreten`, `exit_date` im Fenster, `home_club_id` gesetzt, `join_date` vor Saison → `kategorie=aktiv_mit`, `half=true`, `half_reason=austritt`, `betrag_cent=4800`
- [ ] 3.2 Commit: `test(beitragslauf): Halbierung aktiv_mit bei unterjährigem Austritt mit Stammverein`

## 4. Abschluss

- [ ] 4.1 `go test ./internal/beitragslauf/...` grün, dann `go test ./...` grün; `openspec validate test-finance-audit --strict` grün
- [ ] 4.2 Rückblick (Roadmap 9.1): Risiko-/Churn-Bild nach Welle 2 neu bewerten; Roadmap-Section 5 abhaken
- [ ] 4.3 Change archivieren (`openspec archive`) — appliziert Capability `fee-run-audit`

## Test-Anforderungen

Route → Testname → erwarteter Status → garantierte Invariante.

**fee-run/confirm** (`internal/beitragslauf`)
- `POST /api/fee-run/confirm` → `TestConfirm_HappyPath_SchreibtProtokoll` → 200 → Lauf schreibt Protokoll (Nr/Betrag/Erfolg)
- `POST …/confirm` → `TestConfirm_ProtokollOhneIBAN` → 200 → Protokoll enthält keine IBAN/Bankdaten
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

**Halbierung** (`internal/beitragslauf`)
- `GET /api/fee-run/preview` → `TestPreview_UnterjaehrigerAustrittMitStammvereinAktivMit` → 200 → Austritt + home_club → `aktiv_mit`, halbiert (4800), `half_reason=austritt`
