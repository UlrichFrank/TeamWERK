## Context

Die Nutzerverwaltung (`AdminUsersPage`) besitzt aktuell einen „+ Einladung"-Button, der synchron ein `invitation_token` anlegt und sofort eine Einladungs-E-Mail verschickt. Es gibt keinen Bulk-Import-Weg.

Die Mitgliederliste liegt als CSV mit den Feldern `Email` und `Email 2` vor. Viele Mitglieder haben zwei E-Mail-Adressen (z.B. Eltern-Email + Kind-Email). Insgesamt ~227 unique Adressen bei 181 Personen.

Aktueller Datenfluss:
```
POST /auth/invite → invitation_tokens (INSERT) + SMTP-Versand → 204
POST /auth/register (token) → users (INSERT) → 200
```

Ziel ist es, den Import-Schritt vom E-Mail-Versand zu trennen und eine Vorab-Verknüpfung Einladung↔Mitglied zu ermöglichen.

## Goals / Non-Goals

**Goals:**
- CSV-Upload legt Tokens für alle neuen unique E-Mails an, ohne E-Mail zu versenden
- Einladungs-E-Mail nur on demand (ActionMenu)
- Einladungen können vor Registrierung mit einem `members`-Eintrag verknüpft werden
- Beim Registrieren wird die Verknüpfung automatisch auf den neuen User übertragen
- `last_login_at` auf `users` wird bei jedem Login gesetzt und in der UI angezeigt

**Non-Goals:**
- Kein Import von Namen, Telefonnummern oder anderen CSV-Feldern
- Keine automatische Mitglied-Zuordnung aus der CSV (zu fehleranfällig)
- Kein SMTP-Versand ohne explizite Admin-Aktion
- Kein Re-Invite für abgelaufene Tokens beim CSV-Import (nur on demand über ActionMenu)

## Decisions

### 1. CSV-Parsing: Backend vs. Frontend

**Entscheidung:** Backend parst die CSV.

Frontend sendet die CSV-Datei als `multipart/form-data` an `POST /api/admin/invitations/import-csv`. Das Backend liest `Email` und `Email 2`, dedupliziert, prüft gegen `users` und `invitation_tokens` und legt neue Tokens an.

**Alternative:** Frontend parst CSV mit `papaparse`, sendet nur die Liste der E-Mails. Verworfen: eine weitere Frontend-Dependency (papaparse) für einen einmaligen Admin-Vorgang ist nicht gerechtfertigt. Go's `encoding/csv` ist built-in und ausreichend.

### 2. Kein E-Mail-Versand beim Import

Einladungs-E-Mails werden beim CSV-Import nie automatisch versendet. Der neue Endpoint schreibt nur `invitation_tokens` mit `used_at = NULL`. Der bestehende `Invite`-Handler bleibt für Einzel-Einladungen vorhanden, wird aber aus dem Haupt-UI entfernt — er ist die Basis für den neuen „Einladung senden"-ActionMenu-Eintrag, der `POST /api/admin/invitations/{id}/send` aufruft.

### 3. `invitation_tokens.member_id` — nullable FK

```sql
ALTER TABLE invitation_tokens ADD COLUMN member_id INTEGER REFERENCES members(id) ON DELETE SET NULL;
```

Admin verknüpft eine Einladung manuell via ActionMenu → Modal mit Member-Suche. Beim Registrieren (`POST /auth/register`): wenn `invitation_tokens.member_id IS NOT NULL`, setzt der Handler `UPDATE members SET user_id = ? WHERE id = ?`.

**Risiko:** Mehrere Einladungen könnten auf dasselbe `member_id` zeigen. Mitigiert durch: vor dem Registrieren prüfen, ob `members.user_id` bereits gesetzt ist (Conflict zurückgeben).

### 4. `last_login_at` auf `users`

```sql
ALTER TABLE users ADD COLUMN last_login_at DATETIME;
```

Im Login-Handler nach erfolgreicher Authentifizierung:
```go
h.db.ExecContext(ctx, `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`, userID)
```

Keine eigene Tabelle, kein Login-History-Log — nur der letzte Zeitpunkt reicht.

### 5. UI — Dropdown-Button

Der bisherige „+ Einladung"-Button wird durch einen „CSV importieren"-Button ersetzt. Der Einzel-Einlade-Flow (bisheriges Modal) entfällt als primärer Einstieg; Einzel-E-Mails werden über das ActionMenu in der Einladungs-Tabelle versendet.

## Risks / Trade-offs

- **CSV-Encoding:** Mitgliederliste könnte in UTF-8 oder Latin-1 vorliegen. Go's `encoding/csv` erwartet UTF-8. Mitigation: Encoding-Fehler werden im Ergebnis angezeigt, der Admin muss die Datei ggf. konvertieren.
- **SMTP-Fehler beim Einzel-Versand:** `POST /api/admin/invitations/{id}/send` gibt 502 zurück wenn SMTP fehlschlägt — gleich wie der bisherige `Invite`-Handler.
- **Doppelte member_id-Verknüpfung:** Wenn zwei Einladungen auf dasselbe Mitglied zeigen und beide registrieren, gewinnt der Erste. Der Zweite erhält einen Conflict-Fehler beim Registrieren. Mitigation: im Register-Handler explizit prüfen.

## Migration Plan

1. Migration `00N_invitation_member_link.up.sql`: `ALTER TABLE invitation_tokens ADD COLUMN member_id INTEGER REFERENCES members(id) ON DELETE SET NULL`
2. Migration `00N+1_users_last_login.up.sql`: `ALTER TABLE users ADD COLUMN last_login_at DATETIME`
3. Deployment via `make deploy` (führt `migrate up` automatisch aus)
4. Rollback: `.down.sql` mit `ALTER TABLE ... DROP COLUMN` (SQLite ≥ 3.35)
