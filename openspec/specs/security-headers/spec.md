# security-headers Specification

## Purpose
TBD - created by archiving change security-response-headers. Update Purpose after archive.
## Requirements
### Requirement: Sicherheitsheader auf allen HTTP-Antworten

Das System SHALL auf allen HTTP-Antworten die folgenden Header setzen: `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin` und eine `Content-Security-Policy`, die mindestens `default-src 'self'`, `frame-ancestors 'none'`, `object-src 'none'` und `base-uri 'self'` enthält. Die Header SHALL serverseitig in der Go-Middleware-Kette gesetzt werden, damit sie unabhängig vom vorgelagerten Reverse-Proxy wirken.

#### Scenario: Header sind auf einer API-Antwort vorhanden
- **WHEN** ein beliebiger Endpoint eine HTTP-Antwort erzeugt
- **THEN** enthält die Antwort `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy` und eine `Content-Security-Policy` mit `frame-ancestors 'none'`

#### Scenario: Einbettung per iframe wird verhindert
- **WHEN** eine fremde Seite versucht, die Anwendung in einem `<iframe>` einzubetten
- **THEN** verhindert `X-Frame-Options: DENY` / `frame-ancestors 'none'` das Rendering

#### Scenario: CSP bricht die Anwendung nicht
- **WHEN** das gebaute Frontend (Vite-Assets, Service Worker, SSE-Verbindung, Hanken-Grotesk-Fonts) geladen wird
- **THEN** erlaubt die CSP diese same-origin- bzw. explizit gewhitelisteten Quellen, ohne legitime Ressourcen zu blockieren

---

### Requirement: HSTS erst nach TLS-Aufschaltung

Das System SHALL `Strict-Transport-Security` NUR dann senden, wenn TLS/Live-Zertifikat aktiv ist; die Aktivierung SHALL über Konfiguration steuerbar sein und im Standard (vor Live-Cert) deaktiviert bleiben, um Aussperrung bei noch fehlender gültiger Zertifikatskette zu vermeiden.

#### Scenario: HSTS deaktiviert vor Live-Zertifikat
- **WHEN** die HSTS-Konfiguration deaktiviert ist
- **THEN** enthält die Antwort keinen `Strict-Transport-Security`-Header

#### Scenario: HSTS aktiv nach Aufschaltung
- **WHEN** die HSTS-Konfiguration aktiviert ist
- **THEN** enthält die Antwort `Strict-Transport-Security` mit einer `max-age`-Direktive

