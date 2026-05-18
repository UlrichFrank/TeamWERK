## 1. Backend: Einladungen auflisten und löschen

- [x] 1.1 Handler `ListInvitations` in `internal/auth/handler.go` implementieren: Query `SELECT id, email, role, COALESCE(t.name,'') FROM invitation_tokens LEFT JOIN teams t ON t.id=team_id WHERE used_at IS NULL AND expires_at > CURRENT_TIMESTAMP ORDER BY expires_at`, Antwort als JSON-Array mit id, email, role, team_name, expires_at
- [x] 1.2 Handler `DeleteInvitation` in `internal/auth/handler.go` implementieren: 404-Prüfung per rowsAffected, dann `DELETE FROM invitation_tokens WHERE id=?`, HTTP 204
- [x] 1.3 Routen `GET /api/admin/invitations` und `DELETE /api/admin/invitations/{id}` in `cmd/teamwerk/main.go` unter Admin-only-Gruppe registrieren

## 2. Backend: Beitrittsanfrage löschen

- [x] 2.1 Handler `DeleteMembershipRequest` in `internal/auth/handler.go` implementieren: 404-Prüfung per rowsAffected, dann `DELETE FROM membership_requests WHERE id=?`, HTTP 204
- [x] 2.2 Route `DELETE /api/admin/membership-requests/{id}` in `cmd/teamwerk/main.go` unter Admin+Trainer-Gruppe registrieren

## 3. Frontend: Unified Table

- [x] 3.1 In `AdminUsersPage.tsx` Typen für `Invitation` (id, email, role, team_name, expires_at) und `MembershipRequest` (id, name, email, team_name?) anlegen; State für `invitations` und `requests` hinzufügen
- [x] 3.2 Alle drei Datenquellen parallel beim Mount laden: `GET /admin/users`, `GET /admin/invitations`, `GET /admin/membership-requests`
- [x] 3.3 Tabelle zu einem gemischten Array zusammenführen (Typ-Diskriminator `kind: 'user' | 'invitation' | 'request'`); Anfragen und Einladungen zuerst sortieren, dann Nutzer alphabetisch
- [x] 3.4 Status-Badge je Typ rendern: `Anfrage` (brand-yellow/black), `Einladung` (gray-200/gray-700), Rollen-Badges für Nutzer wie bisher
- [x] 3.5 Aktions-Spalte je Typ: User → Löschen; Invitation → Löschen (mit confirm + `DELETE /admin/invitations/{id}`); Request → Genehmigen + Ablehnen + Löschen
- [x] 3.6 Optimistisches Update nach jeder Aktion: Eintrag sofort aus dem lokalen Array entfernen / Liste neu laden
