## 1. Roadmap-Artefakte

- [ ] 1.1 Proposal, Design und Test-Strategie-Spec eingecheckt und `openspec validate test-coverage-roadmap` grün
- [ ] 1.2 Kurzverweis in `docs/agent/07-testing.md` auf die neue `test-strategy`-Capability ergänzen (ein Satz plus Link)
- [ ] 1.3 Commit: `docs(openspec): Roadmap test-coverage-roadmap (Prio, Nicht-Ziele, Refactor-vor-Test)`

## 2. Phase 1 — Bestehendes zu Ende bringen (kein neuer Change nötig)

- [ ] 2.1 `test-coverage-fachlich` Section 3 (duties) abarbeiten — 18 Tests, siehe dort
- [ ] 2.2 `test-coverage-fachlich` Section 4 (members) abarbeiten — 10 Tests, siehe dort
- [ ] 2.3 `test-coverage-fachlich` Section 5 (kader) abarbeiten — 5 Tests
- [ ] 2.4 `test-coverage-fachlich` Sections 6–9 (games/trainings/absences/chat) abarbeiten
- [ ] 2.5 `test-coverage-fachlich` archivieren (`openspec archive`)

## 3. Phase 2 — Neuer Change `test-files-permissions` proposen

- [ ] 3.1 Proposal skizzieren: Ordner-CRUD-Autorisierung, Datei-Upload/Download-Rechte, `everyone.can_read/write`-Flags, Rekursion, Anleitungen-Bilder-Pfad (`/dokumente/datei/{fileId}` in Markdown)
- [ ] 3.2 Design festhalten: `internal/testutil/`-Helfer für Datei-Fixtures (`CreateFolder`, `CreateFile`, `SetFolderPermission`), In-Memory-Storage-Adapter oder Temp-Dir
- [ ] 3.3 Tasks pro Test-Case: Vorstand-CRUD, Trainer-Ordner-Sicht, Elternteil-nur-Gelesen, `everyone`-Ordner (Anleitungen), 403 bei fehlender Berechtigung, Datei-Löschung mit Berechtigungs-Fallback
- [ ] 3.4 `openspec validate test-files-permissions` grün
- [ ] 3.5 Umsetzen, testen, archivieren

## 4. Phase 3 — Neuer Change `test-absences-attendance` proposen

- [ ] 4.1 Proposal skizzieren: Autorisierungsgrenzfälle (Trainer sieht nur eigenes Team, Eltern für Kind), Kalender-Aggregation, Preview-Endpoint, RSVP-Konfliktfälle
- [ ] 4.2 Design: nutzt `testutil.CreateAbsence`/`CreateAttendance` (ggf. neu), keine Frontend-Änderung
- [ ] 4.3 Tasks pro Handler-Route mit Happy-Path + Fehlerfall (siehe `docs/agent/07-testing.md`)
- [ ] 4.4 `openspec validate test-absences-attendance` grün
- [ ] 4.5 Umsetzen, testen, archivieren

## 5. Phase 4 — Neuer Change `test-authz-arch-gate` proposen

- [ ] 5.1 Proposal skizzieren: Arch-Test in `internal/arch/authz_test.go` analog `broadcast_test.go`; parst `BuildRouter`, erkennt `RequireRole`/`RequireClubFunction`, prüft dass das Ziel-Package einen Test mit erwartetem 401-oder-403-Assertion enthält
- [ ] 5.2 Design: Allowlist-Struktur mit Begründung pro Ausnahme; Präzedenzfall dokumentieren (öffentlich zugängliche Routen, Legacy)
- [ ] 5.3 Tasks: Parser, Erkennungs-Regeln, Allowlist-Skelett, Test-Fixtures, Doku-Update
- [ ] 5.4 `openspec validate test-authz-arch-gate` grün
- [ ] 5.5 Umsetzen: erwartetes Ergebnis — Gate meldet konkrete Lücken, die dann als kleine PRs pro Package geschlossen werden
- [ ] 5.6 Archivieren

## 6. Phase 4 parallel — Frontend

- [ ] 6.1 `frontend-e2e-tests` Playwright-Setup abschließen (existierender Change)
- [ ] 6.2 Golden-Path-E2E: Login → Dashboard → Dienstbörse → Slot claimen → Logout (in Change `frontend-e2e-tests` oder Folge-Change)
- [ ] 6.3 Golden-Path-E2E: Mitglied bearbeiten (Vorstand), Bank-Daten-Envelope schreiben (kein Klartext) — Zero-Knowledge-Pfad
- [ ] 6.4 Nach Abschluss: entscheiden, ob eine eigene Frontend-Roadmap-Change nötig ist (siehe Open Question in design.md)

## 7. Roadmap-Kontrolle

- [ ] 7.1 Nach jeder abgeschlossenen Phase: Rückblick — hat sich das Risiko-Bild verschoben? Nächste Phase noch die richtige?
- [ ] 7.2 Wenn nach Phase 1 die Welt anders aussieht (z.B. ein Bug außerhalb der Prio-Liste), Roadmap explizit updaten oder archivieren — nicht sklavisch abarbeiten
- [ ] 7.3 Roadmap archivieren, wenn alle vier Phasen entweder abgeschlossen oder als „nicht mehr relevant" markiert sind
