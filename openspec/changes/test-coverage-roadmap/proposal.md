## Why

Coverage steht bei **Go 43,0 %** (Report `metrics/REPORT.md`) und **Frontend n/a** (Vitest-Coverage ungemessen). In den ~4 Wochen seit dem letzten Report bewegte sich die Go-Zahl praktisch nicht (42,3 → 43,0 %), obwohl `test-coverage-fachlich` als „51/72" gilt — Beleg für die Projektregel, dass **Coverage-% ein irreführender Indikator** ist. Ohne strategische Reihenfolge und klare Nicht-Ziele drohen zwei Anti-Patterns: (a) Coverage-% als Ziel statt als Indikator, (b) Unit-Tests auf Code, der vor dem Testen refactored gehört (`members.Import`, ~532 Zeilen, cog~177, ist der Prototyp).

Eine parallele read-only Analyse-Fleet (15 spezialisierte Agenten, Juli 2026) hat die Blindspots empirisch vermessen. Vier Befunde ändern die ursprüngliche Reihenfolge:

- **Der Authz-Gate existiert schon — ist aber blind auf einer ganzen Domäne.** `internal/permissions/matrix_test.go` läuft bereits über den Router. Aber `internal/testutil/prodserver/prodserver.go` verdrahtet `h.MatchReports`/`h.Settings`/`h.Stammvereine` **nie** → die `if h.X != nil {…}`-Blöcke in `router.go` werden im Test-Router nie registriert → die komplette Spielbericht-Autorisierung ist unsichtbar. `/api/stammvereine` hat zudem **keinen** Nil-Guard → Nil-Panic → Recoverer → HTTP 500, das der `httpAllowed`-Check still als „bestanden" wertet. **Kleiner Fix, größter Hebel im ganzen Vorhaben.**
- **`internal/files` (938 LOC, 1 Testfile) — 9 von 12 Routen nur über die reine `resolveAccess`-Funktion getestet, nie über den HTTP-Handler.** `DownloadFile` (public gemountet), `AddPermission`/`checkAntiEscalation` (Rechte-Vergabe), `DeleteFolder`, `UploadFile` haben keinen Route-Test. Höchstes PII-Leak-Risiko im Repo; ACL wurde in 90 Tagen bereits einmal live gefixt.
- **`internal/beitragslauf`: `fee-run/confirm` und `fee-run/protocol` (der einzige rechtliche Audit-Nachweis eines SEPA-Laufs) sind komplett ungetestet**, ebenso der `export-data`-400-Ausschlusspfad (verhindert Lastschrift ohne Mandat).
- **Die Analyse hat drei mögliche Code-Bugs aufgedeckt (verifiziert):** `members.UpdateStatus` liefert **204 statt 404** für nicht-existente Mitglieder und verschluckt den DB-Fehler (P1, Code-Fix vor Test); `files.checkAntiEscalation` vergibt `can_read` ohne den eigenen `can_read` des Callers zu prüfen (P2); `files.DownloadFile`-Bearer-Pfad ist toter Code (fail-closed 401, **kein** Leck — reklassifiziert).

Zusätzlich blind: **`internal/venues`** (5 von 6 Routen inkl. destruktivem `DeleteAll`/`Delete` und Import-Parser nie getestet — von der ursprünglichen Roadmap übersehen).

## What Changes

Diese Roadmap ist **selbst kein Test-Code**, sondern legt Reihenfolge, Prinzipien und explizite Nicht-Ziele fest (durabel in der `test-strategy`-Capability). Priorisiert wird nach **Risiko × Churn × Hebel × Vorbedingung** — die ursprüngliche rein-risikobasierte Reihenfolge wird durch die Churn-Daten korrigiert (siehe design.md D6). Sie mündet in vier präzise, sequenzielle Wellen-Changes (jeweils eigener Proposal-Zyklus, **einer nach dem anderen**, nicht parallel aufgefahren):

1. **`test-harness-preconditions`** (Welle 0, Enabler) — `prodserver.go` fixen (MatchReports/Settings/Stammvereine verdrahten + Nil-Guard), zentrale Fixtures (`CreateFolder`, `SetFolderPermission`, `CreateFile`, `PostMultipart`, `CreateAbsence`, `Record*Attendance`, `Set*Envelope`, `CreateMemberWithFields`), Authz-Drift-Detektor in `internal/arch`. Macht die folgenden Wellen überhaupt erst testbar und aktiviert sofort Gate-Abdeckung für eine blinde Domäne.
2. **`test-pii-route-authz`** (Welle 1) — Route-Ebene-Autorisierung des PII-Clusters: `files` (12 Route-Tests), `absences` (`Calendar?show_team`, Update/Delete), `attendance` (Cross-Family/Trainer-falsches-Team), `matchreports` (`images.go` + Router-Tier, jetzt via Welle 0 sichtbar), `duties` (`match_report_guard` inkl. Proxy-Kind-Rollenverschiebung).
3. **`test-finance-audit`** (Welle 2a) — `fee-run/confirm`+`protocol`+`export-data`-400, Halbierungsmatrix-Restfälle. Parallel möglich: **`auth`-Fehlerpfade** (Session-Invalidierung, Reauth, Token-Ablauf/Signatur) als kleiner eigener Change oder in `test-coverage-fachlich`.
4. **`refactor-members-import`** (Welle 3) — Vorbedingung für Import-Tests: **14 Charakterisierungstests zuerst** (Blackbox, kein Codeeingriff), dann Extract-Method in 6 Stufen; die HTTP-Charakterisierungstests bleiben dauerhaft die Abnahme-Instanz. Enthält den einzigen Produktionscode-Eingriff.

**Parallel weiter**: `frontend-e2e-tests` (Playwright-Setup abschließen, dann Golden-Path Login → Dienstbörse → Claim + Mitglied-Bearbeiten mit Bank-Envelope/Zero-Knowledge).

**Code-Fix vorab (nicht Teil einer Test-Welle, aber Vorbedingung für ehrliche Tests):** `members.UpdateStatus` 204→404 + Fehlerprüfung; Entscheidung zu `checkAntiEscalation` (fix vs. dokumentieren).

Explizit **nicht** Teil der Roadmap:
- Vitest-Coverage-Zahl heben (misleading Metric — Playwright > Vitest-% für den Solo-Dev).
- Coverage-% als CI-Gate (widerspricht Projektregel „Coverage ist Indikator, kein Gate").
- Tests auf Hotspots mit cog>50, **bevor** dort refactored wurde (`members.Import`, ggf. `games.regenSingleDay`).
- `venues`/`trainings-DeleteSession`/`games-regen`: bewusst nachgelagert (billige Versicherung bzw. Refactor-Vorbehalt).

## Capabilities

### New Capabilities

- `test-strategy`: Verbindliche Grundsätze für neue Tests im Projekt (Prinzipien, Nicht-Ziele, Refactor-vor-Test, Bug-vor-Test-Verifikation, Arch-Test-Präferenz, Authz-Gate über Produktions-Router). Referenzdokument, das nachfolgende Test-Changes verpflichtet.

### Modified Capabilities

*(keine — bestehende Fach-Specs bleiben unverändert)*

## Impact

- **Betroffen**: `openspec/` (neue Change-Ordner in Folge-Iterationen), `docs/agent/07-testing.md` (Verweis auf Strategie-Spec ergänzen).
- **Produktionscode nur an drei eng umrissenen Stellen**: der `members.UpdateStatus`-Fix (Vorab), der `members.Import`-Refactor (Welle 3, mit Charakterisierungsnetz), sowie `internal/testutil/prodserver` (Test-Harness, kein Laufzeitverhalten). Alle übrigen Wellen fügen ausschließlich `*_test.go` + `testutil`-Helfer hinzu → Funktionalität bleibt unberührt.
- **CI**: keine Änderung an Gates in diesem Change. Ein optionaler späterer Ratchet-Schritt (Coverage darf nur steigen) ist bewusst nicht Teil dieser Roadmap.
- **WIP-Hygiene (Empfehlung)**: Es sind **23 Changes in-flight**, mehrere tot (`rename-mitfahrten` 0/46, `golangci-lint-v2-cleanup` 0/25, `harden-field-encryption-key` 0/29). Bevor vier neue Wellen-Changes dazukommen, sollten die Leichen archiviert/abgebrochen werden — sonst frisst genau das die Roadmap, wovor D5 warnt.
- **Team-Kontext**: Solo-Dev. Priorisierung optimiert daher **Wartungslast der Tests** gleich stark wie Bug-Fang — begünstigt Arch-Gates und Charakterisierungstests gegenüber tiefen Unit-Bäumen.
