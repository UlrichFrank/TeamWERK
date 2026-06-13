## Why

Die fachlich kritischen Domänen `auth`, `duties` und `members` haben null Testabdeckung, obwohl sie Sicherheits- und Geschäftslogik mit konkreten Invarianten enthalten (Token-Rotation, Audience-Filterung, Dienstbörse-Kapazität, Familienlink-Grenzen). Fehler dort fallen erst im Produktionsbetrieb auf. Der Testfall-Report unter `docs/test-report.md` dokumentiert 58 Testfälle, die aus dem echten Code heraus verifiziert wurden — jetzt werden sie implementiert.

## What Changes

- **Neu:** `internal/auth/handler_test.go` — 15 Tests für Login, Token-Refresh, Logout, Register, ForgotPassword, ResetPassword, UpdateUserRole, DeleteUser
- **Neu:** `internal/duties/handler_test.go` — 18 Tests für Claim/Unclaim, Board-Audience-Filter, Dienstkonten, Slot-CRUD
- **Neu:** `internal/members/handler_test.go` — 10 Tests für Mitgliederliste, Familienlinks, Proxy-Accounts
- **Neu:** `internal/kader/handler_test.go` — 5 Tests für AutoAssign (Bracket-Logik) und MemberSuggestions
- **Erweiterung:** `internal/games/handler_test.go` — 3 Tests für `ListTeamsForUser` (Trainer/Admin/Spieler)
- **Erweiterung:** `internal/trainings/handler_test.go` — 2 Tests für GetAttendances und Eltern-RSVP
- **Erweiterung:** `internal/absences/handler_test.go` — 2 Tests für Autorisierungsgrenzfall und leeres Preview
- **Erweiterung:** `internal/chat/handler_test.go` — 3 Tests für LeaveConversation-Varianten

Kein Produktionscode wird verändert. Kein neuer Handler, keine neue Route, keine Migration.

## Capabilities

### New Capabilities

- `test-auth`: Testsuite für das auth-Package — Login (inkl. Proxy-Account-Sperre), Token-Rotation, Passwort-Reset-Lifecycle, Rollenänderung, Nutzer-Cascade-Löschung
- `test-duties`: Testsuite für das duties-Package — Claim/Unclaim-Kapazitätsverwaltung, Board-Audience-Filterung nach Rolle, Dienstkonto-Sichtbarkeit, Slot-is_custom-Verhalten
- `test-members`: Testsuite für das members-Package — Paginierung, Namenssuche, Ausgetreten-Filter, Trainer-Scope, Familienlink-Grenzen, Proxy-Account-Erstellung
- `test-kader-handler`: Testsuite für die kader-Handler-Logik — AutoAssign mit DHB-Jahrgangs-Brackets, dedicated_birth_year, MemberSuggestions-Filter

### Modified Capabilities

*(keine — bestehende Specs bleiben unverändert)*

## Impact

- **Betroffen:** `internal/auth/`, `internal/duties/`, `internal/members/`, `internal/kader/`, `internal/games/`, `internal/trainings/`, `internal/absences/`, `internal/chat/`
- **Testinfrastruktur:** Bestehende `internal/testutil/`-Helfer werden genutzt; ggf. kleine Ergänzungen (z.B. `CreateDutyType`, `CreateDutySlot`, `CreateInvitationToken`)
- **Keine API-Änderungen**, keine neuen Abhängigkeiten, kein Frontend-Impact
- **CI:** `go test ./...` läuft nach der Änderung grün durch
