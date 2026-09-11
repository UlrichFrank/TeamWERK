# auth Specification

## Purpose

Diese Spezifikation beschreibt die Capability `auth`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Role-based access control
The system SHALL enforce access based on the user's system role and club functions embedded in the JWT claims.

System roles are persisted in `users.role` and accept exactly two values: `admin` (full platform access, bypasses all `RequireClubFunction` checks) and `standard` (default; all access decisions delegated to club functions and ownership). The endpoint `PUT /api/admin/users/{id}/role` MUST reject any other value with HTTP 400.

Club functions (`spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`, `kassierer`, `sportliche_leitung`) and parent status (`is_parent`) are additional JWT claims that gate domain-specific features, not system access. Club functions and `is_parent` are NEVER stored as `users.role` values.

#### Scenario: Admin accesses admin-only route
- **WHEN** an `admin` user calls an admin-protected endpoint
- **THEN** the system processes the request normally

#### Scenario: Standard user accesses admin-only route
- **WHEN** a `standard` user calls an admin-protected endpoint
- **THEN** the system returns HTTP 403

#### Scenario: Trainer-function user accesses trainer-gated feature
- **WHEN** a `standard` user whose JWT contains `club_functions: ["trainer"]` calls a trainer-gated endpoint
- **THEN** the system processes the request normally

#### Scenario: User without trainer function accesses trainer-gated feature
- **WHEN** a `standard` user whose JWT does not contain `trainer` in `club_functions` calls a trainer-gated endpoint
- **THEN** the system returns HTTP 403

#### Scenario: Role or function change requires re-login
- **WHEN** an admin changes a user's system role or a member's club functions
- **THEN** the change takes effect only after the affected user's next login or token refresh (existing JWT claims are not updated mid-session)

#### Scenario: UpdateUserRole rejects legacy role names
- **WHEN** an admin sends `PUT /api/admin/users/{id}/role` with body `{"role":"trainer"}` (or `"vorstand"`, `"spieler"`, `"elternteil"`, `"sportliche_leitung"`)
- **THEN** the system returns HTTP 400 with body `"invalid role"` and does not modify `users.role`

#### Scenario: UpdateUserRole accepts standard role
- **WHEN** an admin sends `PUT /api/admin/users/{id}/role` with body `{"role":"standard"}`
- **THEN** the system updates `users.role` to `standard` and returns HTTP 204

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

### Requirement: users.role akzeptiert ausschließlich `admin` und `standard`
Das System SHALL im `users.role`- und im `invitation_tokens.role`-CHECK-Constraint nur die
Werte `admin` und `standard` akzeptieren. Die Rolle bleibt hierarchisch: `admin ⊇ standard`.
Bestandszeilen mit `role='presseteam'` SHALL die Migration auf `standard` überführen — die
Rolle erlaubte nichts, was ein Standard-Nutzer nicht künftig ebenfalls darf.

`PUT /api/users/{id}/role` und die Einladung (`POST /api/invitations`) SHALL jeden anderen
Wert — `presseteam` eingeschlossen — mit HTTP 400 `"invalid role"` ablehnen, ohne die Rolle
zu ändern.

#### Scenario: Bestands-Presseteam wird Standard
- **WHEN** die Migration auf einer Datenbank mit einer Zeile `users.role='presseteam'` läuft
- **THEN** trägt die Zeile danach `role='standard'`
- **AND** enthält weder `users` noch `invitation_tokens` danach eine Zeile mit `role='presseteam'`

#### Scenario: presseteam wird vom CHECK abgelehnt
- **WHEN** `INSERT INTO users (…, role) VALUES (…, 'presseteam')` ausgeführt wird
- **THEN** lehnt der CHECK-Constraint mit Fehler ab

#### Scenario: Rollenvergabe lehnt presseteam ab
- **WHEN** ein Admin `PUT /api/users/{id}/role` mit Body `{"role":"presseteam"}` sendet
- **THEN** liefert das System HTTP 400 mit Body `"invalid role"` und ändert `users.role` nicht

#### Scenario: Bestehende Werte bleiben gültig
- **WHEN** eine Zeile mit `role='standard'` oder `role='admin'` besteht
- **THEN** bleibt sie unverändert und funktionsfähig

### Requirement: RequireRole gated ohne impliziten Admin-Bypass
Das System SHALL die Middleware `auth.RequireRole(rollen...)` mit variabler Anzahl
Rollen-Argumente erlauben. Ein Request mit `role IN rollen` läuft durch. Es gibt **keinen**
impliziten Admin-Bypass: `admin` kommt nur durch, wo die Guard-Signatur ihn explizit
aufführt — bei einem reinen `RequireRole("admin")` ist er die einzige erlaubte Rolle.

#### Scenario: Admin an Admin-Guard
- **WHEN** ein User mit `role='admin'` eine Route hinter `RequireRole("admin")` aufruft
- **THEN** wird der Request durchgelassen

#### Scenario: Standard-User an Admin-Guard
- **WHEN** ein User mit `role='standard'` eine Route hinter `RequireRole("admin")` aufruft
- **THEN** liefert das System HTTP 403

### Requirement: JWT-Secret hat eine Mindeststärke

Der Server MUST den Start verweigern, wenn `JWT_SECRET` kürzer als 32 Byte ist oder dem Beispielwert der Konfigurationsvorlage entspricht. Die Fehlermeldung MUST den Grund und einen Weg zur Erzeugung eines geeigneten Werts nennen.

#### Scenario: Kurzes Secret
- **WHEN** der Server mit einem `JWT_SECRET` von 16 Byte startet
- **THEN** bricht er mit einer Fehlermeldung ab

#### Scenario: Beispielwert
- **WHEN** der Server mit dem Wert aus der Vorlage startet
- **THEN** bricht er mit einer Fehlermeldung ab

