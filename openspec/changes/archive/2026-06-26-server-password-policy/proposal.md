## Why

Die drei passwortsetzenden Handler — Register (`internal/auth/handler.go:528`), ResetPassword (`:644`) und ChangePassword (`:1190`) — prüfen serverseitig nur auf `!= ""`, bevor `bcrypt.GenerateFromPassword` aufgerufen wird. Es gibt keine Mindestlänge, keine Maximallänge und keine Komplexitätsregel; ein 1-Zeichen-Passwort wird akzeptiert. Das einzige Längenkriterium ist ein clientseitiges HTML-`minLength={8}` (`RegisterPage.tsx:104`, `ResetPasswordPage.tsx:38`), das durch Direktaufruf der API trivial umgangen wird (Sicherheitsaudit 2026-06-26, **B-6**). In Kombination mit fehlendem Login-Rate-Limiting (siehe `auth-rate-limiting`) sind schwache Passwörter schnell bruteforce-bar. ResetPassword aktiviert zudem Kind-Accounts beim ersten Passwortsetzen (`can_login=1`).

## What Changes

- **Serverseitige Passwortvalidierung** in Register, ResetPassword und ChangePassword: Mindestlänge ≥ 12 Zeichen; Eingaben > 72 Byte (bcrypt-Limit — sonst stille Trunkierung) werden mit klarem HTTP 400 abgelehnt.
- **Gemeinsame Validierungsfunktion**, damit alle drei Pfade dieselbe Regel teilen (ein Wahrheitsort).
- **Frontend spiegelt** die Regel (`minLength`/Hinweistext), die Durchsetzung liegt aber serverseitig.
- Optional (Folgeschritt, hier nur erwähnt): Abgleich gegen eine kleine Common-Password-Blocklist.

## Capabilities

### New Capabilities
- `password-policy`: Serverseitig erzwungene Passwort-Mindeststärke (Länge, bcrypt-72-Byte-Grenze) für alle passwortsetzenden Endpunkte.

### Modified Capabilities
<!-- keine -->

## Impact

- **Code:** gemeinsamer Validator in `internal/auth`, aufgerufen in Register-/ResetPassword-/ChangePassword-Handler; `internal/config` optional für die Mindestlänge.
- **Frontend:** `RegisterPage.tsx`, `ResetPasswordPage.tsx` und der Passwort-Ändern-Dialog spiegeln Mindestlänge + Fehlermeldung.
- **API-Verhalten:** zu kurze/zu lange Passwörter → 400 (vorher fälschlich akzeptiert). Bestehende Accounts mit schwachem Passwort bleiben gültig (kein Zwangs-Reset).
- **Tests:** pro Handler Happy-Path (gültiges Passwort) + Fehlerfall (zu kurz → 400, > 72 Byte → 400).
- **Daten/Migration:** keine.
