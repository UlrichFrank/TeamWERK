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

### Requirement: users.role akzeptiert `presseteam`
Das System SHALL im `users.role`-CHECK-Constraint die Werte `admin`, `standard` und `presseteam` akzeptieren. Die Rolle ist hierarchisch: `admin ⊇ presseteam ⊇ standard`. Ein Presseteam-User kann alles, was ein Standard-User kann, plus die auf Presseteam eingeschränkten Match-Report-Aktionen. Ein Admin kann alles.

#### Scenario: Migration akzeptiert neuen Wert
- **WHEN** `INSERT INTO users (…, role) VALUES (…, 'presseteam')` ausgeführt wird
- **THEN** akzeptiert die Datenbank die Zeile

#### Scenario: Alter Wert weiterhin gültig
- **WHEN** eine Zeile mit `role='standard'` oder `role='admin'` besteht
- **THEN** bleibt sie unverändert und funktionsfähig

#### Scenario: Unzulässiger Wert
- **WHEN** `INSERT INTO users (…, role) VALUES (…, 'foo')` ausgeführt wird
- **THEN** lehnt der CHECK-Constraint mit Fehler ab

### Requirement: RequireRole akzeptiert Rollen-Liste
Das System SHALL die Middleware `auth.RequireRole(rollen...)` mit variabler Anzahl Rollen-Argumente erlauben. Ein Request mit `role IN rollen` läuft durch. Rolle `admin` fällt hierarchisch überall durch, wenn die Guard-Signatur `RequireRole("presseteam","admin")` lautet — Admin ist immer eine explizit erlaubte Alternative.

#### Scenario: Presseteam-User an Presseteam-Guard
- **WHEN** ein User mit `role='presseteam'` eine Route hinter `RequireRole("presseteam","admin")` aufruft
- **THEN** wird der Request durchgelassen

#### Scenario: Admin an Presseteam-Guard
- **WHEN** ein User mit `role='admin'` eine Route hinter `RequireRole("presseteam","admin")` aufruft
- **THEN** wird der Request durchgelassen

#### Scenario: Standard-User an Presseteam-Guard
- **WHEN** ein User mit `role='standard'` eine Route hinter `RequireRole("presseteam","admin")` aufruft
- **THEN** liefert das System HTTP 403

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

