## Context

Die Seite `AdminUsersPage.tsx` enthält derzeit nur ein Einladungsformular. Das Backend hat bereits `GET /api/admin/users` (gibt id, name, email, role zurück), aber kein Delete-Endpunkt. Das Rollen-Modell und die Foreign-Key-Constraints sind vollständig vorhanden.

## Goals / Non-Goals

**Goals:**
- Nutzertabelle in bestehender AdminUsersPage anzeigen (name, email, rolle, team)
- `DELETE /api/admin/users/{id}` im Backend implementieren
- Nutzer-Löschen aus der UI ermöglichen (mit Bestätigungsdialog)
- `GET /api/admin/users` um Team-Name erweitern

**Non-Goals:**
- Rollen oder Teamzugehörigkeit im Admin-Bereich editieren
- Passwort-Reset durch Admin
- Soft-Delete / Archivierung (kein Bedarf)
- Pagination (Vereinsgröße rechtfertigt keine)

## Decisions

**1. Kaskadierendes Löschen via FK-Constraints vs. manuelles Cleanup**

SQLite hat `PRAGMA foreign_keys=ON` aktiv. Die bestehenden FK-Definitionen in den Tabellen `refresh_tokens`, `invitation_tokens`, `password_reset_tokens` sind mit `ON DELETE CASCADE` angelegt (zu prüfen). `family_links`, `duty_assignments`, `duty_accounts` müssen vor dem User-Delete geprüft und ggf. manuell in derselben Transaktion gelöscht werden.

→ **Entscheidung:** Explizites DELETE in einer Transaktion (refresh_tokens, family_links, duty_assignments, duty_accounts → dann users). Kein Verlass auf implizites Cascade, da Schema-Details variieren können.

**2. Self-Delete-Schutz**

Ein Admin darf sich nicht selbst löschen — sonst könnte der letzte Admin das System sperren.

→ **Entscheidung:** Backend prüft `claims.UserID != targetID`. Wenn gleich, 400 Bad Request.

**3. ListUsers um Team-Name erweitern**

Aktuell liefert `ListUsers` nur `id, name, email, role`. Team-Name via LEFT JOIN auf `teams` hinzufügen.

→ `SELECT u.id, u.name, u.email, u.role, COALESCE(t.name, '') AS team_name FROM users u LEFT JOIN teams t ON t.id = u.team_id ORDER BY u.name`

**4. Frontend: Tabelle + Einladungsformular nebeneinander**

Bestehende Einladungssektion bleibt erhalten. Darunter kommt die Nutzertabelle. Kein Tab-Layout nötig.

→ Tabelle mit Spalten: Name, E-Mail, Rolle, Team, Aktionen (Löschen-Button). Bestätigungsdialog via `window.confirm`.

## Risks / Trade-offs

- **Cascade-Vollständigkeit** → Mitigation: Transaktion und explizite Deletes in definierter Reihenfolge; bei Fehler rollback
- **Kein Undo** → Mitigation: Bestätigungsdialog im Frontend; kein Soft-Delete geplant (Anforderung klar)
- **Letzter Admin** → Mitigation: Self-Delete-Schutz reicht; Multi-Admin-Schutz ist out of scope

## Migration Plan

1. Backend: `DELETE /api/admin/users/{id}` Handler in `internal/auth/handler.go` + Route in `main.go`
2. Backend: `ListUsers` Query erweitern (kein DB-Migration nötig)
3. Frontend: `AdminUsersPage.tsx` um Nutzertabelle + Delete-Aktion erweitern
4. Kein Datenbankschema-Change — keine Migration-Datei erforderlich
