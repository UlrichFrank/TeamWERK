## Why

Weder die Go-Middleware-Kette (`internal/app/router.go:79-86`: `InFlightMiddleware → Recoverer → CleanPath → CORS`) noch der nginx-Reverse-Proxy (`deploy/nginx-intern.conf`) drosseln Anfragen. Die unauthentifizierten Auth-Routen `POST /api/auth/login`, `/refresh`, `/forgot-password`, `/reset-password` (`internal/auth/handler.go`) sind damit mit **unbegrenzten Versuchen** erreichbar; es gibt keinen Fehlversuchszähler, keine Sperre, kein CAPTCHA (Sicherheitsaudit 2026-06-26, **B-2 Medium**).

Login ist durch bcrypt-Cost 10 + konstant-zeitigen Dummy-Hash gegen Enumeration gehärtet und der Reset-Token ist 256-bit-Zufall — der reale Schaden ist daher **nicht** Passwort-Bruteforce, sondern (a) Mail-Bombing/SMTP-Erschöpfung über ungedrosseltes `forgot-password` (ein Mailversand pro Request) und (b) CPU-DoS auf dem 1-GB-VPS, da jeder Login/Reset eine teure bcrypt-Operation auslöst. Ein account-basierter Lockout schließt zusätzlich nachhaltiges Online-Bruteforcing schwacher Passwörter (verstärkt durch fehlende Passwortpolicy, siehe `server-password-policy`).

## What Changes

- **IP-basiertes Rate-Limiting** vor die vier öffentlichen Auth-Routen (`go-chi/httprate` als Chi-Middleware auf der Auth-Gruppe, z.B. ~5–10 Versuche/min/IP; bei Überschreitung HTTP 429 mit `Retry-After`).
- **Account-basierter Lockout:** `failed_login_count` + `locked_until` auf `users`; nach N aufeinanderfolgenden Fehlversuchen exponentielles Backoff. Erfolgreicher Login setzt den Zähler zurück.
- **Drosselung von `forgot-password`** zusätzlich pro Ziel-E-Mail (verhindert Mailbomb), ohne die Existenz der Adresse preiszugeben (Antwortverhalten bleibt generisch).
- **Konfigurierbarkeit + Test-Override:** Limits über Config (`.env`), in Tests deaktivier-/herabsetzbar, damit bestehende Auth-Tests nicht flaky werden.

## Capabilities

### New Capabilities
- `auth-rate-limiting`: Drosselung und Account-Lockout für die unauthentifizierten Auth-Endpunkte (Schutz gegen Bruteforce, Mail-Bombing und bcrypt-CPU-DoS).

### Modified Capabilities
<!-- keine -->

## Impact

- **Code:** `internal/app/router.go` (Middleware auf der Public-Auth-Gruppe), `internal/auth/handler.go` (Login/Forgot/Reset: Zähler/Lockout), `internal/config` (Limit-Parameter).
- **Dependency:** neue Go-Abhängigkeit `github.com/go-chi/httprate` (oder äquivalent `golang.org/x/time/rate`-Wrapper, Entscheidung im Design).
- **Daten/Migration:** neue Migration `010_user_login_throttle.up.sql/.down.sql` — Spalten `failed_login_count INTEGER NOT NULL DEFAULT 0`, `locked_until TEXT` auf `users` (nächste freie Nummer nach `009`).
- **Betrieb:** optional zusätzlich nginx `limit_req_zone` als zweite Schicht (im Design beschrieben, Deploy-Datei).
- **Tests:** 429 bei Überschreitung, Lockout nach N Fehlversuchen, Reset bei Erfolg; bestehende Auth-Happy-Path-Tests laufen mit hochgesetztem/abgeschaltetem Limit weiter.
