# Design

## Kontext

`feat/kinderaccount-ohne-email` (#48) hat eine zweite Login-Achse eingeführt: `users.login_name` (`Vorname.Nachname`) für Konten ohne E-Mail. `users.email` ist für diese Konten `NULL`. `Login` und `Refresh` selektieren die Identität seither als `COALESCE(NULLIF(email,''), login_name, '')`. Zwei Stellen, die ebenfalls die User-Identität lesen, wurden nicht mitgezogen — daraus resultieren die zwei Defekte dieses Changes.

## Entscheidung 1 — Impersonation: NULL-sichere Identität

**Problem:** `Impersonate` scannt aktuell:

```go
var email, role, firstName, lastName string
... .Scan(&email, &role, &firstName, &lastName)   // Scan(NULL → *string) ⇒ Fehler ⇒ 404
```

**Optionen:**

| Option | Ansatz | Bewertung |
|---|---|---|
| A | `email` als `sql.NullString` scannen, dann manuell auf `login_name` zurückfallen | erfordert zusätzliche Spalte im SELECT + Verzweigung in Go |
| B (gewählt) | SQL übernimmt den Fallback: `SELECT COALESCE(NULLIF(email,''), login_name, ''), role, first_name, last_name` | identisch zu `Login`/`Refresh`, eine Quelle der Wahrheit, kein NULL-Scan |

**Gewählt: B.** Die Identität wird direkt im SQL aufgelöst — exakt das Muster, das `Login`/`Refresh` seit #48 verwenden. Der aufgelöste Wert wird als `email`-Parameter an `IssueAccessToken(...)` übergeben (der Parameter ist die Identitäts-Claim, nicht zwingend eine E-Mail). Damit trägt das Impersonation-JWT denselben Identitätswert, den das Kind bei normalem Login bekäme.

**Begründung:** Minimale, konsistente Änderung; keine neue Verzweigungslogik, die erneut driften könnte.

## Entscheidung 2 — Lösch-Mutation live spiegeln

**Problem:** Backend löscht (204), aber die Liste aktualisiert sich nicht. Drei zusammenwirkende Lücken: kein `Broadcast`, kein `refreshUsers()`, `useLiveUpdates` ignoriert Nutzer-Events.

**Entscheidung:** Defense-in-Depth über zwei Pfade:

1. **Backend** `DeleteUser` ruft nach `tx.Commit()` `h.hub.Broadcast("users")` — erfüllt die SSE-Hard-Rule und informiert alle offenen Sessions.
2. **Frontend** `handleDeleteUser` ruft nach dem `await api.delete(...)` direkt `refreshUsers()` (sofortiges Feedback im auslösenden Tab, unabhängig vom SSE-Round-Trip); der `useLiveUpdates`-Callback reagiert zusätzlich auf `"users"`, damit auch fremde Sessions aktualisieren.

**Event-Name:** `"users"` (nicht `"members"`). `members` ist bereits belegt und löst nur `loadInvitationsAndRequests()` aus — die Nutzerliste hängt an `usePagination('/users')` und braucht ein eigenes Event, sonst bliebe sie stumm.

**Warum beide Pfade?** `refreshUsers()` allein deckt den Tab des Vorstands ab; `Broadcast` hält weitere offene Sessions konsistent und macht `DeleteUser` regelkonform. Der direkte Refresh vermeidet außerdem, dass das Feedback vom SSE-Timing abhängt.

## Nicht-Ziele

- **Kein** Redesign der „Aktivieren"-Aktion für Kinder-Konten (würde eine E-Mail erzwingen — separat zu klären, falls überhaupt gewünscht).
- **Keine** Aufräumlogik für verwaiste `members`-Datensätze nach User-Löschung (`members.user_id` → `SET NULL` ist bestehendes, beabsichtigtes Verhalten).
- **Keine** Änderung an der Sichtbarkeitsbedingung des „Testen als"-Buttons (`!u.proxy`) — bei aktiviertem Kinder-Konto (`can_login=1`) ist `proxy=false`, der Button erscheint korrekt.

## Risiken

- **Gering.** Beide Änderungen sind lokal und additiv. Das `COALESCE` ist bereits in `Login`/`Refresh` erprobt. Der zusätzliche `Broadcast` folgt dem etablierten Hub-Muster.
