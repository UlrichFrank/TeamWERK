## 1. Datenbank-Migration

- [ ] 1.1 Migration `008_email_change_tokens.up.sql` anlegen: Tabelle `email_change_tokens` (user_id FK, token TEXT UNIQUE, new_email TEXT, expires_at DATETIME, used_at DATETIME)
- [ ] 1.2 Migration `008_email_change_tokens.down.sql` anlegen (DROP TABLE)
- [ ] 1.3 Migration in `internal/db/migrations/` kopieren und `make migrate-up` lokal ausführen

## 2. Backend — Name ändern

- [ ] 2.1 Handler `UpdateAccount` in `internal/auth/handler.go`: `PUT /api/profile/account` — liest `name` aus Body, validiert nicht-leer, UPDATE users SET name=? WHERE id=?
- [ ] 2.2 Route `PUT /api/profile/account` unter authentifizierter Gruppe in `main.go` registrieren

## 3. Backend — Passwort ändern

- [ ] 3.1 Handler `ChangePassword` in `internal/auth/handler.go`: `POST /api/profile/password` — prüft `current_password` via bcrypt, setzt neues bcrypt-Hash, löscht alle refresh_tokens des Nutzers
- [ ] 3.2 Route `POST /api/profile/password` unter authentifizierter Gruppe in `main.go` registrieren

## 4. Backend — E-Mail ändern

- [ ] 4.1 Handler `RequestEmailChange` in `internal/auth/handler.go`: `POST /api/profile/email` — prüft Passwort, prüft ob new_email bereits vergeben (HTTP 409), löscht vorherigen Token des Nutzers, generiert neuen Token, speichert in `email_change_tokens`, sendet Bestätigungs-Mail
- [ ] 4.2 Handler `ConfirmEmailChange` in `internal/auth/handler.go`: `GET /api/profile/email/confirm` — prüft Token (exists, not used, not expired), UPDATE users SET email=?, markiert Token als used, löscht refresh_tokens, HTTP 302 → /login
- [ ] 4.3 Fehler-Redirect: abgelaufener/ungültiger Token → HTTP 302 → `/login?error=invalid_token`
- [ ] 4.4 Route `POST /api/profile/email` unter authentifizierter Gruppe registrieren
- [ ] 4.5 Route `GET /api/profile/email/confirm` als **public** Route registrieren (kein Auth-Token nötig beim Klicken des Links)

## 5. Router & Build

- [ ] 5.1 Alle neuen Routen in `cmd/teamwerk/main.go` eintragen
- [ ] 5.2 `go build ./...` — kein Compilerfehler

## 6. Frontend — Profilseite erweitern

- [ ] 6.1 Sektion „Konto" in `ProfilePage.tsx`: Name-Eingabefeld mit aktuellem Wert vorbelegt + Speichern-Button → `PUT /api/profile/account`
- [ ] 6.2 Erfolgs-Feedback nach Name-Speicherung (kurze Meldung, analog Fahrzeug-Sektion)
- [ ] 6.3 Sektion „Passwort ändern": drei Felder (aktuelles PW, neues PW, Wiederholung), client-seitige Übereinstimmungsprüfung
- [ ] 6.4 Nach erfolgreichem Passwort-Change: Hinweis „Du wirst ausgeloggt…" + `logout()` aus AuthContext nach kurzem Delay
- [ ] 6.5 Sektion „E-Mail ändern": zwei Felder (neue E-Mail, aktuelles PW) + Absenden → `POST /api/profile/email`
- [ ] 6.6 Nach erfolgreichem E-Mail-Request: Formular ausblenden, Hinweis „Bestätigungs-Mail gesendet" anzeigen
- [ ] 6.7 Fehlermeldungen: 403 → „Passwort nicht korrekt", 409 → „E-Mail bereits vergeben"

## 7. Frontend — E-Mail-Bestätigung

- [ ] 7.1 Route `/profil/email/bestaetigen` in `App.tsx` anlegen (oder direkt Backend-Redirect nutzen — Backend redirected zu `/login`, kein eigener Frontend-Screen nötig)
- [ ] 7.2 Login-Seite: `?error=invalid_token` Query-Param erkennen und Fehlermeldung anzeigen
