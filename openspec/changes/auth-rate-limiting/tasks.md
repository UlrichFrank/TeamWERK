## 1. Konfiguration & Abhängigkeit

- [ ] 1.1 `go-chi/httprate` als Abhängigkeit hinzufügen (`go get`, in `go.mod`)
- [ ] 1.2 Config-Parameter in `internal/config` ergänzen: Auth-Limit (Versuche/Fenster), Lockout-Schwelle, Lockout-Dauer; sinnvolle Defaults; in Tests deaktivierbar

## 2. Migration

- [ ] 2.1 `internal/db/migrations/010_user_login_throttle.up.sql`: `ALTER TABLE users ADD COLUMN failed_login_count INTEGER NOT NULL DEFAULT 0; ADD COLUMN locked_until TEXT;`
- [ ] 2.2 Passende `010_user_login_throttle.down.sql`

## 3. IP-Rate-Limiting

- [ ] 3.1 `httprate`-Middleware nur auf die Public-Auth-Routengruppe in `internal/app/router.go` setzen (Schlüssel = Client-IP via `RealIP`)
- [ ] 3.2 Sicherstellen, dass 429 VOR bcrypt/Mailversand greift; `Retry-After` setzen
- [ ] 3.3 `RealIP`-Quelle korrekt an nginx `X-Forwarded-For` koppeln

## 4. Account-Lockout

- [ ] 4.1 Login-Handler (`internal/auth/handler.go`): bei gesperrtem Konto (`locked_until` in der Zukunft) ohne bcrypt mit Drosselungsantwort reagieren
- [ ] 4.2 Fehlversuch erhöht `failed_login_count`; bei Schwellenüberschreitung `locked_until` (exponentielles Backoff) setzen
- [ ] 4.3 Erfolgreicher Login setzt `failed_login_count=0`, `locked_until=NULL`
- [ ] 4.4 Generisches Antwortverhalten beibehalten (keine Enumeration: gesperrtes existierendes Konto ununterscheidbar von gedrosselter nicht-existenter E-Mail)

## 5. forgot-password-Drosselung

- [ ] 5.1 Zusätzliche Drosselung pro Ziel-E-Mail für `forgot-password`, ohne Existenz preiszugeben

## 6. nginx (optionale zweite Schicht)

- [ ] 6.1 `limit_req_zone` + `limit_req` für `location ~ ^/api/auth/(login|refresh|forgot-password|reset-password)` in `deploy/nginx-intern.conf`

## 7. Tests & Verifikation

- [ ] 7.1 429 nach Limitüberschreitung; innerhalb Limit kein 429
- [ ] 7.2 Lockout nach N Fehlversuchen; Reset bei Erfolg; gesperrtes Konto antwortet ohne bcrypt
- [ ] 7.3 `forgot-password`: gedrosselte Anfrage versendet keine Mail
- [ ] 7.4 Bestehende Auth-Happy-Path-Tests laufen mit hochgesetztem/abgeschaltetem Limit grün
- [ ] 7.5 `/verify-change` + `openspec validate auth-rate-limiting --strict`
