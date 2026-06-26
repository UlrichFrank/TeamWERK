## 1. Go-Header-Middleware

- [ ] 1.1 `securityHeaders`-Middleware in `internal/app/` anlegen: setzt `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `Content-Security-Policy`
- [ ] 1.2 CSP-Wert: `default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' https://fonts.googleapis.com; font-src https://fonts.gstatic.com`
- [ ] 1.3 Middleware in die globale Kette in `internal/app/router.go` einhängen (vor den Routengruppen)
- [ ] 1.4 HSTS über Config-Flag (`internal/config`) steuerbar; Default aus; bei aktiv `Strict-Transport-Security: max-age=63072000; includeSubDomains`

## 2. CSP gegen realen Build verifizieren

- [ ] 2.1 Frontend laden (Vite-Assets, Service Worker, SSE, Fonts) und CSP-Verstöße in der Browser-Konsole prüfen; CSP bei Bedarf minimal nachjustieren (nur tatsächlich genutzte Quellen)

## 3. nginx als zweite Schicht

- [ ] 3.1 Dieselben Header als `add_header ... always;` in `deploy/nginx-intern.conf` ergänzen (HSTS dort ebenfalls erst nach Live-Cert)

## 4. Tests & Verifikation

- [ ] 4.1 Middleware-Test: Beispielantwort enthält alle Header inkl. `frame-ancestors 'none'`
- [ ] 4.2 Test: HSTS fehlt bei deaktiviertem Flag, vorhanden bei aktiviertem
- [ ] 4.3 `/verify-change` + `openspec validate security-response-headers --strict`
