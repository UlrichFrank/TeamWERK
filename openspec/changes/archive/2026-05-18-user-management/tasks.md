## 1. Backend: ListUsers erweitern

- [x] 1.1 Query in `ListUsers` (`internal/auth/handler.go`) um LEFT JOIN auf `teams` ergänzen, sodass `team_name` im JSON-Response enthalten ist
- [x] 1.2 Response-Struct in `ListUsers` um `TeamName string` (json:"team_name") erweitern

## 2. Backend: DeleteUser implementieren

- [x] 2.1 Handler `DeleteUser` in `internal/auth/handler.go` implementieren: Self-Delete-Prüfung (HTTP 400), Nutzer-Existenzprüfung (HTTP 404), dann kaskadierende Deletes in einer Transaktion (refresh_tokens → family_links → duty_assignments → duty_accounts → users), HTTP 204 bei Erfolg
- [x] 2.2 Route `DELETE /api/admin/users/{id}` in `cmd/teamwerk/main.go` unter Admin-only-Gruppe registrieren

## 3. Frontend: Nutzertabelle

- [x] 3.1 In `AdminUsersPage.tsx` State für Nutzerliste (`users`) und Lade-State hinzufügen; Nutzerliste via `GET /api/admin/users` beim Mount laden
- [x] 3.2 Tabelle mit Spalten Name, E-Mail, Rolle, Team und Aktionen unterhalb des Einladungsformulars rendern; Rolle als Badge darstellen (Farbe je Rolle)
- [x] 3.3 Löschen-Button pro Tabellenzeile implementieren: `window.confirm`-Dialog, dann `DELETE /api/admin/users/{id}`, danach Nutzer aus lokaler Liste entfernen
- [x] 3.4 Self-Delete verhindern: Löschen-Button für den eigenen Eintrag deaktivieren (Vergleich mit `user.email` aus AuthContext oder eigener ID — ggf. ID in JWT-Claims prüfen)
