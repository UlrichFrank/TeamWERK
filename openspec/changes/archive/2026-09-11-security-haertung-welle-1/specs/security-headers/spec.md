## MODIFIED Requirements

### Requirement: HSTS erst nach TLS-Aufschaltung

Das System SHALL `Strict-Transport-Security` NUR dann senden, wenn TLS/Live-Zertifikat aktiv ist; die Aktivierung SHALL über Konfiguration (`HSTS_ENABLED`) steuerbar sein und im Standard (vor Live-Cert) deaktiviert bleiben. Im Produktivbetrieb MUST der Header `max-age=63072000; includeSubDomains` gesendet werden; der Deploy-Prozess MUST `HSTS_ENABLED=true` in der Server-Umgebung idempotent setzen.

#### Scenario: HSTS deaktiviert vor Live-Zertifikat
- **WHEN** die HSTS-Konfiguration deaktiviert ist
- **THEN** enthält die Antwort keinen `Strict-Transport-Security`-Header

#### Scenario: HSTS aktiv nach Aufschaltung
- **WHEN** die HSTS-Konfiguration aktiviert ist
- **THEN** enthält die Antwort `Strict-Transport-Security` mit `max-age=63072000; includeSubDomains`

#### Scenario: Deploy setzt HSTS
- **WHEN** `make deploy` auf einen Bestandsserver ohne `HSTS_ENABLED` läuft
- **THEN** steht danach `HSTS_ENABLED=true` in der Server-Umgebung
