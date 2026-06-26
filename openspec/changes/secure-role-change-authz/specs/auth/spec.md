## ADDED Requirements

### Requirement: Rollenänderung gegen Admin-Degradierung und Selbständerung geschützt

Das System SHALL `PUT /api/users/{id}/role` so absichern, dass ein Aufrufer ohne System-Rolle `admin`:
1. einen Account mit aktueller Rolle `admin` NICHT herabstufen kann, und
2. die eigene Rolle NICHT ändern kann.

In beiden Fällen SHALL der Server mit HTTP 403 antworten, ohne die Rolle zu ändern. Das Vergeben der Rolle `admin` SHALL weiterhin ausschließlich Aufrufern mit System-Rolle `admin` möglich sein (bestehendes Verhalten, unverändert).

#### Scenario: Vorstand darf einen Admin nicht herabstufen
- **WHEN** ein Aufrufer mit Vereinsfunktion `vorstand` (System-Rolle `standard`) `PUT /api/users/{adminId}/role` mit `{"role":"standard"}` für einen Account aufruft, dessen aktuelle Rolle `admin` ist
- **THEN** antwortet der Server mit HTTP 403 und die Rolle des Ziel-Accounts bleibt `admin`

#### Scenario: Selbst-Rollenänderung ist untersagt
- **WHEN** ein Aufrufer ohne System-Rolle `admin` `PUT /api/users/{id}/role` für die eigene User-ID aufruft
- **THEN** antwortet der Server mit HTTP 403 und die eigene Rolle bleibt unverändert

#### Scenario: Admin darf Rollen weiterhin verwalten
- **WHEN** ein Aufrufer mit System-Rolle `admin` `PUT /api/users/{id}/role` aufruft
- **THEN** wird die Rolle gemäß bestehender Validierung gesetzt (kein zusätzlicher 403 durch diese Anforderung)

#### Scenario: Vergabe von admin bleibt admin-only
- **WHEN** ein Aufrufer ohne System-Rolle `admin` `PUT /api/users/{id}/role` mit `{"role":"admin"}` aufruft
- **THEN** antwortet der Server mit HTTP 403 (bestehendes Verhalten)
