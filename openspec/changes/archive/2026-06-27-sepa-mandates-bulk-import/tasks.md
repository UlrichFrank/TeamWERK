## 1. Backend: Match-Helper & Tests

- [x] 1.1 In `internal/upload/` neue Datei `sepa_match.go` mit Funktion `normalizeName(s string) string` (lowercase, Umlaut-Substitution `ä→ae`, `ö→oe`, `ü→ue`, `ß→ss`, Strip `' ', '-', '_', '.', '\''`).
- [x] 1.2 In `sepa_match.go` Funktion `matchMemberByFilename(db *sql.DB, basename string) (matched []int, err error)` — lädt alle `(id, first_name, last_name)` aus `members`, baut Lookup-Map `normalizeName(first+last)` und `normalizeName(last+first)` → `[]int`, gibt **alle** Treffer für `normalizeName(basename)` zurück. Caller entscheidet anhand der Länge (`0`/`1`/`>1`) den Status.
- [x] 1.3 Tests in `sepa_match_test.go`: Happy-Path (eindeutig), Umlaut-Normalisierung, Reverse-Reihenfolge, Bindestrich/Apostroph im Namen, Ambiguität (zwei IDs zurückgegeben), No-Match (leeres Slice).

## 2. Backend: Bulk-Import-Handler

- [x] 2.1 In `internal/upload/handler.go` Konstante `maxBulkBytes = 50 << 20` und Datei-Limit-Konstante (`maxSepaBytes = 2 << 20` bereits vorhanden).
- [x] 2.2 In `internal/upload/handler.go` neue Methode `(h *Handler) BulkImportSepaMandate(w http.ResponseWriter, r *http.Request)`:
  - `r.Body = http.MaxBytesReader(nil, r.Body, maxBulkBytes+1024)`
  - `r.ParseMultipartForm(8 << 20)` (Disk-Spill ab 8 MB)
  - Match-Lookup einmal pro Request via `matchMemberByFilename`-Helper
  - Iteration über `r.MultipartForm.File["files"]`
  - Per File: MIME-Check (`application/pdf`, mit Magic-Byte-Fallback `%PDF`), Size-Check (≤ `maxSepaBytes`), Match-Lookup, Skip-Prüfung (`SELECT sepa_mandat_path FROM members WHERE id=?`), Speichern via `saveFile`-Logik (refactor: kleinen Helper `(h *Handler) saveFileFromHeader(hdr, subdir, allowedTypes, maxBytes)` extrahieren), DB-`UPDATE` (`sepa_mandat_path`, `sepa_mandat=1`), Rollback der Datei bei DB-Fehler.
  - Report-Struktur `{imported, already_exists, no_match, ambiguous}` aufbauen und als JSON zurückgeben.
- [x] 2.3 Hub-Feld in `upload.Handler` ergänzen (`hub *hub.EventHub`) — Konstruktor anpassen, `cmd/teamwerk/main.go` & alle Aufrufer mit Hub durchreichen. Nach mindestens einem `imported`-Eintrag `h.hub.Broadcast("members-updated")`.
- [x] 2.4 Route in `internal/app/router.go` unter dem `Vorstand+Kassierer`-Tier eintragen: `r.Post("/api/members/sepa-mandates/import", h.Upload.BulkImportSepaMandate)`.
- [x] 2.5 `internal/permissions/matrix_test.go`: neuer Eintrag `{method: "POST", path: "/api/members/sepa-mandates/import", expected: exVorstandKassierer}`.

## 3. Backend: Handler-Tests

- [x] 3.1 Test-Fixture-Erweiterung in `internal/testutil/`: Helper `CreateMemberWithName(t, db, firstName, lastName)` (falls noch nicht vorhanden, sonst Reuse von `CreateMember`).
- [x] 3.2 In `internal/upload/bulk_sepa_test.go` (neue Datei) Tests entlang der Test-Anforderungen aus `proposal.md` plus zusätzlich `TestBulkImport_NoBroadcastWhenNothingImported`.
- [x] 3.3 Multipart-Test-Helper: `pdfBody(size)` + `postBulk(...)` mit `multipart.Writer` und realem `%PDF`-Header.

## 4. Frontend: Modal & Submit

- [x] 4.1 In `web/src/pages/MembersPage.tsx` neuen State-Block: `showSepaBulk`, `sepaFiles`, `sepaImporting`, `sepaReport`, `sepaError`, `sepaInputRef`.
- [x] 4.2 Dropdown-Eintrag „Import SEPA-Mandate" zwischen „Import CSV" und „Export CSV" — sichtbar wenn `isAdmin || hasCapability('manage_fees')` (deckt vorstand + kassierer + admin).
- [x] 4.3 Modal mit `<input type="file" webkitdirectory multiple accept="application/pdf">` + Button „Verzeichnis wählen". PDF-Filter clientseitig.
- [x] 4.4 Submit-Handler: `FormData.append('files', f)`; `api.post('/members/sepa-mandates/import', fd)`; Report → State; HTTP-Fehler in `sepaError`.
- [x] 4.5 Report-Anzeige mit vier Sektionen (`SepaBulkSection` + `SepaBulkAmbiguousSection`). Schließen-Button resettet, erfolgreicher Import triggert `refresh()` automatisch.
- [x] 4.6 Buttons & Inputs nutzen brand-Tokens (Primary `bg-brand-yellow text-brand-black`, Modal `bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow`, Alert `bg-brand-danger-light border-brand-danger/30`).

## 5. Frontend: Live-Updates & Tests

- [x] 5.1 Live-Updates: Backend broadcastet das bestehende `"members"`-Event; `MembersPage.tsx` lauscht bereits (`useLiveUpdates((event) => { if (event === 'members') refresh() })`). Spec/Tests entsprechend angepasst.
- [ ] 5.2 Vitest-Tests für das Modal (optional; bestehende 384 Tests grün, MembersPage hat keine eigene Test-Datei — manueller Smoke-Test deckt den Pfad).

## 6. Verifikation & Doku

- [x] 6.1 `go test ./...` (inkl. Architektur-Test) grün — 31 Packages ok.
- [x] 6.2 `pnpm -C web build && pnpm -C web test && pnpm -C web lint` grün (384 Vitest-Tests, Build OK).
- [x] 6.3 `openspec validate sepa-mandates-bulk-import --strict` grün.
- [ ] 6.4 Manueller Smoke-Test lokal (vom Nutzer durchzuführen): Verzeichnis mit 3 PDFs hochladen, Report verifizieren, DB-Stand prüfen.
- [x] 6.5 Eintrag in `web/public/CHANGELOG.md` ergänzt.
- [ ] 6.6 `/verify-change` durchlaufen lassen (optional — manueller Trigger).
- [ ] 6.7 Conventional-Commits + Archive nach Merge (offen für den Nutzer).
