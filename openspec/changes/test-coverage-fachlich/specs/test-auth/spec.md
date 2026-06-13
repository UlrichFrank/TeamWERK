## ADDED Requirements

### Requirement: Login-Verhalten
Das System SHALL Login-Anfragen anhand von E-Mail (case-insensitive) und bcrypt-Passwort prüfen. Nur Nutzer mit `can_login = 1` dürfen sich einloggen.

#### Scenario: Valider Login
- **WHEN** POST /api/auth/login mit korrekter E-Mail und Passwort
- **THEN** HTTP 200, `access_token` im Body, Cookie `refresh_token` gesetzt (HttpOnly), `refresh_tokens`-Eintrag in DB, `users.last_login_at` aktualisiert

#### Scenario: Falsches Passwort
- **WHEN** POST /api/auth/login mit falschem Passwort
- **THEN** HTTP 401, kein neuer `refresh_tokens`-Eintrag

#### Scenario: Unbekannte E-Mail
- **WHEN** POST /api/auth/login mit nicht existierender E-Mail
- **THEN** HTTP 401

#### Scenario: Proxy-Account Login gesperrt
- **WHEN** POST /api/auth/login für einen User mit `can_login = 0`
- **THEN** HTTP 401 (Query filtert `can_login = 1`)

### Requirement: Token-Rotation beim Refresh
Das System SHALL beim Refresh den alten Refresh-Token löschen und einen neuen ausstellen (Token-Rotation).

#### Scenario: Gültiger Refresh
- **WHEN** POST /api/auth/refresh mit gültigem Cookie
- **THEN** HTTP 200, neuer `access_token`, neuer Cookie, alter Token-Hash aus DB gelöscht

#### Scenario: Ungültiger Refresh
- **WHEN** POST /api/auth/refresh mit gefälschtem Cookie-Wert
- **THEN** HTTP 401

### Requirement: Logout löscht Session
Das System SHALL beim Logout den Refresh-Token aus der DB löschen und den Cookie mit MaxAge=-1 löschen.

#### Scenario: Logout mit gültigem Cookie
- **WHEN** POST /api/auth/logout mit Cookie
- **THEN** HTTP 204, Token in DB gelöscht, Cookie hat MaxAge = -1

### Requirement: Registrierung via Einladungstoken
Das System SHALL die Registrierung nur mit einem gültigen, ungenutzten, nicht abgelaufenen Einladungstoken erlauben.

#### Scenario: Gültiger Token
- **WHEN** POST /api/auth/register mit gültigem Token, Vorname, Passwort
- **THEN** HTTP 201, User angelegt, `invitation_tokens.used_at` gesetzt

#### Scenario: Abgelaufener Token
- **WHEN** POST /api/auth/register mit Token dessen `expires_at` in der Vergangenheit liegt
- **THEN** HTTP 400

#### Scenario: Bereits benutzter Token
- **WHEN** POST /api/auth/register mit Token dessen `used_at IS NOT NULL`
- **THEN** HTTP 400

### Requirement: Passwort-Reset Anti-Enumeration
Das System SHALL auf POST /api/auth/forgot-password immer HTTP 204 antworten, unabhängig davon ob die E-Mail existiert.

#### Scenario: Bekannte E-Mail
- **WHEN** POST /api/auth/forgot-password mit existierender E-Mail
- **THEN** HTTP 204, `password_reset_tokens`-Eintrag angelegt

#### Scenario: Unbekannte E-Mail
- **WHEN** POST /api/auth/forgot-password mit unbekannter E-Mail
- **THEN** HTTP 204, kein Token angelegt

### Requirement: Passwort-Reset invalidiert alle Sessions
Das System SHALL beim Zurücksetzen des Passworts alle Refresh-Tokens des Nutzers löschen.

#### Scenario: Gültiger Reset-Token
- **WHEN** POST /api/auth/reset-password mit gültigem Token und neuem Passwort
- **THEN** HTTP 204, Passwort in DB geändert, `password_reset_tokens.used_at` gesetzt, alle `refresh_tokens` des Users gelöscht

#### Scenario: Abgelaufener Reset-Token
- **WHEN** POST /api/auth/reset-password mit Token dessen `expires_at` in Vergangenheit
- **THEN** HTTP 400

### Requirement: Rollenvergabe nur durch Admin
Das System SHALL die Rolle `"admin"` nur durch einen Admin vergeben lassen. Nur `"admin"` und `"standard"` sind gültige Rollenwerte.

#### Scenario: Admin vergibt Admin-Rolle
- **WHEN** Admin PUT /api/admin/users/{id}/role mit `{ role: "admin" }`
- **THEN** HTTP 204, Rolle aktualisiert

#### Scenario: Nicht-Admin versucht Admin-Rolle
- **WHEN** Standard-User PUT /api/admin/users/{id}/role mit `{ role: "admin" }`
- **THEN** HTTP 403

#### Scenario: Ungültige Rolle
- **WHEN** PUT /api/admin/users/{id}/role mit `{ role: "trainer" }`
- **THEN** HTTP 400

### Requirement: Nutzer-Löschung mit Cascade
Das System SHALL beim Löschen eines Nutzers alle abhängigen Daten in einer Transaktion entfernen. Ein Admin darf sich nicht selbst löschen.

#### Scenario: Selbstlöschung verboten
- **WHEN** Admin DELETE /api/admin/users/{eigene_id}
- **THEN** HTTP 400

#### Scenario: Cascade-Löschung
- **WHEN** Admin DELETE /api/admin/users/{andere_id}
- **THEN** HTTP 204; `refresh_tokens`, `duty_assignments`, `duty_accounts`, `family_links` des Nutzers sind entfernt
