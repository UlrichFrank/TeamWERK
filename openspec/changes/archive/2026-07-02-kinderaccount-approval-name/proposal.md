## Why

Beim Bestätigen eines Kinder-Beitrittsantrags (`ApproveMembershipRequest` → `approveChildRequest`, `internal/auth/handler.go`) wird das Kinder-Konto **ohne Namen** angelegt. Das INSERT (`handler.go:521`) setzt `login_name`, aber nicht `first_name`/`last_name`:

```sql
INSERT INTO users (email, login_name, password, role, can_login, recovery_email)
  VALUES (NULL, ?, '', 'standard', 0, ?)
```

Die Werte `firstName`/`lastName` liegen als Funktionsparameter bereits vor, werden aber verworfen. Folge:

- `GET /api/users` liest den Namen direkt aus `users.first_name`/`last_name` (`handler.go:851`) → auf `/nutzer` bleibt die Namenszelle **leer** (Ausgangs-Bug).
- Die Impersonate-Antwort baut den Anzeigenamen aus `first_name`/`last_name` (`handler.go:989`) → leer.
- `CreateMemberFromUser` (`internal/members/handler.go`) übernimmt `first_name`/`last_name` aus der User-Zeile → legt man später „Mitglied anlegen" an, bekommt auch **das Mitglied einen leeren Namen**.

Der `login_name` ist als Namensquelle **verlustbehaftet** (Umlaut-Transliteration, Sonderzeichen-Strip, Kollisions-Suffix in `internal/auth/loginname.go`) und daher kein Ersatz. Die verlässliche Namensquelle sind die Antragsdaten in `membership_requests`.

Es existiert mindestens ein produktives Kinder-Konto, das bereits ohne Namen angelegt wurde → zusätzlich zum Code-Fix ist ein Backfill nötig.

**Nicht betroffen (bewusst):** Fehlt „Testen als" bei einem noch nicht aktivierten Kind (`can_login=0` → `proxy=true`), ist das kein Bug — ohne Zugriff braucht Impersonate nicht zu funktionieren. Sobald die Eltern das Passwort setzen (`can_login=1`), erscheint der Eintrag automatisch. Die Impersonate-UI-Regel (`!u.proxy`, `AdminUsersPage.tsx:915`) bleibt unverändert.

## What Changes

- **Invariante:** Beim Approve eines `is_child=1`-Antrags SHALL der erzeugte `users`-Datensatz `first_name`/`last_name` des Kindes tragen (aus dem Antrag). Der Name darf nach dem Approve nicht leer sein.
- **Code:** In `approveChildRequest` (`internal/auth/handler.go`) `first_name`/`last_name` mit ins `INSERT INTO users` aufnehmen.
- **Backfill:** Migration füllt bestehende namenlose Kinder-Konten (`can_login=0`, `login_name` gesetzt, `email IS NULL`, leerer `first_name`) aus `membership_requests` nach — konservativ nur bei eindeutigem Match, kein Rateverfahren über den verlustbehafteten `login_name`.
- **Tests:** Approve eines Kinderantrags → `users.first_name`/`last_name` gesetzt (Happy-Path); Bestandsverhalten (`login_name`, `can_login=0`, Eltern-Mail, kein `family_link`) bleibt grün.

**Diese Proposal wird vorerst NICHT umgesetzt** (nur angelegt).

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `kinderaccount-antrag`: (a) das beim Approve erzeugte Kinder-Konto trägt den echten Kindnamen in `users.first_name`/`last_name`; (b) Korrektur einer Fehlformulierung — die Requirement beschrieb bisher die Anlage eines `members`-Datensatzes, obwohl der Approve bewusst nur ein `users`-Konto anlegt.

## Test-Anforderungen

| Trigger | Testname (erwartet) | Status / Invariante |
|---|---|---|
| `POST /api/membership-requests/{id}/approve` mit `is_child=1` | `TestApproveChildRequest_SetsName` | 204; danach `users.first_name`/`last_name` == Antragsname, `login_name` gesetzt, `can_login=0` |
| Bestandsverhalten (Regress) | `TestApproveChildRequest_KeepsExistingBehavior` | Eltern-Mail versandt, **kein** `family_link`, Antrag-Status `approved` |
| Backfill-Migration `016` | `TestMigration016_BackfillChildNames` | eindeutig zuordenbares namenloses Konto wird gefüllt; mehrdeutiges/nicht matchbares bleibt leer (kein falscher Name) |

**Garantierte Invariante:** Nach dem Approve eines Kinderantrags ist der Name des Kinder-Kontos in `users.first_name`/`last_name` nicht leer.

## Impact

- **Code:** `internal/auth/handler.go` (`approveChildRequest`).
- **Migration:** neue `internal/db/migrations/016_*` (Daten-Backfill; nächste freie Nummer, aktuell 015).
- **API-Verhalten:** `GET /api/users` und Impersonate zeigen für Kinder-Konten künftig den Namen; keine Breaking Changes.
- **Spec-Korrektur:** die Requirement „Approve … erzeugt Konto, **Mitglied** und Eltern-Mail" wird ersetzt — Approve legt bewusst nur ein `users`-Konto an, „Mitglied" war eine Fehlformulierung. Nur Doku/Spec, kein Code-Verhalten geändert.
