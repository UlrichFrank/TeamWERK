## 1. Datenbank-Migrationen

- [x] 1.1 Migration `019_invitation_member_link.up.sql` anlegen: `ALTER TABLE invitation_tokens ADD COLUMN member_id INTEGER REFERENCES members(id) ON DELETE SET NULL`
- [x] 1.2 Migration `019_invitation_member_link.down.sql` anlegen: `ALTER TABLE invitation_tokens DROP COLUMN member_id`
- [x] 1.3 Migration `020_users_last_login.up.sql` anlegen: `ALTER TABLE users ADD COLUMN last_login_at DATETIME`
- [x] 1.4 Migration `020_users_last_login.down.sql` anlegen: `ALTER TABLE users DROP COLUMN last_login_at`
- [x] 1.5 `make migrate-up` lokal ausführen und prüfen, dass beide Migrationen angewendet wurden

## 2. Backend — CSV-Import Endpoint

- [x] 2.1 `POST /api/admin/invitations/import-csv` in `internal/auth/handler.go` implementieren: CSV als `multipart/form-data` entgegennehmen, `encoding/csv` zum Lesen der Spalten `Email` und `Email 2` verwenden
- [x] 2.2 E-Mail-Deduplizierung im Handler: unique Emails aus beiden Spalten sammeln, gegen `users.email` und `invitation_tokens.email` prüfen (Case-insensitiv), neue Tokens mit Rolle `standard` und Ablaufzeit +48h anlegen
- [x] 2.3 Response `{ "created": N, "skipped": M }` zurückgeben (200 OK); 400 bei fehlender CSV oder fehlender `Email`-Spalte
- [x] 2.4 Route in `cmd/teamwerk/main.go` registrieren (Admin-only Middleware)

## 3. Backend — Einladung senden Endpoint

- [x] 3.1 `POST /api/admin/invitations/{id}/send` in `internal/auth/handler.go` implementieren: Token aus DB lesen, Einladungs-E-Mail versenden (gleicher Body wie bisheriger `Invite`-Handler), 204 bei Erfolg, 502 bei SMTP-Fehler, 404 wenn nicht gefunden
- [x] 3.2 Route in `cmd/teamwerk/main.go` registrieren (Admin-only Middleware)

## 4. Backend — Einladung ↔ Mitglied verknüpfen

- [x] 4.1 `PUT /api/admin/invitations/{id}/member` implementieren: Body `{ "member_id": N }` setzt `invitation_tokens.member_id`; prüft vorher ob `members.user_id` bereits gesetzt ist (409 Conflict); `member_id: null` hebt Verknüpfung auf
- [x] 4.2 Route in `cmd/teamwerk/main.go` registrieren (Admin-only Middleware)
- [x] 4.3 `GET /api/admin/invitations`-Response um `member_id` und `member_name` (JOIN auf `members`) erweitern

## 5. Backend — Register-Handler: Auto-Link Mitglied

- [x] 5.1 In `Register`-Handler (`internal/auth/handler.go`): nach `INSERT INTO users` prüfen ob der verwendete Token `member_id IS NOT NULL` hat; wenn ja und `members.user_id IS NULL`, dann `UPDATE members SET user_id = ? WHERE id = ?` ausführen

## 6. Backend — last_login_at

- [x] 6.1 Im Login-Handler (`internal/auth/handler.go`) nach erfolgreicher Passwort-Prüfung: `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`
- [x] 6.2 `last_login_at` im `GET /api/admin/users`-Response mitliefern (Handler und SQL-Query anpassen)

## 7. Frontend — AdminUsersPage: CSV-Import Button & Modal

- [x] 7.1 „+ Einladung"-Button durch „CSV importieren"-Button ersetzen; bisheriges Invite-Modal entfernen
- [x] 7.2 CSV-Import-Modal implementieren: Datei-Upload-Feld (accept=".csv"), Upload-Button, Ladezustand, Ergebnisanzeige „X Einladungen angelegt, Y übersprungen", Fehleranzeige
- [x] 7.3 `api.post('/admin/invitations/import-csv', formData, { headers: { 'Content-Type': 'multipart/form-data' } })` aufrufen; nach Erfolg Einladungsliste neu laden

## 8. Frontend — AdminUsersPage: ActionMenu Erweiterungen

- [x] 8.1 ActionMenu jeder Einladungs-Zeile um „Einladung senden" ergänzen (`POST /api/admin/invitations/{id}/send`); Erfolg/Fehler als Inline-Feedback anzeigen
- [x] 8.2 ActionMenu jeder Einladungs-Zeile um „Mit Mitglied verknüpfen" ergänzen: öffnet Modal mit durchsuchbarer Mitgliederliste (`GET /api/members?search=...`), nach Auswahl `PUT /api/admin/invitations/{id}/member` aufrufen
- [x] 8.3 Verknüpftes Mitglied in der Einladungs-Zeile anzeigen (Name wenn vorhanden); ActionMenu um „Verknüpfung aufheben" ergänzen wenn `member_id` gesetzt

## 9. Frontend — AdminUsersPage: last_login_at Spalte

- [x] 9.1 `User`-Interface um `last_login_at?: string | null` erweitern
- [x] 9.2 Neue Spalte „Letzter Login" in der Nutzertabelle: relative Zeitanzeige (z.B. „vor 3 Tagen") oder „–" wenn `null`; Hilfsfunktion für relative Zeitformatierung inline implementieren
