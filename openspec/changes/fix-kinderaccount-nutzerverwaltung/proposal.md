## Why

Der Merge `feat/kinderaccount-ohne-email` (#48) legt Kinder-Accounts mit `users.email = NULL` an (Login über `login_name`). `Login` und `Refresh` wurden dafür auf `COALESCE(NULLIF(email,''), login_name, '')` umgestellt — zwei Stellen, die die fehlende E-Mail abfangen, blieben dabei unbeachtet:

1. **„Testen als" (Impersonation) schlägt fehl.** `Impersonate` (`internal/auth/handler.go`) scannt `email` in ein nicht-nullbares `string`. Bei `email = NULL` bricht `Scan` ab und der Handler antwortet fälschlich mit `404 "user not found"`. Der „Testen als"-Klick auf ein aktiviertes Kinder-Konto (`can_login=1`) läuft damit ins Leere.

2. **„Löschen" wirkt wirkungslos.** Der Backend-`DeleteUser` löscht zwar erfolgreich (HTTP 204), aber (a) `handleDeleteUser` im Frontend ruft danach **kein** `refreshUsers()` auf, (b) `DeleteUser` ruft **kein** `h.hub.Broadcast(...)` (Verstoß gegen die SSE-Hard-Rule), und (c) `useLiveUpdates` der Nutzerverwaltung lauscht nur auf `members`, nicht auf Nutzer-Mutationen. Der gelöschte Account bleibt bis zum harten Reload in der Liste sichtbar — für den Vorstand wirkt das wie „lässt sich nicht löschen". Dieser Defekt betrifft alle Nutzer, fällt aber jetzt mit den Kinder-Accounts auf.

## What Changes

- **Impersonation für Konten ohne E-Mail**: `Impersonate` liest die Identität analog zu `Login`/`Refresh` über `COALESCE(NULLIF(email,''), login_name, '')` (per `sql.NullString` oder SQL-`COALESCE`) und gibt diesen Wert als Identitäts-Claim an `IssueAccessToken` weiter. Ein aktiviertes Kinder-Konto kann damit impersoniert werden; der Token trägt den `login_name` statt einer leeren E-Mail.
- **Lösch-Mutation wird live sichtbar**: `DeleteUser` ruft nach erfolgreichem Commit `h.hub.Broadcast("users")`. Die Nutzerverwaltung abonniert `users` via `useLiveUpdates` und lädt die Liste neu; zusätzlich ruft `handleDeleteUser` nach dem `DELETE` direkt `refreshUsers()` (sofortiges Feedback, unabhängig vom SSE-Round-Trip).

## Capabilities

### Modified Capabilities
- `admin-impersonation`: Impersonation funktioniert auch für Accounts ohne E-Mail (`email IS NULL`); als Identität dient der `login_name`.

### New Capabilities
- `nutzer-loeschen-live-update`: Das Löschen eines Nutzers spiegelt sich ohne manuellen Reload in der Nutzerverwaltung wider (Backend-Broadcast `users` + Frontend-Refresh).

## Impact

- **Backend** `internal/auth/handler.go`: `Impersonate` (NULL-sichere Identität), `DeleteUser` (`h.hub.Broadcast("users")` nach Commit).
- **Frontend** `web/src/pages/AdminUsersPage.tsx`: `handleDeleteUser` ruft `refreshUsers()`; `useLiveUpdates`-Callback reagiert zusätzlich auf `users`.
- **Keine Migration**, keine neuen Routen, keine neuen externen Dienste.
- **SSE**: bringt `DeleteUser` regelkonform zur Hard-Rule (jede Mutation broadcastet).

## Test-Anforderungen

| Route / Verhalten | Testname | Erwarteter Status | Garantierte Invariante |
|---|---|---|---|
| `POST /api/impersonate/{id}` auf Kinder-Konto (`email NULL`, `can_login=1`) | `TestImpersonate_ChildAccountWithoutEmail` | 200 | Antwort enthält gültiges JWT; Identitäts-Claim = `login_name`, kein 404 |
| `POST /api/impersonate/{id}` auf Standard-Konto mit E-Mail | `TestImpersonate_RegularUser` (Regression) | 200 | Identitäts-Claim = E-Mail (unverändert) |
| `POST /api/impersonate/{id}` auf Admin | `TestImpersonate_AdminRejected` (Regression) | 400 | Admin nicht impersonierbar |
| `DELETE /api/users/{id}` (beliebiges Konto) | `TestDeleteUser_Broadcast` | 204 | Genau ein `Broadcast("users")` wird nach erfolgreichem Commit ausgelöst |
| `DELETE /api/users/{id}` auf Kinder-Konto | `TestDeleteUser_ChildAccount` | 204 | User-Zeile entfernt; verknüpfter `members`-Datensatz bleibt mit `user_id = NULL` erhalten (kein FK-Fehler) |
| `DELETE /api/users/{id}` auf eigenes Konto | `TestDeleteUser_SelfRejected` (Regression) | 400 | Selbst-Löschung abgelehnt |
