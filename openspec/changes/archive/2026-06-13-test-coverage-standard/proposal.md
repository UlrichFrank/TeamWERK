## Why

Die Testabdeckung liegt in kritischen Paketen bei 9–37 %. Wichtiger als die Zahl: Kernpfade wie `ChangePassword`, `ApproveMembershipRequest`, `GetProfile` und `Fulfill` — täglich genutzt oder sicherheitsrelevant — haben 0 % Coverage. Gleichzeitig fehlt ein Mechanismus, der verhindert, dass neue Changes ohne Tests gemergt werden. Dieser Change verankert den Test-Standard als Projektkonvention und schließt die identifizierten Lücken.

## What Changes

- **`make coverage`** — neues Makefile-Target: führt `go test -coverprofile` aus, gibt Package-Zusammenfassung auf stdout und öffnet HTML-Report
- **CLAUDE.md** — neuer Abschnitt „Test-Standard": Regel, dass jede neue Route ≥1 Happy-Path + ≥1 Fehlerfall-Test braucht; OpenSpec-Proposals müssen einen „Test-Anforderungen"-Abschnitt enthalten
- **Neue Tests** in `auth`, `members`, `duties`, `trainings`, `kader` für alle fachlich kritischen Lücken (≈25 neue Testfälle)

## Capabilities

### New Capabilities

- `test-standard-rule`: Projektkonvention + Tooling für reproduzierbare Coverage-Sichtbarkeit — `make coverage` als Standard-Workflow, Test-Anforderungen als Pflichtabschnitt in OpenSpec-Proposals
- `test-auth-gaps`: Tests für `ChangePassword` (Sicherheitsinvariante: altes PW, Session-Invalidierung), `ApproveMembershipRequest` / `RejectMembershipRequest` (Onboarding-Workflow), `ListUsers` (Admin-Paginierung)
- `test-members-gaps`: Tests für `GetProfile` und `UpdateProfile` (Spieler-Alltag: eigene Daten lesen/schreiben)
- `test-duties-gaps`: Tests für `Fulfill`, `CashSubstitute` (Dienstnachweis-Workflow) und `ListAssignments` (Trainer-Übersicht)
- `test-trainings-gaps`: Tests für `CreateSession`, `UpdateSession`, `DeleteSeries` (Trainer-Serienverwaltung mit Cascade)
- `test-kader-gaps`: Test für `CopyFromSeason` (jährlicher Saisonwechsel-Workflow)

### Modified Capabilities

*(keine)*

## Impact

- **Makefile**: neues Target `coverage`
- **CLAUDE.md**: neuer Abschnitt am Ende (keine bestehenden Regeln geändert)
- **Testdateien**: Ergänzungen in `internal/auth/handler_test.go`, `internal/members/handler_test.go`, `internal/duties/handler_test.go`, `internal/trainings/handler_test.go`, `internal/kader/handler_test.go`
- **Kein Produktionscode** wird verändert
- **Keine neuen Abhängigkeiten**
