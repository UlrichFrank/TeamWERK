## MODIFIED Requirements

### Requirement: HSTS erst nach TLS-Aufschaltung

Im Produktivbetrieb MUST der Server `Strict-Transport-Security: max-age=63072000; includeSubDomains` senden. Der Deploy-Prozess MUST `HSTS_ENABLED=true` in der Server-Umgebung idempotent setzen. Lokale Entwicklung ohne TLS bleibt ohne HSTS.

#### Scenario: HSTS auf Prod
- **WHEN** ein Request die produktive Instanz erreicht
- **THEN** enthält die Antwort den HSTS-Header mit zwei Jahren Gültigkeit

#### Scenario: Kein HSTS lokal
- **WHEN** `HSTS_ENABLED` nicht gesetzt ist
- **THEN** fehlt der Header
