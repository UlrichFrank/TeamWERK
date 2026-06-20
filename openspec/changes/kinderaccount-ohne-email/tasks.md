## 1. Datenbank-Migration

- [x] 1.1 `internal/db/migrations/002_kinderaccount_login.up.sql`: `ALTER TABLE users ADD COLUMN login_name TEXT;`
- [x] 1.2 In derselben up-Migration: `CREATE UNIQUE INDEX users_login_name_unique ON users(LOWER(login_name)) WHERE can_login=1 AND login_name IS NOT NULL;`
- [x] 1.3 In derselben up-Migration: `ALTER TABLE membership_requests ADD COLUMN is_child INTEGER NOT NULL DEFAULT 0;` und `ADD COLUMN parent_email TEXT;`
- [x] 1.4 `002_kinderaccount_login.down.sql`: Index droppen und Spalten zurückrollen (SQLite `DROP COLUMN` bzw. Tabellen-Rebuild)
- [x] 1.5 `make migrate-up` lokal ausführen und Schema prüfen

## 2. Spielername-Normalisierung & Generierung (Backend)

- [x] 2.1 Helper `normalizeLoginName(first, last string) string` in `internal/auth/`: Trim, Umlaut/ß-Transliteration via `strings.NewReplacer`, Leerzeichen→Bindestrich je Namensteil, Reduktion auf `[A-Za-z0-9-]`, Format `Vorname.Nachname`
- [x] 2.2 Helper `generateUniqueLoginName(ctx, tx, base string) (string, error)`: prüft Eindeutigkeit case-insensitiv gegen ALLE `login_name` (unabhängig von `can_login`), hängt bei Kollision Suffix `2`,`3`,… an den Nachnamen-Teil, harte Obergrenze (z. B. 1000) → Fehler
- [x] 2.3 Unit-Tests für 2.1 (Umlaute, Doppelname, Sonderzeichen, leeres Ergebnis) und 2.2 (Kollision → Suffix, mehrere inaktive Konten)

## 3. Login um Spielername erweitern (Backend)

- [x] 3.1 `Login` in `internal/auth/handler.go`: Query auf `WHERE (LOWER(email)=? OR LOWER(login_name)=?) AND can_login=1` umstellen (gleicher lowercased Eingabewert für beide Parameter)
- [x] 3.2 Sicherstellen, dass das Timing-Safe-Dummy-Hash-Verhalten erhalten bleibt (kein Enumeration-Leak über Spielername)
- [x] 3.3 Tests: Login per `login_name` (Erfolg, case-insensitiv), per E-Mail (Regression), `can_login=0` → 401, falsches Passwort → 401

## 4. Beitrittsantrag-Kindervariante (Backend)

- [x] 4.1 `RequestMembership` erweitern: Felder `is_child`, `parent_email` annehmen; bei `is_child=1` Pflicht-Validierung von Vor-/Nachname + gültiger `parent_email`, sonst HTTP 400; Insert mit den neuen Spalten
- [x] 4.2 `ListMembershipRequests` um `is_child`/`parent_email` in der Ausgabe ergänzen (Vorstand sieht Eltern-Adresse vor dem Approve)
- [x] 4.3 Tests: Kinderantrag-Anlage (Erfolg), fehlende/ungültige `parent_email` → 400, Standard-Antrag bleibt `is_child=0`

## 5. Approve-Flow für Kinderanträge (Backend)

- [x] 5.1 `ApproveMembershipRequest` verzweigt bei `is_child=1`: Transaktion mit (a) `generateUniqueLoginName`, (b) `INSERT INTO users (login_name, email, password, role, can_login) VALUES (?, NULL, '', 'standard', 0)` + `LastInsertId()`, (c) `INSERT INTO members (first_name, last_name, user_id, …)`, (d) Passwort-Setz-Token (48 h) anlegen
- [x] 5.2 Nach Commit: Mail an `parent_email` mit Spielername + `/reset-password?token=…` über `h.mailer.Send`; Mailfehler protokollieren, keine committeten Daten zurückrollen
- [x] 5.3 `h.hub.Broadcast("members")` nach erfolgreichem Approve (Member wird angelegt → bestehende AdminUsersPage hört auf `members`)
- [x] 5.4 Standard-Approve (E-Mail-Antrag) bleibt unverändert (invitation_token-Pfad)
- [x] 5.5 Tests: Kinder-Approve legt User+Member an (`can_login=0`, `user_id` verknüpft, Status `approved`), Kollision → Suffix, kein `family_link` angelegt, Standard-Approve-Regression

## 6. Passwort setzen / Account-Aktivierung (Backend)

- [x] 6.1 Set-Password-Route (bestehenden Reset-Flow wiederverwenden/erweitern): bei gültigem Token `password=<bcrypt>` setzen UND `can_login=1`, Token invalidieren
- [x] 6.2 Tests: gültiger Token → 204 + `can_login=1`, abgelaufener/verbrauchter Token → 400 (Konto bleibt `can_login=0`)

## 7. Frontend

- [x] 7.1 Beitrittsantrag-Formular (`web/src/pages/…`): Toggle „Kinderaccount anlegen"; im Kinder-Modus Felder Kind-Vorname/-Nachname + Eltern-E-Mail statt eigener E-Mail; `is_child`/`parent_email` an die API senden
- [x] 7.2 Login-Seite: Feld-Label auf „E-Mail oder Spielername" anpassen (Input `type=text`, sonst blockt Browser-Validierung den Spielernamen)
- [x] 7.3 Vorstand-Ansicht der Anträge: Kindkennzeichnung (Badge) + Eltern-E-Mail anzeigen
- [x] 7.4 brand-Tokens & lucide-Icons (`Baby`) einhalten; `pnpm -C web build` + lint grün

## 8. Verifikation & Abschluss

- [x] 8.1 `/verify-change` ausführen (Build/Test/Lint, Route→Tests, Mutation→Broadcast, brand-Tokens, lucide-Icons, Migrationsnummer)
- [x] 8.2 `openspec validate kinderaccount-ohne-email --strict`
- [ ] 8.3 Manueller Durchlauf: Kinderantrag stellen → akzeptieren → Eltern-Mail → Passwort setzen → Kind-Login mit `Vorname.Nachname` (offen — braucht laufende App + SMTP; der End-to-End-Pfad ist durch Integrationstests in `loginname_handler_test.go` abgedeckt)
