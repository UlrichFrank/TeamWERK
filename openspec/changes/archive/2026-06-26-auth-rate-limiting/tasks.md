## 1. Konfiguration & Abhängigkeit

- [x] 1.1 `go-chi/httprate` als Abhängigkeit hinzufügen (`go get`, in `go.mod`)
- [x] 1.2 Config-Parameter in `internal/config` ergänzen: Auth-Limit (Versuche/Fenster), Lockout-Schwelle, Lockout-Dauer; sinnvolle Defaults; in Tests deaktivierbar

## 2. Migration

- [x] 2.1 `internal/db/migrations/010_user_login_throttle.up.sql`: `ALTER TABLE users ADD COLUMN failed_login_count INTEGER NOT NULL DEFAULT 0; ADD COLUMN locked_until TEXT;`
- [x] 2.2 Passende `010_user_login_throttle.down.sql`

## 3. IP-Rate-Limiting

- [x] 3.1 `httprate`-Middleware nur auf die Public-Auth-Routengruppe in `internal/app/router.go` setzen (Schlüssel = Client-IP via `RealIP`)
- [x] 3.2 Sicherstellen, dass 429 VOR bcrypt/Mailversand greift; `Retry-After` setzen
- [x] 3.3 `RealIP`-Quelle korrekt an nginx `X-Forwarded-For` koppeln

## 4. Account-Lockout

- [x] 4.1 Login-Handler (`internal/auth/handler.go`): bei gesperrtem Konto (`locked_until` in der Zukunft) ohne bcrypt mit Drosselungsantwort reagieren
- [x] 4.2 Fehlversuch erhöht `failed_login_count`; bei Schwellenüberschreitung `locked_until` (exponentielles Backoff) setzen
- [x] 4.3 Erfolgreicher Login setzt `failed_login_count=0`, `locked_until=NULL`
- [x] 4.4 Generisches Antwortverhalten beibehalten (keine Enumeration: gesperrtes existierendes Konto ununterscheidbar von gedrosselter nicht-existenter E-Mail)

## 5. forgot-password-Drosselung

- [x] 5.1 Zusätzliche Drosselung pro Ziel-E-Mail für `forgot-password`, ohne Existenz preiszugeben

## 6. nginx (optionale zweite Schicht)

- [x] 6.1 `limit_req_zone` + `limit_req` für `location ~ ^/api/auth/(login|refresh|forgot-password|reset-password)` in `deploy/nginx-intern.conf`

## 7. Tests & Verifikation

- [x] 7.1 429 nach Limitüberschreitung; innerhalb Limit kein 429
- [x] 7.2 Lockout nach N Fehlversuchen; Reset bei Erfolg; gesperrtes Konto antwortet ohne bcrypt
- [x] 7.3 `forgot-password`: gedrosselte Anfrage versendet keine Mail
- [x] 7.4 Bestehende Auth-Happy-Path-Tests laufen mit hochgesetztem/abgeschaltetem Limit grün
- [x] 7.5 `/verify-change` + `openspec validate auth-rate-limiting --strict`
