## Context

38 Tests existieren bereits (kader/age_brackets, games, absences, trainings, chat, notify). Die drei fachlich schwergewichtigen Pakete `auth`, `duties` und `members` haben 0 % Testabdeckung. Das `internal/testutil`-Package ist fertig ausgebaut: `NewDB` öffnet eine In-Memory-SQLite mit allen Migrationen, `NewServer` betreibt einen httptest-Server mit auth-Middleware, Helfer wie `CreateUser`, `CreateMember`, `CreateSeason`, `CreateGame`, `CreateKader`, `Token` decken den Großteil des Setup-Codes ab.

## Goals / Non-Goals

**Goals:**
- 58 neue Integrationstests, die echte Handler gegen eine echte (in-memory) SQLite-Datenbank testen
- Fachliche Korrektheit im Vordergrund: jeder Test prüft eine konkrete Geschäftsregel aus dem Code
- `go test ./...` läuft grün, kein Flaky-Test

**Non-Goals:**
- Kein Mocking von Datenbankaufrufen
- Keine Frontend-Tests
- Keine End-to-End-Tests gegen den echten VPS
- Kein Coverage-Tooling oder CI-Gate (wird separat entschieden)

## Decisions

**D1 — Integrationstests statt Unit-Tests**
Alle Handler werden via `httptest.Server` gegen echte SQLite-DB getestet (wie in games/ und trainings/). Unit-Tests einzelner Funktionen nur wo sinnvoll (z.B. `autoAssignMembers`). Begründung: Der Code hat kaum extrahierbare Hilfslogik; die Hauptkomplexität liegt in den SQL-Queries — diese müssen gegen die reale DB geprüft werden.

**D2 — testutil erweitern, nicht duplizieren**
Fehlt ein Fixture-Helfer (z.B. `CreateDutyType`, `CreateDutySlot`, `CreateInvitationToken`), wird er in `internal/testutil/fixtures.go` ergänzt. Paket-lokale Helfer (z.B. `insertDutyAssignment`) bleiben in der jeweiligen `_test.go`.

**D3 — Ein Commit pro Paket**
Jedes Test-Paket bekommt einen eigenen `test(<scope>): ...`-Commit. Das hält die Diffs lesbar und macht Bisect einfach.

**D4 — Mailer wird nicht gemockt**
`auth.NewHandler` erwartet einen `mailer.Mailer`-Interface. Für Tests wird eine No-op-Implementierung (`testutil.NoopMailer`) verwendet, die alle Mails verwirft. Kein SMTP-Server erforderlich.

**D5 — Auth-Tests ohne echten JWT-Geheimnis-Wechsel**
Tests verwenden `testutil.TestJWTSecret` (bereits vorhanden). Refresh-Token-Tests senden den opaque Token als Cookie und prüfen DB-Zustand direkt.

## Risks / Trade-offs

- [Mailer-Interface fehlt in testutil] → `testutil.NoopMailer{}` hinzufügen oder prüfen ob bereits vorhanden
- [ForgotPassword sendet asynchron E-Mail] → Test prüft nur den DB-Zustand (token angelegt), nicht den E-Mail-Versand
- [DeleteSlot-Push-Notification ist fire-and-forget] → Test prüft, dass `notify.Send` aufgerufen wurde, indem er den DB-Eintrag in `notification_log` prüft (falls vorhanden) oder den Slot-Count
- [duty_accounts.ist wird von Fulfill() nicht aktualisiert] → Test dokumentiert dieses Verhalten explizit als bekannte Invariante
