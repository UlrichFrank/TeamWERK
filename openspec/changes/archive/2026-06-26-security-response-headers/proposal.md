## Why

Weder nginx (`deploy/nginx-intern.conf`) noch die Go-Middleware-Kette (`internal/app/router.go:79-86`) emittieren Browser-Härtungsheader. Es fehlen `Content-Security-Policy`, `X-Frame-Options`/`frame-ancestors`, `Strict-Transport-Security`, `Referrer-Policy` und ein **globales** `X-Content-Type-Options` (heute nur per-Handler in `internal/files/handler.go:771`). Dadurch ist die App per `<iframe>` einbettbar (Clickjacking gegen eingeloggte Vorstands-/Kassierer-Aktionen wie Beitragslauf bestätigen oder Member löschen) und es fehlt eine Defense-in-Depth-Schicht gegen künftiges XSS (Sicherheitsaudit 2026-06-26, **B-4**, kalibriert auf Low/Medium).

## What Changes

- **Go-Header-Middleware** in der globalen Kette (`internal/app/router.go`), damit der Schutz unabhängig von nginx greift: `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, eine restriktive `Content-Security-Policy` (`default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'`, dazu `connect-src 'self'`, `img-src 'self' data:`, `style-src 'self' https://fonts.googleapis.com`, `font-src https://fonts.gstatic.com` für die Hanken-Grotesk-Einbindung).
- **HSTS** wird vorbereitet, aber **erst nach** Live-Zertifikat/Domain aktiviert (laut `docs/agent/10-deployment.md` noch ausstehend) — solange als auskommentierte/konfigurationsgesteuerte Option, um SSL-Strip nach TLS-Aufschaltung zu schließen.
- **nginx** spiegelt dieselben Header als zweite Schicht (`add_header ... always;`).
- CSP wird zunächst gegen die reale Asset-/Font-/SSE-Nutzung verifiziert (kein Bruch von Vite-Assets, Service Worker, SSE-`connect-src`).

## Capabilities

### New Capabilities
- `security-headers`: Serverseitige HTTP-Sicherheitsheader auf allen Antworten (Clickjacking-Schutz, CSP-Defense-in-Depth, Referrer-/Sniffing-/Transport-Härtung).

### Modified Capabilities
<!-- keine -->

## Impact

- **Code:** neue Middleware in `internal/app/` (z.B. `securityHeaders`), eingehängt in die globale Kette in `router.go`; ggf. Config-Flag für HSTS.
- **Betrieb:** `deploy/nginx-intern.conf` (`add_header`-Block).
- **Risiko CSP:** Eine zu strenge CSP kann Frontend-Assets/SSE/Service Worker brechen — daher CSP gegen den realen Build verifizieren (Vite-Bundles same-origin, SSE `connect-src 'self'`, Fonts gewhitelistet).
- **Tests:** Middleware-Test, der das Vorhandensein der Header auf einer Beispielantwort prüft; HSTS nur bei aktiviertem Flag.
- **Daten/Migration:** keine.
