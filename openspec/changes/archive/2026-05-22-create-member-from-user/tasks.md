## 1. Backend: Endpoint

- [x] 1.1 `POST /api/admin/users/{id}/create-member` in `internal/members/handler.go` implementieren
- [x] 1.2 Name des Users laden, per Leerzeichen in `first_name` / `last_name` splitten
- [x] 1.3 Prüfen ob User bereits ein Mitglied hat — bei Konflikt HTTP 409 zurückgeben
- [x] 1.4 Mitglied mit `status='aktiv'` und `user_id` insertieren (kein RETURNING, stattdessen LastInsertId)
- [x] 1.5 Route in `cmd/teamwerk/main.go` unter Admin-only-Gruppe eintragen

## 2. Backend: Nutzerliste erweitern

- [x] 2.1 `GET /api/admin/users`-Response um `member_id` (nullable) ergänzen — JOIN auf `members.user_id`

## 3. Frontend: AdminUsersPage

- [x] 3.1 `member_id`-Feld im User-Interface-Typ ergänzen
- [x] 3.2 „Mitglied anlegen"-Button in der Nutzerzeile rendern — nur wenn `member_id == null`
- [x] 3.3 Button-Klick: `POST /api/admin/users/{id}/create-member` aufrufen
- [x] 3.4 Nach Erfolg: `member_id` im lokalen State setzen, Button verschwindet (kein Reload)
- [x] 3.5 Fehlerfall: Fehlermeldung inline in der Zeile anzeigen
