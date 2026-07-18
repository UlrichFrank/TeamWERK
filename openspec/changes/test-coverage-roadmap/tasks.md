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
- [ ] 5.2 Optional als eigener kleiner Change / in `test-coverage-fachlich`: `auth`-Fehlerpfade (Session-Invalidierung nach E-Mail-Änderung, Passwort-Reauth, abgelaufener/manipulierter Token)
- [x] 5.3 `openspec validate` grün
- [x] 5.4 Umgesetzt, getestet, archiviert — 15 neue Tests in `internal/beitragslauf` (confirm/protocol + export-data-400 + Preview-Halbierung/Summen); keine neuen Fixtures. Zwei parallele Review-Agenten: keine False-Greens, drei geldnahe Härtungen eingearbeitet (summe_erfolgreich_cent, Confirm-404-keine-Datei, IBAN-Tripwire ehrlich kommentiert).

## 6. Welle 3 — `refactor-members-import` (Funktionserhalt-kritisch)

- [x] 6.1 Proposal: 14 Charakterisierungstests ZUERST (Delimiter/BOM, Dedup, Enrich-Ambiguität, Fehlerpfade 400, Authz `kassierer/standard → 403`), dann 6-Stufen-Extract (normalize* top-level → parseImportCSV → detectCSVDuplicates → lookupExistingMember → buildMemberUpdate)
- [x] 6.2 `## Test-Anforderungen`: die HTTP-Charakterisierungstests SIND die dauerhafte Abnahme-Instanz für jeden Refactor-Schritt (Suite nach jedem Schritt grün)
- [x] 6.3 `openspec validate` grün
- [x] 6.4 Umgesetzt, getestet, archiviert — zweiphasig: 19 Charakterisierungstests (PR #153), dann 6-Stufen-Extract (PR #155). `Import` gocognit 182→60, byte-genaue Verhaltenserhaltung (adversarial verifiziert, String-Contracts intakt). metrics-gate bewusst re-baselined (35→38/12→14, dokumentiert). Agenten: 2× Detail-Specs, 3× Review (fanden Charakterisierungs-Lücken + verschärften Pins), 1× Refactor, 1× Verhaltens-Verifikation.

## 7. Parallel — Frontend

- [ ] 7.1 `frontend-e2e-tests` Playwright-Setup abschließen (existierender Change)
- [ ] 7.2 Golden-Path-E2E: Login → Dashboard → Dienstbörse → Slot claimen → Logout
- [ ] 7.3 Golden-Path-E2E: Mitglied bearbeiten (Vorstand), Bank-Daten-Envelope schreiben (kein Klartext) — Zero-Knowledge-Pfad
- [ ] 7.4 Nach Abschluss: entscheiden, ob eine eigene Frontend-Roadmap-Change nötig ist (Open Question in design.md)

## 8. Nachgelagert (bewusst nach den Wellen)

- [ ] 8.1 `venues`: 403-Authz + destruktive Routen (`Delete`/`DeleteAll`) + Import-Fehlerpfade — billige Versicherung, niedrigster Churn
- [ ] 8.2 `trainings` DeleteSession + Cross-Family-Authz
- [ ] 8.3 `games`-Regen: erst Refactor-Vorbehalt für `regenSingleDay` klären (cog prüfen), dann Tests

## 9. Roadmap-Kontrolle

- [ ] 9.1 Nach jeder abgeschlossenen Welle: Rückblick — hat sich das Risiko-/Churn-Bild verschoben? Nächste Welle noch die richtige?
- [ ] 9.2 Wenn nach einer Welle die Welt anders aussieht, Roadmap explizit updaten oder archivieren — nicht sklavisch abarbeiten
- [ ] 9.3 Roadmap archivieren, wenn alle Wellen entweder abgeschlossen oder als „nicht mehr relevant" markiert sind
