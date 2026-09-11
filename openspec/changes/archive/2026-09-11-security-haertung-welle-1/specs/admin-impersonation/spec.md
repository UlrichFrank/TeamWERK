## ADDED Requirements

### Requirement: Impersonation ist auditiert und im Token erkennbar

Jede Impersonation MUST eine Audit-Spur hinterlassen: einen strukturierten Log-Eintrag mit Akteur und Ziel sowie einen Eintrag im Event-Log des ausführenden Admins (Kategorie `admin`). Das ausgestellte Access-Token MUST den Claim `impersonated_by` mit der ID des Admins tragen; reguläre Logins tragen diesen Claim nicht.

#### Scenario: Impersonation erzeugt Audit-Eintrag
- **WHEN** ein Admin `POST /api/admin/impersonate/{id}` aufruft
- **THEN** existiert danach ein `user_events`-Eintrag für den Admin mit Kategorie `admin` und dem Namen des Ziels

#### Scenario: Token ist unterscheidbar
- **WHEN** das Impersonations-Token geparst wird
- **THEN** enthält es `impersonated_by` mit der Admin-ID; ein per Login ausgestelltes Token enthält den Claim nicht
