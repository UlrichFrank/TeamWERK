## 1. Roadmap-Artefakte

- [x] 1.1 Proposal, Design und `test-strategy`-Spec eingecheckt und `openspec validate test-coverage-roadmap` grün
- [x] 1.2 Kurzverweis in `docs/agent/07-testing.md` auf die `test-strategy`-Capability ergänzen (ein Satz plus Link)
- [x] 1.3 Commit: `docs(openspec): Roadmap test-coverage-roadmap (Risiko×Churn, Wellen, Bug-vor-Test)`

## 2. Vorab — WIP-Hygiene & verifizierte Code-Bugs

- [x] 2.1 Tote In-Flight-Changes sichten und archivieren/abbrechen (`rename-mitfahrten`, `golangci-lint-v2-cleanup`, `harden-field-encryption-key`, …) — vor dem Auffahren neuer Wellen
- [x] 2.2 `test-coverage-fachlich` Section 3 (duties): Checkboxen an den real umgesetzten Stand angleichen (Spec/Code-Drift) — Section 3 komplett umgesetzt (20/20, Namens-Drift bei 3.1/3.14 dokumentiert); Change bleibt offen wegen Section 4 (members)
- [x] 2.3 Code-Fix `members.UpdateStatus`: RowsAffected → 404 bei unbekannter ID + `Exec`-Fehler prüfen (nicht verschlucken); Test `TestMemberStatus_NotFound404` absichert (eigener kleiner `fix(members)`-Commit)
- [x] 2.4 Entscheidung `files.checkAntiEscalation`: `newRead` gegen eigenen `can_read` durchgesetzt (Read-Escalation für write-ohne-read geschlossen) + Kommentar korrigiert; Invariante in `TestCheckAntiEscalation_*` (4 Fälle) festgenagelt

## 3. Welle 0 — `test-harness-preconditions` (Enabler)

- [x] 3.1 Change proposen (siehe eigener Ordner `test-harness-preconditions`) und `openspec validate` grün
- [x] 3.2 `internal/testutil/prodserver/prodserver.go`: `MatchReports`/`Settings`(+Store)/`Stammvereine` verdrahten wie in `cmd/teamwerk/main.go`; Nil-Guard für `/api/stammvereine`
- [x] 3.3 Zentrale Fixtures nach `internal/testutil`: `CreateFolder`, `SetFolderPermission`, `CreateFile`, `PostMultipart` (Server-Helper), `CreateAbsence`, `RecordTrainingAttendance`/`RecordGameAttendance`, `SetMemberBankEnvelope`, `SetClubSepaEnvelope`, `CreateMemberWithFields` (Options-Struct, ersetzt `CreateMember` NICHT)
- [x] 3.4 Authz-Drift-Detektor in `internal/arch` (analog `broadcast_test.go`): Erwartungs-Maps aus `permissions/matrix_test.go` ↔ `router.go` synchron; verwaiste Einträge failen
- [x] 3.5 Umsetzen, testen, archivieren — erwartetes Nebenresultat: Matrix-Test deckt jetzt matchreports/settings/stammvereine ab

## 4. Welle 1 — `test-pii-route-authz` (PII-Cluster, Route-Ebene)

- [x] 4.1 Proposal skizziert: files (Route-Tests CreateFolder/DeleteFolder/UploadFile/AddPermission/DeletePermission/Download-Token), absences (`Calendar?show_team`, Update/Delete-Authz, List-Fremdzugriff), matchreports (`images.go` ServeImage-Authz + Router-Tier), duties (`match_report_guard` inkl. Proxy-Kind-Rollenverschiebung). **Scope geschärft:** attendance-**Stats** bereits abgedeckt → nur **Recording** (`Training`/`Games.SaveAttendances`, Package `training`/`games`)
- [x] 4.2 `## Test-Anforderungen`-Abschnitt vorhanden: Route → Testname + erwarteter Status + garantierte Invariante (pro Bereich)
- [x] 4.3 `openspec validate test-pii-route-authz --strict` grün
- [x] 4.4 Umgesetzt (nutzt Welle-0-Fixtures), getestet, archiviert — 12 neue Tests über files/matchreports/duties/trainings/absences; games-Recording + attendance-Stats waren bereits abgedeckt (nicht dupliziert). Adversariales Review fand 1 False-Green (absences Calendar-Leak, per Mutations-Test verifiziert gefixt) + 4 schwache Assertions, alle gehärtet.

## 5. Welle 2 — `test-finance-audit` (+ optional auth-Fehlerpfade)

- [x] 5.1 Proposal: `fee-run/confirm` (Happy/404/400 + Protokoll-Schreiben, keine Bankdaten), `fee-run/protocol` (Rücklesen/404), `export-data`-400 (Mitglied ohne Mandat, unbekannte ID) + Halbierungsmatrix-Restfälle (unterjähriger Austritt + home_club)
- [x] 5.2 `auth`-Fehlerpfade: 6 Tests — E-Mail-Change-Session-Invalidierung (ConfirmEmailChange löscht refresh_tokens) + Passwort-Reauth (RequestEmailChange 403/204) + abgelaufener/manipulierter Access-Token (401) + abgelaufener Refresh-Token (401). Passwort-Reauth beim Passwort-Ändern war bereits abgedeckt (nicht dupliziert).
- [x] 5.3 `openspec validate` grün
- [x] 5.4 Umgesetzt, getestet, archiviert — 15 neue Tests in `internal/beitragslauf` (confirm/protocol + export-data-400 + Preview-Halbierung/Summen); keine neuen Fixtures. Zwei parallele Review-Agenten: keine False-Greens, drei geldnahe Härtungen eingearbeitet (summe_erfolgreich_cent, Confirm-404-keine-Datei, IBAN-Tripwire ehrlich kommentiert).

## 6. Welle 3 — `refactor-members-import` (Funktionserhalt-kritisch)

- [x] 6.1 Proposal: 14 Charakterisierungstests ZUERST (Delimiter/BOM, Dedup, Enrich-Ambiguität, Fehlerpfade 400, Authz `kassierer/standard → 403`), dann 6-Stufen-Extract (normalize* top-level → parseImportCSV → detectCSVDuplicates → lookupExistingMember → buildMemberUpdate)
- [x] 6.2 `## Test-Anforderungen`: die HTTP-Charakterisierungstests SIND die dauerhafte Abnahme-Instanz für jeden Refactor-Schritt (Suite nach jedem Schritt grün)
- [x] 6.3 `openspec validate` grün
- [x] 6.4 Umgesetzt, getestet, archiviert — zweiphasig: 19 Charakterisierungstests (PR #153), dann 6-Stufen-Extract (PR #155). `Import` gocognit 182→60, byte-genaue Verhaltenserhaltung (adversarial verifiziert, String-Contracts intakt). metrics-gate bewusst re-baselined (35→38/12→14, dokumentiert). Agenten: 2× Detail-Specs, 3× Review (fanden Charakterisierungs-Lücken + verschärften Pins), 1× Refactor, 1× Verhaltens-Verifikation.

## 7. Parallel — Frontend

> **Status (2026-07-18):** **Vollständig & real verifiziert** (echtes Chromium gegen Prod-Binary +
> Seed-DB, `7 passed`). Single-Origin-Setup (kein Vite-Proxy): `webServer` baut die Test-Binary,
> seedet via Go-Subcommand `e2e-seed`, serviert die eingebettete SPA + API auf :18080.
> `make test-e2e` verdrahtet, CI-`e2e`-Job parallel zum Gate (PR #161 Fundament, #162 Golden-Paths).

- [x] 7.1 `frontend-e2e-tests` Playwright-Setup: Playwright + Chromium installiert, `web/e2e/` (config/fixtures/tsconfig), `e2e-seed`-Subcommand, `make test-e2e` + `test:e2e`-Script, `.gitignore`, Doku. **Real grün.**
- [x] 7.2 Golden-Path-E2E: Login (`auth-login.spec.ts`, Login + Fehlerpfad) **+ Dienstbörse** (`duty-golden-path.spec.ts`: Login → Dienste → Slot eintragen/`Austragen` → Abmelden; Duty-Slot-Seed steht). **Real grün.**
- [x] 7.3 Golden-Path-E2E: Zero-Knowledge-Pfad als **Tresor-Setup-Smoke** (`vault-setup.spec.ts`: RSA-OAEP + PBKDF2 + AES-GCM im echten Browser-WebCrypto). Bewusster Scope: der Bank-Envelope-Schreibpfad ist bereits durch Go- + Vitest-Tests abgedeckt; E2E ergänzt die im Node-WebCrypto **nicht** testbare ausgelieferte Browser-Krypto. **Real grün.**
- [x] 7.4 chat-scroll-Killer-Cases (`chat-scroll.spec.ts`: Bild-Decode-Scroll — der ursprüngliche Bug-Vector, Unread-Divider, Deep-Link `?openUser`; Bild-Seed steht) + CI-`e2e`-Job. **Real grün.** (Entscheidung „eigener Frontend-Roadmap-Change" bleibt als Open Question in design.md — nicht nötig für diese Wellen.)

## 8. Nachgelagert (bewusst nach den Wellen)

- [x] 8.1 `venues`: 8 Tests — destruktive Routen (Delete 204/404/400, DeleteAll schont Home-Venue) + Import-Fehlerpfade + Upsert-Happy. 403-Authz war bereits via `permissions/matrix_test.go` abgedeckt (nicht dupliziert).
- [x] 8.2 `trainings`: DeleteSession-Authz (own/foreign/404) + Cross-Family + GetAttendances/GetSession/Create/UpdateSession-Fehlerpfade. **Dabei echten Broken-Access-Control-Bug in `Respond` gefunden & gefixt** (Rollen-Switch auf tote „spieler/elternteil" → jeder konnte fremde RSVP setzen); jetzt Ownership-Gate (own/parent/staff → sonst 403).
- [x] 8.3 `games`-Regen: **Komplexitäts-Vorbehalt geklärt** — `regenSingleDay` ist gocognit **124** (>> members.Import 60), Refactor gerechtfertigt, ABER voll black-box-testbar → **Tests zuerst** (5 Regen-Invarianten festgenagelt: template_id=NULL, Konflikt, removed-Notification, skipped-Inhalt, same_day reduced). **`regenSingleDay`-Refactor als eigener Folgeschritt vertagt** (analog Welle 3, jetzt durch die Tests abgesichert).

## 9. Roadmap-Kontrolle

- [x] 9.1 Rückblick nach jeder Welle erfolgt — Scope wurde mehrfach am realen Code geschärft (W1: attendance-Stats schon abgedeckt; W2: games-Recording schon durch; Nachgelagert: venues-Authz via matrix_test). Adversariale Reviews fingen pro Welle echte Mängel (False-Green W1, geldnahe Gaps W2, Charakterisierungs-Lücken W3, **zwei Broken-Access-Control-Bugs** im Nachgelagert-Teil).
- [x] 9.2 Roadmap wurde durchgehend explizit fortgeschrieben statt sklavisch abgearbeitet — u.a. Duplikate weggelassen, `regenSingleDay`-Refactor bewusst als eigener Folge-Change (nicht in 8.3 gequetscht), Frontend-E2E bewusst als eigene Session abgegrenzt.
- [ ] 9.3 Roadmap archivieren — **noch offen**: einziger Rest ist Abschnitt 7 (Frontend-E2E). Archivierung erst, wenn Welle 7 abgeschlossen ODER bewusst verworfen ist.
