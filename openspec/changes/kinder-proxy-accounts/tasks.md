## 1. Datenbank-Migration

- [ ] 1.1 Migration `031_proxy_accounts.up.sql` anlegen: `users`-Tabelle via Rename → Create → Insert → Drop neu aufbauen mit `can_login INTEGER NOT NULL DEFAULT 1`, nullable `email`, partiellen Unique-Index `users_email_login_unique ON users(email) WHERE can_login = 1 AND email IS NOT NULL`
- [ ] 1.2 Migration `031_proxy_accounts.down.sql` anlegen: partiellen Index entfernen, `email` NOT NULL setzen, globalen UNIQUE-Constraint `users(email)` wiederherstellen, Proxy-Accounts (can_login = 0) löschen

## 2. Backend: Auth-Queries absichern

- [ ] 2.1 `internal/auth/handler.go` — Login-Query auf `WHERE email = ? AND can_login = 1` einschränken
- [ ] 2.2 `internal/auth/handler.go` — Passwort-Reset-Query auf `WHERE email = ? AND can_login = 1` einschränken (kein Reset-Link für Proxy-Accounts)
- [ ] 2.3 `internal/auth/handler.go` / `internal/members/handler.go` — E-Mail-Eindeutigkeitsprüfungen bei Registrierung, Invite und E-Mail-Änderung auf `can_login = 1`-Accounts beschränken

## 3. Backend: Proxy-Account-Verwaltung

- [ ] 3.1 `internal/members/handler.go` — Endpoint `POST /api/members/{id}/proxy-account` implementieren: `users`-Datensatz mit `can_login = 0` anlegen, `members.user_id` setzen, HTTP 409 wenn `user_id` bereits gesetzt
- [ ] 3.2 `cmd/teamwerk/main.go` — Route für `POST /api/members/{id}/proxy-account` unter der Vorstand-Gruppe registrieren
- [ ] 3.3 `internal/members/handler.go` oder `internal/auth/handler.go` — `GET /api/users`-Response um Feld `"proxy": bool` (`can_login == 0`) erweitern
- [ ] 3.4 `internal/auth/handler.go` — `PUT /api/users/{id}` erlaubt Admin, `can_login` auf 1 zu setzen und E-Mail einzutragen; Uniqueness-Prüfung gegen andere `can_login = 1`-Accounts; HTTP 409 bei Konflikt

## 4. Backend: Proxy-Kinder für Elternteil abrufbar machen

- [ ] 4.1 Neuer Endpoint `GET /api/family/proxy-accounts` (oder Erweiterung von `GET /api/profile/me`): gibt für das eingeloggte Elternteil alle via `family_links` verknüpften Mitglieder zurück, die einen Proxy-Account haben (`members.user_id IS NOT NULL` und `users.can_login = 0`), mit `{ user_id, member_id, name }`
- [ ] 4.2 Route in `cmd/teamwerk/main.go` unter der Authenticated-Gruppe registrieren

## 5. Backend: Duty-Claim für Familienmitglied

- [ ] 5.1 `internal/duties/handler.go` — Claim-Endpoint (`POST /api/duty-board/{slotId}/claim`) um optionales Body-Feld `user_id` erweitern; fehlt `user_id`, wird `claims.UserID` verwendet (Rückwärtskompatibilität)
- [ ] 5.2 Wenn `user_id != claims.UserID`: prüfen, dass die Ziel-`user_id` via `family_links` + `members.user_id` mit dem eingeloggten Elternteil verknüpft ist und `can_login = 0` hat; sonst HTTP 403
- [ ] 5.3 `duty_assignments`-Insert und `duty_accounts`-Upsert verwenden die Ziel-`user_id` (nicht notwendigerweise den eingeloggten User)

## 6. Frontend: Admin — Proxy-Account anlegen

- [ ] 6.1 `web/src/components/admin/MemberFamilieTab.tsx` — „Proxy-Account anlegen"-Button anzeigen, wenn `member.user_id` null ist; `POST /api/members/{id}/proxy-account` aufrufen; nach Erfolg Tab neu laden
- [ ] 6.2 Fehlerfall anzeigen (HTTP 409: „Mitglied hat bereits einen Account")

## 7. Frontend: Admin — Proxy-Accounts in Nutzerliste

- [ ] 7.1 `web/src/pages/AdminUsersPage.tsx` — Proxy-Accounts (`proxy: true`) mit einem Badge „Proxy" kenntlich machen, Login-bezogene Aktionen (Passwort-Reset, Einladung) für Proxy-Accounts ausblenden
- [ ] 7.2 „Aktivieren"-Button für Proxy-Accounts: öffnet Modal mit E-Mail-Eingabe; `PUT /api/users/{id}` mit `{ can_login: 1, email }` aufrufen; bei HTTP 409 Fehlermeldung anzeigen

## 8. Frontend: Dienstbörse — „Für wen?"-Dialog

- [ ] 8.1 `web/src/pages/DutyPage.tsx` — beim Mounten `GET /api/family/proxy-accounts` für Elternteile abrufen und im State halten
- [ ] 8.2 Claim-Button-Logik: wenn `proxyChildren.length > 0`, Dialog öffnen statt direkt zu claimen
- [ ] 8.3 Dialog implementieren: Liste mit eigenem Namen (default) + je ein Eintrag pro Proxy-Kind; Bestätigungsbutton ruft `POST /api/duty-board/{slotId}/claim` mit `{ user_id }` auf
- [ ] 8.4 Nach erfolgreichem Claim Dialog schließen, Duty-Board neu laden (SSE-Event abfangen oder explizit refetchen)
